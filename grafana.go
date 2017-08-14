package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var grafanaURL string

func addGrafanaFlags(cmd *cobra.Command) {
	f := cmd.PersistentFlags()
	f.StringVarP(&grafanaURL, "grafana-url", "", "http://localhost:3000", "grafana url")
}

func getGrafanaURL() *url.URL {
	u, err := url.Parse(grafanaURL)
	if err != nil {
		errorLog("unable to parse grafana url: %s", err)
	}

	return u
}

// inspired by https://github.com/coreos/prometheus-operator/tree/master/contrib/grafana-watcher/grafana

type grafanaDatasource struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Access string `json:"access"`
	URL    string `json:"URL"`
}

type datasourcesClient struct {
	baseURL    *url.URL
	HTTPClient *http.Client
}

func newDatasourcesClient(baseURL *url.URL, c *http.Client) *datasourcesClient {
	d := &datasourcesClient{
		baseURL:    baseURL,
		HTTPClient: c,
	}

	if c == nil {
		d.HTTPClient = http.DefaultClient
	}
	return d
}

func (c *datasourcesClient) List() ([]*grafanaDatasource, error) {
	allURL := makeURL(c.baseURL, "/api/datasources")
	resp, err := c.HTTPClient.Get(allURL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get %s", allURL)
	}
	defer closeCloser(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}
	datasources := make([]*grafanaDatasource, 0)

	err = json.NewDecoder(resp.Body).Decode(&datasources)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode response body from %s", allURL)
	}

	return datasources, nil
}

func (c *datasourcesClient) Get(name string) (*grafanaDatasource, error) {
	getURL := makeURL(c.baseURL, "/api/datasources/name/"+name)
	resp, err := c.HTTPClient.Get(getURL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get %s", getURL)
	}
	defer closeCloser(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
	// it's ok.
	case http.StatusNotFound:
		return nil, nil
	default:
		return nil, errors.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	var d grafanaDatasource

	err = json.NewDecoder(resp.Body).Decode(&d)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode response body from %s", getURL)
	}

	return &d, nil
}

func (c *datasourcesClient) Delete(id int) error {
	deleteURL := makeURL(c.baseURL, "/api/datasources/"+strconv.Itoa(id))
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to create http request for %s", deleteURL)
	}

	return doRequest(c.HTTPClient, req)
}

func (c *datasourcesClient) Create(d *grafanaDatasource) error {
	createURL := makeURL(c.baseURL, "/api/datasources")

	data, err := json.Marshal(d)
	if err != nil {
		return errors.Wrap(err, "failed to marshal datasource")
	}
	req, err := http.NewRequest("POST", createURL, bytes.NewBuffer(data))
	if err != nil {
		return errors.Wrapf(err, "failed to create http request for %s", createURL)
	}
	req.Header.Add("Content-Type", "application/json")

	return doRequest(c.HTTPClient, req)
}

func (c *datasourcesClient) Update(d *grafanaDatasource) error {
	updateURL := makeURL(c.baseURL, "/api/datasources/"+strconv.Itoa(d.ID))

	data, err := json.Marshal(d)
	if err != nil {
		return errors.Wrap(err, "failed to marshal datasource")
	}
	req, err := http.NewRequest("PUT", updateURL, bytes.NewBuffer(data))
	if err != nil {
		return errors.Wrapf(err, "failed to create http request for %s", updateURL)
	}
	req.Header.Add("Content-Type", "application/json")

	return doRequest(c.HTTPClient, req)
}

func makeURL(baseURL *url.URL, endpoint string) string {
	result := &url.URL{}
	*result = *baseURL

	result.Path = path.Join(result.Path, endpoint)

	return result.String()
}

func doRequest(c *http.Client, req *http.Request) error {
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer closeCloser(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}
	return nil
}

func closeCloser(c io.Closer) {
	_ = c.Close()
}
