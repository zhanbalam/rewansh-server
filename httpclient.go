package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const ResponseReadLimit = 1024

type httpClient struct {
	http.Client
}

func newHTTPClient(t time.Duration) *httpClient {
	return &httpClient{http.Client{
		Timeout: t,
	}}
}

func (c *httpClient) curl(ctx context.Context, h httpHost) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.Url, bytes.NewBufferString(h.RequestData))
	if err != nil {
		return err
	}
	res, err := c.Do(req)
	if err != nil {
		log.Printf("%s %s: error: %s", req.Method, req.URL, err)
		return err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(io.LimitReader(res.Body, ResponseReadLimit))
	if err != nil {
		log.Printf("Read body %s: error: %s", req.URL, err)
		return err
	}
	_, _ = io.Copy(ioutil.Discard, res.Body) // Drain body for close and reuse

	if len(h.RequestData) > 0 {
		if diff := bytes.Compare(body, []byte(h.ResponseData)); diff != 0 {
			return fmt.Errorf("Unexpected response data: %s", string(body))
		}
	}

	log.Printf("%s %s: %s", req.Method, req.URL, res.Status)

	return nil
}
