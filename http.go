package main

import (
	"fmt"
	"net/http"
)

func http_get(url string) (*http.Response, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return resp, fmt.Errorf("Status Code: %d", resp.StatusCode)
	}

	return resp, err
}
