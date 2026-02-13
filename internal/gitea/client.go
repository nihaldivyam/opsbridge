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

// Consolidated Gitea User and Issue structs
type GiteaUser struct {
	Username string `json:"username"`
}

type Issue struct {
	Number    int          `json:"number"`
	Title     string       `json:"title"`
	Body      string       `json:"body"`
	State     string       `json:"state"`
	Assignees []GiteaUser  `json:"assignees"`
	Labels    []GiteaLabel `json:"labels"` // NEW: Capture alert labels
}

type AssigneePayload struct {
	Assignees []string `json:"assignees"`
}

type CommentPayload struct {
	Body string `json:"body"`
}

type GiteaLabel struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		config:     cfg,
	}
}

func (c *Client) setAuth(req *http.Request) {
	if c.config.GiteaBasicUser != "" {
		req.SetBasicAuth(c.config.GiteaBasicUser, c.config.GiteaBasicPass)
		q := req.URL.Query()
		q.Add("token", c.config.GiteaToken)
		req.URL.RawQuery = q.Encode()
	} else {
		req.Header.Add("Authorization", "token "+c.config.GiteaToken)
	}
}

// FindIssueByLabels fetches all issues and scans for labels case-insensitively
func (c *Client) FindIssueByLabels(alertname, certname string) (Issue, error) {
	// We use state=all to find both open and closed issues
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues?state=all&limit=100",
		c.config.GiteaURL, c.config.GiteaOwner, c.config.GiteaRepo)

	req, _ := http.NewRequest("GET", url, nil)
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Issue{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Issue{}, fmt.Errorf("failed to fetch issues, status: %d", resp.StatusCode)
	}

	var issues []Issue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return Issue{}, err
	}

	searchAlert := strings.ToLower(strings.TrimSpace(alertname))
	searchCert := strings.ToLower(strings.TrimSpace(certname))

	for _, issue := range issues {
		t, b := strings.ToLower(issue.Title), strings.ToLower(issue.Body)
		if (strings.Contains(t, searchCert) || strings.Contains(b, searchCert)) &&
			(strings.Contains(t, searchAlert) || strings.Contains(b, searchAlert)) {
			return issue, nil
		}
	}

	return Issue{}, fmt.Errorf("no match found for %s on %s", alertname, certname)
}

func (c *Client) AssignIssue(issueNumber int, username string) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d",
		c.config.GiteaURL, c.config.GiteaOwner, c.config.GiteaRepo, issueNumber)

	body, _ := json.Marshal(AssigneePayload{Assignees: []string{username}})
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(body))
	c.setAuth(req)
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Gitea returns 200 or 201 for successful updates
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to assign issue, status: %d", resp.StatusCode)
	}
	return nil
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
