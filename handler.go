package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

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
