package worker

import (
	"context"
	"github.com/JacoobH/crontab/common"
	clientv3 "go.etcd.io/etcd/client/v3"
	"net"
	"time"
)

//
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

//注册到/cron/workers/ip，并且自动续约
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
		//1.生成租约
		if leaseGrantResp, err = register.lease.Grant(context.TODO(), 10); err != nil {
			goto ReTry
		}
		//生成上下文
		ctx, ctxFunc = context.WithCancel(context.TODO())
		//自动续租
		if leaseKeepRespChan, err = register.lease.KeepAlive(ctx, leaseGrantResp.ID); err != nil {
			goto ReTry
		}
		//注册到etcd
		if _, err = register.kv.Put(context.TODO(), registerKey, "", clientv3.WithLease(leaseGrantResp.ID)); err != nil {
			goto ReTry
		}

		for {
			select {
			case leaseKeep = <-leaseKeepRespChan:
				if leaseKeep == nil {
					//证明此时连接异常，需要重新连接
					goto ReTry
				}
			}
		}
	ReTry:
		//取消租约
		if ctxFunc != nil {
			//因为一开始的时候没有生成ctxFunc，所以给一个nil来判断是哪里出的错误
			//将租约取消并且过期
			ctxFunc()
			register.lease.Revoke(context.TODO(), leaseGrantResp.ID)
		}
		//如果失败了，就过一秒再试能否上线
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
