package master

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/k8svip/crontab/common"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"time"
)

// 获取/cron/workers/下面的k、v,就结束了
type GroupMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

var (
	G_GroupMgr *GroupMgr
)

// 获取在线worker列表

func (groupMgr *GroupMgr) ListGruops() (groupArr []*common.GroupInfo, err error) {
	var (
		getResp   *clientv3.GetResponse
		kv        *mvccpb.KeyValue
		groupInfo *common.GroupInfo
		groupKey  string
	)
	// 初始化数组
	groupArr = make([]*common.GroupInfo, 0)

	// 获取目录下所有的kv
	if getResp, err = groupMgr.kv.Get(context.TODO(), common.JOB_WORKER_GROUP_DIR, clientv3.WithPrefix()); err != nil {
		return
	}

	//解析每个节点的IP
	for _, kv = range getResp.Kvs {
		groupInfo = &common.GroupInfo{}
		groupKey = common.ExtractWorkerGroup(string(kv.Key))
		if err = json.Unmarshal(kv.Value, &groupInfo);err != nil {
			err = nil
			continue
		}
		fmt.Println(groupKey, groupInfo, "--------")
		if groupKey == groupInfo.GroupName {
		groupArr = append(groupArr, groupInfo)
		}
	}


	fmt.Println(groupArr)

	return
}

func (groupMgr *GroupMgr) SaveGroupInfo(groupInfo *common.GroupInfo) (oldGroupInfo *common.GroupInfo, err error) {
	//把任务保存到/cron/jobs/任务名 --> json
	var (
		putResp         *clientv3.PutResponse
		groupKey        string
		groupInfoValue  []byte
		oldGroupInfoObj common.GroupInfo
	)

	// etcd的保存key
	groupKey = common.JOB_WORKER_GROUP_DIR + groupInfo.GroupName

	// 任务信息json
	if groupInfoValue, err = json.Marshal(groupInfo); err != nil {
		return
	}

	// 保存到etcd
	if putResp, err = groupMgr.kv.Put(context.TODO(), groupKey, string(groupInfoValue), clientv3.WithPrevKV()); err != nil {
		fmt.Println("出错了")
		return
	}

	//如果是更新，那么就返回旧值
	if putResp.PrevKv != nil {
		// 对旧值进行反序列化
		if err = json.Unmarshal(putResp.PrevKv.Value, &oldGroupInfoObj); err != nil {
			err = nil
			return
		}
		oldGroupInfo = &oldGroupInfoObj
	}

	return
}

func InitGroupMgr() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
	)

	config = clientv3.Config{
		Endpoints:   G_config.EtcdEndpoints,
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond,
	}

	// 建立连接
	if client, err = clientv3.New(config); err != nil {
		return
	}

	// 等到KV和Lease的API子集
	kv = clientv3.NewKV(client)

	G_GroupMgr = &GroupMgr{
		client: client,
		kv:     kv,
	}
	return
}
