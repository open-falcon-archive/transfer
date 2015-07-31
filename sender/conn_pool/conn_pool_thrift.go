package conn_pool

import (
	"fmt"
	thrift "github.com/niean/thrift/lib/go/thrift"
	rrd "github.com/open-falcon/transfer/sender/rrd"
	spool "github.com/toolkits/pool/simple_conn_pool"
	"net"
	"sync"
	"time"
)

// ConnPools Manager
type ThriftConnPools struct {
	sync.RWMutex
	M           map[string]*spool.ConnPool
	MaxConns    int
	MaxIdle     int
	ConnTimeout int
	CallTimeout int
	Protocol    string
}

func NewThriftConnPools(maxConns, maxIdle, connTimeout, callTimeout int, cluster []string,
	protocol string) *ThriftConnPools {

	cp := &ThriftConnPools{M: make(map[string]*spool.ConnPool), MaxConns: maxConns, MaxIdle: maxIdle,
		ConnTimeout: connTimeout, CallTimeout: callTimeout, Protocol: protocol}

	ct := time.Duration(cp.ConnTimeout) * time.Millisecond
	for _, address := range cluster {
		if _, exist := cp.M[address]; exist {
			continue
		}
		cp.M[address] = createOneThriftPool(address, address, ct, maxConns, maxIdle, protocol)
	}

	return cp
}

// send
func (this *ThriftConnPools) Send(addr string, items []*rrd.GraphItem) error {
	connPool, exists := this.Get(addr)
	if !exists {
		return fmt.Errorf("Send, %s has no connection pool", addr)
	}

	conn, err := connPool.Fetch()
	if err != nil {
		return fmt.Errorf("Send, %s get connection fail: conn %v, err %v. proc: %s", addr, conn, err, connPool.Proc())
	}

	cli := conn.(RRDClient).cli
	callTimeout := time.Duration(this.CallTimeout) * time.Millisecond

	done := make(chan error)
	go func() {
		msg, err := cli.Send(items)
		if err == nil && msg == "OK" {
			done <- nil
		} else {
			done <- fmt.Errorf("%v, msg: %s", err, msg)
		}
	}()

	select {
	case <-time.After(callTimeout):
		connPool.ForceClose(conn)
		return fmt.Errorf("Send, %s timeout", addr)
	case err = <-done:
		if err != nil {
			connPool.ForceClose(conn)
			err = fmt.Errorf("Send, %s call failed, err %v. proc: %s", addr, err, connPool.Proc())
		} else {
			connPool.Release(conn)
		}
		return err
	}
}

func (this *ThriftConnPools) Ping(addr string) error {
	connPool, exists := this.Get(addr)
	if !exists {
		return fmt.Errorf("Ping, %s has no connection pool", addr)
	}

	conn, err := connPool.Fetch()
	if err != nil {
		return fmt.Errorf("Ping, %s get connection fail: conn %v, err %v. proc: %s", addr, conn, err, connPool.Proc())
	}

	cli := conn.(RRDClient).cli
	callTimeout := time.Duration(this.CallTimeout) * time.Millisecond

	done := make(chan error)
	go func() {
		err := cli.Ping()
		done <- err
	}()

	select {
	case <-time.After(callTimeout):
		connPool.ForceClose(conn)
		return fmt.Errorf("Ping, %s timeout", addr)
	case err = <-done:
		if err != nil {
			connPool.ForceClose(conn)
			err = fmt.Errorf("Ping, %s call failed, err %v. proc: %s", addr, err, connPool.Proc())
		} else {
			connPool.Release(conn)
		}
		return err
	}
}

// query
func (this *ThriftConnPools) Query(addr string, requests []*rrd.QueryRequest) (rdata []*rrd.QueryResponse, errt error) {
	connPool, exists := this.Get(addr)
	if !exists {
		errt = fmt.Errorf("Query, %s has no connection pool", addr)
		return
	}

	conn, err := connPool.Fetch()
	if err != nil {
		errt = fmt.Errorf("Query, %s get connection fail: conn %v, err %v. proc: %s", addr, conn, err, connPool.Proc())
		return
	}

	cli := conn.(RRDClient).cli
	callTimeout := time.Duration(this.CallTimeout) * time.Millisecond

	done := make(chan error)
	go func() {
		rdata, err = cli.Query(requests)
		done <- err
	}()

	select {
	case <-time.After(callTimeout):
		connPool.ForceClose(conn)
		errt = fmt.Errorf("Query, %s timeout", addr)
		return
	case err = <-done:
		if err != nil {
			connPool.ForceClose(conn)
			errt = fmt.Errorf("Query, %s call failed, err %v. proc: %s", addr, err, connPool.Proc())
		} else {
			connPool.Release(conn)
			errt = nil
		}
		return
	}
}

// last
func (this *ThriftConnPools) Last(addr string, requests []*rrd.LastRequest) (rdata []*rrd.LastResponse, errt error) {
	connPool, exists := this.Get(addr)
	if !exists {
		errt = fmt.Errorf("Last, %s has no connection pool", addr)
		return
	}

	conn, err := connPool.Fetch()
	if err != nil {
		errt = fmt.Errorf("Last, %s get connection fail: conn %v, err %v. proc: %s", addr, conn, err, connPool.Proc())
		return
	}

	cli := conn.(RRDClient).cli
	callTimeout := time.Duration(this.CallTimeout) * time.Millisecond

	done := make(chan error)
	go func() {
		rdata, err = cli.Last(requests)
		done <- err
	}()

	select {
	case <-time.After(callTimeout):
		connPool.ForceClose(conn)
		errt = fmt.Errorf("Last, %s timeout", addr)
		return
	case err = <-done:
		if err != nil {
			connPool.ForceClose(conn)
			errt = fmt.Errorf("Last, %s call failed, err %v. proc: %s", addr, err, connPool.Proc())
		} else {
			connPool.Release(conn)
			errt = nil
		}
		return
	}
}

// statistics
func (this *ThriftConnPools) ProcOne(addr string) string {
	proc, _ := this.M[addr]
	return proc.Proc()
}

func (this *ThriftConnPools) Proc() []string {
	procs := []string{}
	for _, cp := range this.M {
		procs = append(procs, cp.Proc())
	}
	return procs
}

func (this *ThriftConnPools) Get(address string) (*spool.ConnPool, bool) {
	this.RLock()
	defer this.RUnlock()
	p, exists := this.M[address]
	return p, exists
}

func (this *ThriftConnPools) Destroy() {
	this.Lock()
	defer this.Unlock()
	addresses := make([]string, 0, len(this.M))
	for address := range this.M {
		addresses = append(addresses, address)
	}

	for _, address := range addresses {
		this.M[address].Destroy()
		delete(this.M, address)
	}
}

func createOneThriftPool(name string, address string, connTimeout time.Duration, maxConns int, maxIdle int,
	protocol string) *spool.ConnPool {
	p := spool.NewConnPool(name, address, maxConns, maxIdle)
	p.New = func(connName string) (spool.NConn, error) {
		_, err := net.ResolveTCPAddr("tcp", p.Address)
		if err != nil {
			return nil, err
		}

		// protocol factory
		var protocolFactory thrift.TProtocolFactory
		switch protocol {
		case "compact":
			protocolFactory = thrift.NewTCompactProtocolFactory()
		case "simplejson":
			protocolFactory = thrift.NewTSimpleJSONProtocolFactory()
		case "json":
			protocolFactory = thrift.NewTJSONProtocolFactory()
		case "binary", "":
			protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()
		default:
			return nil, fmt.Errorf("invalid protocol %s", protocol)
		}

		// transport factory
		var transportFactory thrift.TTransportFactory
		transportFactory = thrift.NewTTransportFactory()

		// transport
		var transport thrift.TTransport
		//transport, err = thrift.NewTSocketTimeout(p.Address, connTimeout)
		transport, err = thrift.NewTSocket(p.Address)
		if err != nil {
			return nil, err
		}
		transport = transportFactory.GetTransport(transport)
		err = transport.Open()
		if err != nil {
			return nil, err
		}

		// client
		var client *rrd.RRDHBaseBackendClient
		client = rrd.NewRRDHBaseBackendClientFactory(transport, protocolFactory)

		return RRDClient{cli: client, name: connName}, nil
	}

	return p
}

// RRDClient
type RRDClient struct {
	cli  *rrd.RRDHBaseBackendClient
	name string
}

func (this RRDClient) Name() string {
	return this.name
}

func (this RRDClient) Closed() bool {
	return (this.cli == nil || this.cli.Transport.IsOpen())
}

func (this RRDClient) Close() error {
	if this.cli == nil {
		return nil
	}

	if this.cli.Transport.IsOpen() {
		err := this.cli.Transport.Close()
		this.cli = nil
		return err
	}

	this.cli = nil
	return nil
}
