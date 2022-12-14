// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package env

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// HTTP client time out.
	httpTimeOut = 5 * time.Second

	// Maximum number of ping the server to wait.
	maxAttempts = 30
)

// HTTPGet send GET
func HTTPGet(url string) (code int, respBody string, err error) {
	log.Println("HTTP GET", url)
	client := &http.Client{Timeout: httpTimeOut}
	resp, err := client.Get(url)
	if err != nil {
		log.Println(err)
		return 0, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return 0, "", err
	}
	respBody = string(body)
	code = resp.StatusCode
	return code, respBody, nil
}

// HTTPGet send GET
func HTTPTlsGet(url, rootdir string, port uint16) (code int, respBody string, err error) {
	certPool := x509.NewCertPool()
	bs, err := os.ReadFile(filepath.Join(rootdir, "testdata/certs/cert-chain.pem"))
	if err != nil {
		return 0, "", fmt.Errorf("failed to read client ca cert: %s", err)
	}
	ok := certPool.AppendCertsFromPEM(bs)
	if !ok {
		return 0, "", fmt.Errorf("failed to append client certs")
	}

	certificate, err := tls.LoadX509KeyPair(filepath.Join(rootdir, "testdata/certs/cert-chain.pem"),
		filepath.Join(rootdir, "testdata/certs/key.pem"))
	if err != nil {
		return 0, "", fmt.Errorf("failed to get certificate")
	}
	tlsConf := &tls.Config{Certificates: []tls.Certificate{certificate}, ServerName: "localhost", RootCAs: certPool}
	tr := &http.Transport{
		TLSClientConfig: tlsConf,
		//DialTLS: func(network, addr string) (net.Conn, error) {
		//	return tls.Dial("tcp", fmt.Sprintf("localhost:%d", port), tlsConf)
		//return tls.Dial(network, addr, tlsConf)
		//},
	}
	log.Println("HTTP TLS GET", url)
	client := &http.Client{Timeout: httpTimeOut, Transport: tr}
	resp, err := client.Get(url)
	log.Println("resp ", resp)
	if err != nil {
		log.Println(err)
		return 0, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return 0, "", err
	}
	respBody = string(body)
	code = resp.StatusCode
	return code, respBody, nil
}

// HTTPPost sends POST
func HTTPPost(url string, contentType string, reqBody string) (code int, respBody string, err error) {
	log.Println("HTTP POST", url)
	client := &http.Client{Timeout: httpTimeOut}
	resp, err := client.Post(url, contentType, strings.NewReader(reqBody))
	if err != nil {
		log.Println(err)
		return 0, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return 0, "", err
	}
	respBody = string(body)
	code = resp.StatusCode
	return code, respBody, nil
}

// ShortLiveHTTPPost send HTTP without keepalive
func ShortLiveHTTPPost(url string, contentType string, reqBody string) (code int, respBody string, err error) {
	log.Println("Short live HTTP POST", url)
	tr := &http.Transport{
		DisableKeepAlives: true,
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Post(url, contentType, strings.NewReader(reqBody))
	if err != nil {
		log.Println(err)
		return 0, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return 0, "", err
	}
	respBody = string(body)
	code = resp.StatusCode
	log.Println(respBody)
	return code, respBody, nil
}

// HTTPGetWithHeaders send HTTP with headers
func HTTPGetWithHeaders(l string, headers map[string]string) (code int, respBody string, err error) {
	log.Println("HTTP GET with headers: ", l)
	client := &http.Client{Timeout: httpTimeOut}
	req := http.Request{}

	req.Header = map[string][]string{}
	for k, v := range headers {
		req.Header[k] = []string{v}
	}
	req.Method = http.MethodGet
	req.URL, err = url.Parse(l)
	if err != nil {
		log.Println(err)
		return 0, "", err
	}

	resp, err := client.Do(&req)
	if err != nil {
		log.Println(err)
		return 0, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return 0, "", err
	}
	respBody = string(body)
	code = resp.StatusCode
	log.Println(respBody)
	return code, respBody, nil
}

// WaitForHTTPServer waits for a HTTP server
func WaitForHTTPServer(url string) error {
	return WaitForHTTPServerWithTLS(url, "", false, 0)
}

// WaitForHTTPServer waits for a HTTP server
func WaitForHTTPServerWithTLS(url, rootDir string, enableTLS bool, port uint16) error {
	for i := 0; i < maxAttempts; i++ {
		log.Println("Pinging URL: ", url)
		var err error
		var code int
		if enableTLS {
			code, _, err = HTTPTlsGet(url, rootDir, port)
		} else {
			code, _, err = HTTPGet(url)
		}
		if err == nil && code == http.StatusOK {
			log.Println("Server is up and running...")
			return nil
		}
		log.Println("Will wait 200ms and try again.")
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for server startup")
}

// WaitForPort waits for a TCP port
func WaitForPort(port uint16) {
	serverPort := fmt.Sprintf("localhost:%v", port)
	for i := 0; i < maxAttempts; i++ {
		log.Println("Pinging port: ", serverPort)
		_, err := net.Dial("tcp", serverPort)
		if err == nil {
			log.Println("The port is up and running...")
			return
		}
		log.Println("Wait 200ms and try again.")
		time.Sleep(200 * time.Millisecond)
	}
	log.Println("Give up the wait, continue the test...")
}

// IsPortUsed checks if a port is used
func IsPortUsed(port uint16) bool {
	serverPort := fmt.Sprintf("localhost:%v", port)
	_, err := net.DialTimeout("tcp", serverPort, 100*time.Millisecond)
	return err == nil
}
