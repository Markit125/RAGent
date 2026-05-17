package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type GoogleSearchClient struct {
	ApiKey string
	CX     string
}

func NewClient(apiKey, cx string) *GoogleSearchClient {
	return &GoogleSearchClient{ApiKey: apiKey, CX: cx}
}

type SearchResult struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	Link    string `json:"link"`
}

type googleResponse struct {
	Items []SearchResult `json:"items"`
}

func (c *GoogleSearchClient) Search(ctx context.Context, query string) (string, error) {
	endpoint := "https://www.googleapis.com/customsearch/v1"

	params := url.Values{}
	params.Add("key", c.ApiKey)
	params.Add("cx", c.CX)
	params.Add("q", query)
	params.Add("num", "3")

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("google api error: %d", resp.StatusCode)
	}

	var data googleResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	if len(data.Items) == 0 {
		return "SYSTEM_OBSERVATION: В интернете ничего не найдено.", nil
	}

	result := "SYSTEM_OBSERVATION: Результаты поиска в Google:\n"
	for _, item := range data.Items {
		result += fmt.Sprintf("- %s: %s (%s)\n", item.Title, item.Snippet, item.Link)
	}

	return result, nil
}
