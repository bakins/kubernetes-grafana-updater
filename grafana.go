package main

// maybe use github.com/apparentlymart/go-grafana-api ?

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"go.uber.org/zap"

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

// based on https://godoc.org/github.com/apparentlymart/go-grafana-api
type grafanaDashboard struct {
	Meta  dashboardMeta          `json:"meta"`
	Model map[string]interface{} `json:"dashboard"`
}

type dashboardMeta struct {
	IsStarred bool   `json:"isStarred"`
	Slug      string `json:"slug"`
}

type grafanaDashboardRequest struct {
	Dashboard map[string]interface{} `json:"dashboard"`
	Overwrite bool                   `json:"overwrite"`
}

type grafanaClient struct {
	baseURL    *url.URL
	HTTPClient *http.Client
}

func newGrafanaClient(baseURL *url.URL, client *http.Client) *grafanaClient {
	c := &grafanaClient{
		baseURL:    baseURL,
		HTTPClient: client,
	}

	if client == nil {
		c.HTTPClient = http.DefaultClient
	}
	return c
}

// wait until api is ready
func (c *grafanaClient) wait() {
	getURL := makeURL(c.baseURL, "/api/datasources")
	for i := 0; i < 100; i++ {
		resp, err := c.HTTPClient.Get(getURL)
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(time.Second)
	}

	err := errors.New("unable to contact grafana")
	logger.Fatal("gave up on grafana", zap.Error(err))
}

func (c *grafanaClient) GetDatasource(name string) (*grafanaDatasource, error) {
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

func (c *grafanaClient) DeleteDatasource(id int) error {
	deleteURL := makeURL(c.baseURL, "/api/datasources/"+strconv.Itoa(id))
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to create http request for %s", deleteURL)
	}

	return doRequest(c.HTTPClient, req)
}

func (c *grafanaClient) CreateDatasource(d *grafanaDatasource) error {
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

func (c *grafanaClient) UpdateDatasource(d *grafanaDatasource) error {
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

func (c *grafanaClient) GetDashboard(slug string) (*grafanaDashboard, error) {
	getURL := makeURL(c.baseURL, "/api/dashboards/db/"+slug)
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

	var d grafanaDashboard

	err = json.NewDecoder(resp.Body).Decode(&d)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode response body from %s", getURL)
	}

	return &d, nil
}

func (c *grafanaClient) DeleteDashboard(slug string) error {
	deleteURL := makeURL(c.baseURL, "/api/dashboards/db/"+slug)
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to create http request for %s", deleteURL)
	}

	return doRequest(c.HTTPClient, req)
}

func (c *grafanaClient) CreateDashboard(d *grafanaDashboard) error {

	r := &grafanaDashboardRequest{
		Dashboard: d.Model,
		Overwrite: false,
	}
	return c.postDashboardRequest(r)
}

func (c *grafanaClient) UpdateDashboard(d *grafanaDashboard) error {
	r := &grafanaDashboardRequest{
		Dashboard: d.Model,
		Overwrite: true,
	}
	return c.postDashboardRequest(r)
}

func (c *grafanaClient) postDashboardRequest(r *grafanaDashboardRequest) error {
	postURL := makeURL(c.baseURL, "/api/dashboards/db")

	data, err := json.Marshal(r)
	if err != nil {
		return errors.Wrap(err, "failed to marshal dashboard request")
	}
	req, err := http.NewRequest("POST", postURL, bytes.NewBuffer(data))
	if err != nil {
		return errors.Wrapf(err, "failed to create http request for %s", postURL)
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
		msg := ""
		body, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			msg = string(body)
		}
		return errors.Errorf("unexpected HTTP status: %d %s", resp.StatusCode, msg)
	}
	return nil
}

func closeCloser(c io.Closer) {
	_ = c.Close()
}
