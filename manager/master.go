package manager

import (
	"encoding/json"
	"net"

	"github.com/go-fastlog/fastlog"

	"github.com/maoxs2/gxminer/client"
)

type servant struct {
	udpConn *net.UDPConn

	enc *json.Encoder
	dec *json.Decoder
}

func newServant(conn *net.UDPConn) *servant {
	return &servant{
		udpConn: conn,

		enc: json.NewEncoder(conn),
		dec: json.NewDecoder(conn),
	}
}

type Master struct {
	conf   *MasterConfig
	exitCh chan struct{}

	servants []*servant
}

func NewMaster(conf *MasterConfig) *Master {
	exitCh := make(chan struct{})

	return &Master{
		conf:   conf,
		exitCh: exitCh,

		servants: nil,
	}
}

type MasterConfig struct {
	Enable bool
	Host   string
	Proxy  bool
	Key    string
}

func (m *Master) serveUDP(addr string) {
	l, err := net.Listen("udp", addr)
	if err != nil {
		fastlog.Info(err)
	}
	for {
		if conn, err := l.Accept(); err != nil {
			m.handleUDPConn(conn.(*net.UDPConn))
		}
	}
}

func (m *Master) handleUDPConn(conn *net.UDPConn) {
	s := newServant(conn)
	m.servants = append(m.servants, s)
	m.udpReceiver(s)
}

type MinerStatus struct {
	RigID  string
	Pool   client.PoolConfig
	Shares [2]uint64
	Log    string
}

func (m *Master) udpReceiver(s *servant) {
	go func() {
		for {
			select {
			case <-m.exitCh:
				return
			default:
				var ms MinerStatus
				err := s.dec.Decode(&ms)
				fastlog.Errorln(err)
				// deal with status package
			}
		}
	}()
}

func (m *Master) updatePool() {

}

func (m *Master) updateRigID() {

}

// as the proxy
//func (m *Master) serveTCP(addr string) {
//	l, err := net.Listen("udp", addr)
//	if err != nil {
//		fastlog.Info(err)
//	}
//	for {
//		if conn, err := l.Accept(); err != nil {
//			m.handleTCPConn(conn.(*net.TCPConn))
//		}
//	}
//}
//
//func (m *Master) handleTCPConn(conn *net.TCPConn) {
//
//}
