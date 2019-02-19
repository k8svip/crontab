package worker

import (
	"context"
	"github.com/k8svip/crontab/common"
	"go.etcd.io/etcd/clientv3"
	"strings"
	"time"
)

type CheckGroupMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
}

var (
	G_checkGroupMgr *CheckGroupMgr
)

func (checkGroupMgr *CheckGroupMgr) BaseGroupCheckLoclIP(group string, ip string) (result bool) {
	var (
		checkKey  string
		getResp   *clientv3.GetResponse
		err       error
		groupInfo *common.GroupInfo
		iplist []string
		groupIp string
	)
	checkKey = common.JOB_WORKER_GROUP_DIR + group
	if getResp, err = checkGroupMgr.kv.Get(context.TODO(), checkKey); err != nil {
		return
	}

	if getResp.Count == 0 {
		result = false
		return
	}

	// 反序列化josn
	if groupInfo, err = common.UnpackGroupInfo(getResp.Kvs[0].Value); err != nil {
		return
	}

	iplist = strings.Split(groupInfo.GroupIps,",")
	for _, groupIp = range(iplist){
		if groupIp == ip {
			result = true
			return result
		}
	}

	return
}

func InitCheckGroupMgr() (err error) {

	var (
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
	)
	config = clientv3.Config{
		Endpoints:   G_config.EtcdEndpoints,
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond,
	}

	if client, err = clientv3.New(config); err != nil {
		return
	}

	kv = clientv3.NewKV(client)

	G_checkGroupMgr = &CheckGroupMgr{
		client: client,
		kv:     kv,
	}

	return
}
