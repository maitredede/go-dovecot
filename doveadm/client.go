package doveadm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"go.uber.org/zap"
)

type Client struct {
	addr   string
	auth   Auth
	logger *zap.SugaredLogger
}

func NewClient(logger *zap.SugaredLogger, addr string, auth Auth) (*Client, error) {
	return &Client{
		addr:   addr,
		auth:   auth,
		logger: logger,
	}, nil
}

func (c *Client) ExecuteCommand(ctx context.Context, command string, parameters map[string]interface{}, tag string) (interface{}, error) {

	payload := []interface{}{
		[]interface{}{
			command,
			parameters,
			tag,
		},
	}

	bin, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("payload marshal failed: %w", err)
	}
	body := bytes.NewBuffer(bin)

	req, err := http.NewRequestWithContext(ctx, "POST", c.addr, body)
	if err != nil {
		return nil, fmt.Errorf("new request failed: %w", err)
	}
	req.Header.Add("content-type", "application/json")

	if err := c.auth.apply(req); err != nil {
		return nil, fmt.Errorf("authentication method failed: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request exec failed: %w", err)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	c.logger.Debugf("http response code=%v status=%v", resp.StatusCode, resp.Status)

	responseBin, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("response read failed: %w", err)
	}

	var data interface{}
	if err := json.Unmarshal(responseBin, &data); err != nil {
		return nil, fmt.Errorf("response unmarshal failed: %w", err)
	}

	return data, fmt.Errorf("TODO")
}

func (c *Client) DictGet(ctx context.Context, user string, dictUri string, key string) (interface{}, error) {

	params := make(map[string]interface{})
	params["user"] = user
	if len(dictUri) > 0 {
		params["dictUri"] = dictUri
	}
	params["key"] = key

	return c.ExecuteCommand(ctx, "dictGet", params, "tag1")
}
