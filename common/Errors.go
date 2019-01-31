package common

import "errors"

var (
	ERR_LOCK_ALREADY_REQUIRED = errors.New("锁已经被占用")

	ERR_NO_LOCAL_IP_FOUND = errors.New("没有找到网卡")
)
