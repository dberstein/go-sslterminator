package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	localAddress    string
	backendAddress  string
	certificatePath string
	keyPath         string
	logger          *log.Logger
)

func init() {
	logger = log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)

	flag.StringVar(&localAddress, "listen", ":443", "listen address")
	flag.StringVar(&certificatePath, "cert", "cert.pem", "SSL certificate path")
	flag.StringVar(&keyPath, "key", "key.pem", "SSL key path")
	flag.StringVar(&backendAddress, "backend", "http://127.0.0.1", "Backend URL")
}

func nowMillisecond() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

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
		strconv.Itoa(le.responseLength) + "b",
		"(" + strconv.FormatInt(nowMillisecond()-le.start, 10) + " μs)",
	}, " ")
}

func (le *logEntry) Write() {
	le.log.Println(le.string())
}

type statusWriter struct {
	http.ResponseWriter
	status int
	length int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	n, err := w.ResponseWriter.Write(b)
	w.length += n
	return n, err
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	url := backendAddress + r.RequestURI
	proxyReq, err := http.NewRequest(r.Method, url, bytes.NewReader(reqBody))
	proxyReq.Header = make(http.Header)
	for h, val := range r.Header {
		if h == "Origin" || h == "Host" {
			continue
		}
		proxyReq.Header[h] = val
	}

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for h, val := range resp.Header {
		for _, v := range val {
			w.Header().Set(h, v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	respBody, err := ioutil.ReadAll(resp.Body)
	w.Write(respBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func logRequest(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w}
		le := &logEntry{
			log:   logger,
			start: nowMillisecond(),
			r:     r,
		}
		defer le.Write()
		h.ServeHTTP(sw, r)
		le.statusCode = sw.status
		le.responseLength = sw.length
	})
}

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	_, err := tls.LoadX509KeyPair(certificatePath, keyPath)
	if err != nil {
		log.Fatalf("error in tls.LoadX509KeyPair: %s", err)
	}

	log.Printf("certificate: %s, key: %s, local server on: %s, backend server on: %s", certificatePath, keyPath, localAddress, backendAddress)

	mux := http.NewServeMux()
	mux.Handle("/", logRequest(proxyHandler))

	http.ListenAndServeTLS(localAddress, certificatePath, keyPath, mux)
}
