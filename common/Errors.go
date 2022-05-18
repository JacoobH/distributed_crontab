package common

import "errors"

var (
	ERR_CLOCK_ALREADY_REQUIRED = errors.New("the lock is occupied")
)
