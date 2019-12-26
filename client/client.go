package client

import (
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-fastlog/fastlog"
	"gitlab.com/NebulousLabs/fastrand"

	"github.com/maoxs2/gxminer/worker"
)

type JsonRPC struct {
	JsonRPC string      `json:"jsonrpc"`
	Id      int         `json:"id"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

type PoolConfig struct {
	Pool  string `json:"pool"`
	User  string `json:"user"`
	Pass  string `json:"pass"`
	RigID string `json:"rig-id"`
	TLS   bool   `json:"tls"`
}

func GenRPCMessage(method string, params interface{}) JsonRPC {
	msg := JsonRPC{
		JsonRPC: "2.0",
		Id:      fastrand.Intn(1<<7 - 1),
		Method:  method,
		Params:  params,
	}

	return msg
}

type Client struct {
	conf   PoolConfig
	logger *fastlog.Logger
	Rx     *Rx

	currentConfigIndex int
	pConfigs           []PoolConfig

	mu sync.Mutex

	conn net.Conn

	dec *json.Decoder
	enc *json.Encoder

	RejectNum uint64
	AcceptNum uint64

	Job      worker.Job
	submitCh chan worker.Job
}

func NewClient(pConfigs []PoolConfig, wConf *worker.Config, logger *fastlog.Logger) *Client {

	submitCh := make(chan worker.Job)
	rx := NewRxClient(wConf, submitCh, logger)

	c := Client{
		Rx:       rx,
		submitCh: submitCh,
		logger:   logger,
	}

	c.conf = pConfigs[0]
	c.currentConfigIndex = 0
	c.pConfigs = pConfigs

	var err error
	var retryTimes = 0
Dial:
	if c.conf.TLS {
		logger.Infoln("Dialing to pool:", c.conf.Pool)
		c.conn, err = tls.Dial("tcp", c.conf.Pool, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			retryTimes++
			if retryTimes > 5 {
				retryTimes = 0
				c.failover()
			}
			if err == io.EOF {
				logger.Warnln("Failed to Login, check your username and password for pool")
			}
			c.logger.Warnln(err, "retrying...")
			time.Sleep(10 * time.Second)
			goto Dial
		}
	} else {
		logger.Infoln("Dialing to pool:", c.conf.Pool)
		c.conn, err = net.Dial("tcp", c.conf.Pool)
		if err != nil {
			retryTimes++
			if retryTimes == 5 {
				retryTimes = 0
				c.failover()
			}
			if err == io.EOF {
				logger.Warnln("Failed to Login, check your username and password for pool")
			}
			c.logger.Warnln(err, "retrying...")
			time.Sleep(10 * time.Second)
			goto Dial
		}
	}

	c.enc = json.NewEncoder(c.conn)
	c.dec = json.NewDecoder(c.conn)

	return &c
}

func (c *Client) send(msg interface{}) {
	var retryTimes = 0

Enc:
	err := c.enc.Encode(msg)
	if err != nil {
		c.logger.Errorln(err)
		retryTimes++
		if retryTimes > 5 {
			retryTimes = 0
			c.failover()
		}
		goto Enc
	}
}

// thread safe Reconnect
func (c *Client) Reconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	var err error
	var retryTimes = 0
Redial:
	if c.conf.TLS {
		c.conn, err = tls.Dial("tcp", c.conf.Pool, nil)
		if err != nil {
			retryTimes++
			if retryTimes > 5 {
				retryTimes = 0
				c.failover()
			}
			c.logger.Warnln(err, "retrying...")
			time.Sleep(10 * time.Second)
			goto Redial
		}
	} else {
		c.conn, err = net.Dial("tcp", c.conf.Pool)
		if err != nil {
			retryTimes++
			if retryTimes == 5 {
				retryTimes = 0
				c.failover()
			}
			c.logger.Warnln(err, "retrying...")
			time.Sleep(10 * time.Second)
			goto Redial
		}
	}

	c.dec = json.NewDecoder(c.conn)
	c.enc = json.NewEncoder(c.conn)

	c.SendLogin()
}

func (c *Client) failover() {
	if c.currentConfigIndex+1 < len(c.pConfigs) {
		c.currentConfigIndex++
		c.conf = c.pConfigs[c.currentConfigIndex]
	} else {
		c.currentConfigIndex = 0
		c.conf = c.pConfigs[c.currentConfigIndex]
	}
	c.logger.Warnln("Switched to the failover pool:", c.conf.Pool)
}

type LoginParams struct {
	Login string `json:"login"`
	Pass  string `json:"pass"`
	RigID string `json:"rigid"`
	Agent string `json:"agent"`
}

func (c *Client) SendLogin() {
	params := LoginParams{
		Login: c.conf.User,
		Pass:  c.conf.Pass,
		RigID: c.conf.RigID,
	}

	c.send(GenRPCMessage("login", params))
}

type SubmitParams struct {
	ID     string `json:"id"`
	JobID  string `json:"job_id"`
	Nonce  string `json:"nonce"`
	Result string `json:"result"`
}

func (c *Client) SendSubmit(id, jobID, nonce, result string) {
	params := SubmitParams{
		ID:     id,
		JobID:  jobID,
		Nonce:  nonce,
		Result: result,
	}

	c.send(GenRPCMessage("submit", params))
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		_ = c.conn.Close()
	}
}

func (c *Client) Listen(onLogin func(message JsonRPC)) {
	go func() {
		var retryTimes = 0

		for {
			var message JsonRPC
			err := c.dec.Decode(&message)
			if err != nil {
				if err == io.EOF {
					continue
				}

				if _, ok := err.(net.Error); ok {
					retryTimes++
					if retryTimes > 10 {
						retryTimes = 0
						c.failover()
						c.Reconnect()
					}
				}

				c.logger.Errorln(err)
			}

			if &message == nil {
				continue
			}

			go c.handleRecv(message, onLogin)
		}
	}()
}

func (c *Client) handleRecv(message JsonRPC, onLogin func(message JsonRPC)) {

	if message.Error != nil {
		if err, ok := message.Error.(map[string]interface{}); ok {
			if errMsg, ok := err["message"].(string); ok {

				if errMsg == "Unauthenticated" || errMsg == "IP Address currently banned" || errMsg == "your IP is banned" {
					if len(c.pConfigs) == 1 {
						c.logger.Warnln(errMsg, " trying to reload after 5 min")
						c.Close()
						c.Rx.StopWorkers()
						time.Sleep(5 * time.Minute)
						c.Reconnect()
					} else {
						c.logger.Warnln(errMsg, " trying to load failover pool")
						c.Close()
						c.Rx.StopWorkers()
						c.failover()
						c.Reconnect()
					}

				} else {
					c.logger.Warnln(errMsg)
				}

				c.RejectNum++
			}
		}
	}

	if message.Result != nil {
		results, ok := message.Result.(map[string]interface{})
		if ok && results["job"] != nil {

			if results["status"] != nil {
				if status, ok := results["status"].(string); ok && strings.ToLower(status) == "ok" {
					c.logger.Infoln("Successfully login")
				}
			}

			// register func here
			onLogin(message)

			jobInstance, _ := results["job"].(map[string]interface{})
			newJob := ParseJob(jobInstance)
			if rpcID, ok := results["id"].(string); ok {
				newJob.ID = rpcID
			}

			id, _ := strconv.Atoi(newJob.JobID)
			c.logger.Infoln("User:", c.pConfigs[c.currentConfigIndex].User)
			c.logger.Infoln("Pool:", c.pConfigs[c.currentConfigIndex].Pool)
			c.logger.Infoln("Init Job:", "ID:", fmt.Sprintf("%X", id), "Target:", fmt.Sprintf("%X", newJob.Target), "Seed:", strings.ToUpper(hex.EncodeToString(newJob.SeedHash)[0:16]))

			c.Job = newJob

			if c.Rx.Dataset == nil {
				c.Rx.ReadyInit(newJob.SeedHash)
				c.StartSubmitter(c.submitCh)
			}

			if c.Rx.Workers == nil {
				c.Rx.SpawnWorkers(newJob)
			} else {
				if len(newJob.NextSeedHash) > 0 {
					c.Rx.ReadyNext(newJob.NextSeedHash)
				}

				// assign new job
				if c.CheckUpdate(newJob) {
					// seed has changed, so have to Update the Dataset
					c.Rx.UpdateRxDataset(newJob.SeedHash)
				}
			}

			c.Rx.AssignNewJob(newJob)
		}

		if ok && results["status"] != nil && results["job"] == nil {
			if status, ok := results["status"].(string); ok && strings.ToLower(status) == "ok" {
				if strings.ToLower(status) == "ok" {
					c.AcceptNum++
					c.logger.Infoln("Sol is accepted")
				}
			}
		}

	}

	if message.Params != nil && message.Method == "job" {

		jobInstance, _ := message.Params.(map[string]interface{})
		newJob := ParseJob(jobInstance)

		c.Job = newJob

		id, _ := strconv.Atoi(newJob.JobID)
		c.logger.Println("New Job:", "ID:", fmt.Sprintf("%X", id), "Target:", fmt.Sprintf("%X", newJob.Target), "Seed:", strings.ToUpper(hex.EncodeToString(newJob.SeedHash)[0:16]))

		// assign new job
		if c.Rx.Workers != nil {
			needUpdateDataset := c.CheckUpdate(newJob)
			if needUpdateDataset {
				// seed has changed, so have to Update the Dataset
				c.Rx.UpdateRxDataset(newJob.SeedHash)
			}

			if len(newJob.NextSeedHash) > 0 {
				c.Rx.ReadyNext(newJob.NextSeedHash)
			}
		}

		c.Rx.AssignNewJob(newJob)
	}

}

func ParseJob(job map[string]interface{}) worker.Job {
	if job["seed_hash"] == nil {
		fastlog.Fatal("no seed_hash in Job, maybe the target Pool is not a random-x Pool")
	}
	seedHash, err := hex.DecodeString(job["seed_hash"].(string))
	if err != nil {
		fastlog.Errorln(err)
	}

	var nextSeedHash []byte
	if job["next_seed_hash"] != nil {
		nextSeedHash, err = hex.DecodeString(job["next_seed_hash"].(string))
		if err != nil {
			fastlog.Errorln(err)
		}
	}

	var target uint64
	strTarget, _ := job["target"].(string)
	b, _ := hex.DecodeString(strTarget)
	if len(strTarget) < 16 {
		target = uint64(binary.LittleEndian.Uint32(b))
		target = 0xFFFFFFFFFFFFFFFF / (0xFFFFFFFF / target)
	} else {
		target = binary.LittleEndian.Uint64(b)
	}

	blob, _ := hex.DecodeString(job["blob"].(string))

	id, ok := job["id"].(string)
	if !ok {
		id = "0"
	}

	return worker.Job{
		ID:           id,
		Blob:         blob,
		JobID:        job["job_id"].(string),
		Target:       target,
		SeedHash:     seedHash,
		NextSeedHash: nextSeedHash,

		Nonce: blob[39:43], // init
	}
}

// {"jsonrpc":"2.0","id":1,"method":"keepalived","params":{"job_id":"116187193926528", "nonce": "50521"}}
