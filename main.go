package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"os"
	"runtime"
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

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	_, err := tls.LoadX509KeyPair(certificatePath, keyPath)
	if err != nil {
		log.Fatalf("error in tls.LoadX509KeyPair: %s", err)
	}

	log.Printf("certificate: %s, key: %s, local server on: %s, backend server on: %s", certificatePath, keyPath, localAddress, backendAddress)

	mux := http.NewServeMux()
	mux.Handle("/", func(h http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := &statusWriter{ResponseWriter: w}
			le := &logEntry{log: logger, start: nowMillisecond(), r: r}
			defer le.Write()
			h.ServeHTTP(sw, r)
			le.statusCode = sw.status
			le.responseLength = sw.length
		})
	}(proxyHandler))

	http.ListenAndServeTLS(localAddress, certificatePath, keyPath, mux)
}
