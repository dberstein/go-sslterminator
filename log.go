package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
)

type logEntry struct {
	log            *log.Logger
	start          int64
	statusCode     int
	r              *http.Request
	responseLength int
}

func (le *logEntry) string() string {
	return strings.Join([]string{
		strconv.Itoa(le.statusCode),
		le.r.Method,
		"\"" + le.r.URL.Path + "\"",
		strconv.Itoa(le.responseLength),
		"(" + strconv.FormatInt(nowMillisecond()-le.start, 10) + " μs)",
	}, " ")
}

func (le *logEntry) Write() {
	le.log.Println(le.string())
}
