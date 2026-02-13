package gitea

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nihaldivyam/opsbridge/internal/config"
)

type Client struct {
	httpClient *http.Client
	config     *config.Config
}

type Issue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"` // NEW: We need the body to check for the alertname
}

type CommentPayload struct {
	Body string `json:"body"`
}

type AddTimePayload struct {
	Time int64 `json:"time"`
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		config:     cfg,
	}
}

func (c *Client) setAuth(req *http.Request) {
	if c.config.GiteaBasicUser != "" && c.config.GiteaBasicPass != "" {
		req.SetBasicAuth(c.config.GiteaBasicUser, c.config.GiteaBasicPass)
		q := req.URL.Query()
		q.Add("token", c.config.GiteaToken)
		req.URL.RawQuery = q.Encode()
	} else {
		req.Header.Add("Authorization", "token "+c.config.GiteaToken)
	}
}

// FindIssueByLabels fetches open issues and uses Go to perfectly match the strings, bypassing Gitea's search indexer.
func (c *Client) FindIssueByLabels(alertname, certname string) (int, error) {
	// 1. Fetch up to 100 open issues directly, ignoring the 'q' search parameter
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues?state=open&limit=100",
		c.config.GiteaURL, c.config.GiteaOwner, c.config.GiteaRepo)

	req, _ := http.NewRequest("GET", url, nil)
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to fetch issues, status: %d", resp.StatusCode)
	}

	var issues []Issue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return 0, err
	}

	// 2. Locally scan the exact strings
	for _, issue := range issues {
		// We know the certname is in the Title, and the alertname is in the Body
		if strings.Contains(issue.Title, certname) && strings.Contains(issue.Body, alertname) {
			return issue.Number, nil
		}
	}

	return 0, fmt.Errorf("no open issue found matching alertname: %s on certname: %s", alertname, certname)
}

func (c *Client) AddComment(issueNumber int, text string) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/comments",
		c.config.GiteaURL, c.config.GiteaOwner, c.config.GiteaRepo, issueNumber)

	body, _ := json.Marshal(CommentPayload{Body: text})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))

	c.setAuth(req)
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to add comment, status: %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) AddTimeReg(issueNumber int, durationStr string) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/times",
		c.config.GiteaURL, c.config.GiteaOwner, c.config.GiteaRepo, issueNumber)

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return fmt.Errorf("invalid time format: %v", err)
	}

	body, _ := json.Marshal(AddTimePayload{Time: int64(duration.Seconds())})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))

	c.setAuth(req)
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add time, status: %d", resp.StatusCode)
	}
	return nil
}
