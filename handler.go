package main

import (
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func init() {
	// Disable HTTP/2
	http.DefaultClient.Transport = &http.Transport{
		TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper), // disable HTTP/2
	}
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	backURL := backendAddress + r.RequestURI

	proxyReq, err := http.NewRequest(r.Method, backURL, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	proxyReq.TransferEncoding = []string{"identity"}
	proxyReq.Close = true
	proxyReq.Header = make(http.Header)

	// copy headers
	for h, val := range r.Header {
		if h == "Origin" || h == "Host" {
			continue
		}
		proxyReq.Header[h] = val
	}

	// http client that does not follow redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// call backend
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	// remove host:port from redirects
	if loc, ok := resp.Header["Location"]; ok {
		location := make([]string, len(loc))
		for i, l := range loc {
			u, err := url.Parse(l)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			log.Printf("%s %s %+v", u.Host, backendAddress, strings.Contains(backendAddress, u.Host))
			if strings.Contains(backendAddress, u.Host) {
				location[i] = u.RequestURI()
			} else {
				location[i] = l
			}
		}
		resp.Header["Location"] = location
	}

	defer resp.Body.Close()

	// copy response headers
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
