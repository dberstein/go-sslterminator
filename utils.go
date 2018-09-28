package main

import "time"

func nowMillisecond() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
