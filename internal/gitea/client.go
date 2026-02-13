package gitea

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

func (c *Client) FindIssueByHash(hash string) (int, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues?state=open&q=%s",
		c.config.GiteaURL, c.config.GiteaOwner, c.config.GiteaRepo, hash)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "token "+c.config.GiteaToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to search issues, status: %d", resp.StatusCode)
	}

	var issues []Issue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return 0, err
	}

	if len(issues) == 0 {
		return 0, fmt.Errorf("no issue found for hash: %s", hash)
	}

	return issues[0].Number, nil
}

func (c *Client) AddComment(issueNumber int, text string) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/comments",
		c.config.GiteaURL, c.config.GiteaOwner, c.config.GiteaRepo, issueNumber)

	body, _ := json.Marshal(CommentPayload{Body: text})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Add("Authorization", "token "+c.config.GiteaToken)
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
	req.Header.Add("Authorization", "token "+c.config.GiteaToken)
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
