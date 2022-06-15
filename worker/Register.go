package worker

import (
	"context"
	"github.com/JacoobH/crontab/common"
	clientv3 "go.etcd.io/etcd/client/v3"
	"net"
	"time"
)

// Register Service registration
type Register struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease

	localIP string
}

var (
	G_register *Register
)

func getLocalIP() (ipv4 string, err error) {
	var (
		addrs   []net.Addr
		addr    net.Addr
		ipNet   *net.IPNet
		isIpNet bool
	)
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return
	}
	for _, addr = range addrs {
		if ipNet, isIpNet = addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.IP.String()
				return
			}
		}
	}
	err = common.ERR_NO_LOCAL_IP_FOUND
	return
}

// Register with /cron/workers/ip and renew automatically
func (register *Register) keepOnline() {
	var (
		registerKey       string
		leaseGrantResp    *clientv3.LeaseGrantResponse
		err               error
		ctx               context.Context
		ctxFunc           context.CancelFunc
		leaseKeepRespChan <-chan *clientv3.LeaseKeepAliveResponse
		leaseKeep         *clientv3.LeaseKeepAliveResponse
	)
	registerKey = common.JOB_WORKER_DIR + register.localIP

	for {
		ctxFunc = nil
		//1.Generate the lease
		if leaseGrantResp, err = register.lease.Grant(context.TODO(), 10); err != nil {
			goto ReTry
		}
		//Generation context
		ctx, ctxFunc = context.WithCancel(context.TODO())
		//Automatic renewal
		if leaseKeepRespChan, err = register.lease.KeepAlive(ctx, leaseGrantResp.ID); err != nil {
			goto ReTry
		}
		//Registered to etcd
		if _, err = register.kv.Put(context.TODO(), registerKey, "", clientv3.WithLease(leaseGrantResp.ID)); err != nil {
			goto ReTry
		}

		for {
			select {
			case leaseKeep = <-leaseKeepRespChan:
				if leaseKeep == nil {
					//The connection is abnormal and needs to be re-connected
					goto ReTry
				}
			}
		}
	ReTry:
		//Cancel the lease
		if ctxFunc != nil {
			//Because we didn't generate ctxFunc in the first place, so we give nil to see what went wrong
			//Cancel and expire the lease
			ctxFunc()
			register.lease.Revoke(context.TODO(), leaseGrantResp.ID)
		}
		//If it fails, try again a second later
		time.Sleep(1 * time.Second)
	}

}

func InitRegister() (err error) {
	var (
		config  clientv3.Config
		client  *clientv3.Client
		kv      clientv3.KV
		lease   clientv3.Lease
		localIp string
	)
	config = clientv3.Config{
		Endpoints:   G_config.EtcdEndPoints,                                     //cluster network address
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond, // timeout
	}

	//Establish a client
	if client, err = clientv3.New(config); err != nil {
		return
	}

	//Use to read or write KV of etcd
	kv = clientv3.NewKV(client)

	//Apply for a lease
	lease = clientv3.NewLease(client)

	if localIp, err = getLocalIP(); err != nil {
		return
	}

	G_register = &Register{
		client:  client,
		kv:      kv,
		lease:   lease,
		localIP: localIp,
	}

	go G_register.keepOnline()
	return
}
