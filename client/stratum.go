package client

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/maoxs2/gxminer/worker"
)

func (c *Client) CheckUpdate(newJob worker.Job) (needUpdate bool) {
	if c.Job.JobID == "" {
		return false
	}

	return bytes.Compare(newJob.SeedHash, c.Job.SeedHash) != 0
}

func (c *Client) StartSubmitter(submitCh chan worker.Job) {
	go func() {
		for {
			select {
			case wj := <-submitCh:
				id, _ := strconv.ParseUint(wj.JobID, 10, 64)
				c.logger.Infoln("Submitting Sol:", strings.ToUpper(hex.EncodeToString(wj.Nonce)), "for Job", fmt.Sprintf("%X", id), "Result:", fmt.Sprintf("%X", binary.LittleEndian.Uint64(wj.Result[24:32])))
				c.SendSubmit(wj.ID, wj.JobID, hex.EncodeToString(wj.Nonce), hex.EncodeToString(wj.Result))
			}
		}
	}()
}
