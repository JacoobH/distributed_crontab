package common

import "errors"

var (
	ERR_CLOCK_ALREADY_REQUIRED = errors.New("the lock is occupied")
	ERR_NO_LOCAL_IP_FOUND      = errors.New("No network card found")
)
