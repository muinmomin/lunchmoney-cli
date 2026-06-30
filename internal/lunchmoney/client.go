package lunchmoney

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	envAPIKey      = "LUNCHMONEY_API_KEY"
	defaultBaseURL = "https://api.lunchmoney.dev/v2"
)

type Client struct {
	apiKey     string
	baseURL    *url.URL
	httpClient *http.Client
}

type ListTransactionsParams struct {
	StartDate      string
	EndDate        string
	Status         string
	IncludePending bool
	IsPending      *bool
	Limit          int
}

type Transaction struct {
	ID              int64         `json:"id"`
	Date            string        `json:"date"`
	Amount          string        `json:"amount"`
	Currency        string        `json:"currency"`
	ToBase          float64       `json:"to_base"`
	Payee           string        `json:"payee"`
	CategoryID      *int64        `json:"category_id"`
	ManualAccountID *int64        `json:"manual_account_id"`
	PlaidAccountID  *int64        `json:"plaid_account_id"`
	Notes           *string       `json:"notes"`
	Status          string        `json:"status"`
	IsPending       bool          `json:"is_pending"`
	IsSplitParent   bool          `json:"is_split_parent"`
	SplitParentID   *int64        `json:"split_parent_id"`
	IsGroupParent   bool          `json:"is_group_parent"`
	GroupParentID   *int64        `json:"group_parent_id"`
	TagIDs          []int64       `json:"tag_ids"`
	Children        []Transaction `json:"children,omitempty"`
}

type SplitTransactionChild struct {
	Amount     string  `json:"amount"`
	Payee      *string `json:"payee,omitempty"`
	Date       *string `json:"date,omitempty"`
	CategoryID *int64  `json:"category_id,omitempty"`
	Notes      *string `json:"notes,omitempty"`
}

type UpdateTransactionParams struct {
	CategoryID *int64
	Note       *string
	Date       *string
}

type Category struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	IsIncome          bool   `json:"is_income"`
	ExcludeFromTotals bool   `json:"exclude_from_totals"`
	GroupID           *int64 `json:"group_id"`
	IsGroup           bool   `json:"is_group"`
	Archived          bool   `json:"archived"`
}

type Tag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ManualAccount struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	InstitutionName *string `json:"institution_name"`
	DisplayName     *string `json:"display_name"`
	Type            string  `json:"type"`
	Subtype         *string `json:"subtype"`
	Balance         string  `json:"balance"`
	Currency        string  `json:"currency"`
	ToBase          float64 `json:"to_base"`
	BalanceAsOf     string  `json:"balance_as_of"`
	Status          string  `json:"status"`
}

type PlaidAccount struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	InstitutionName string  `json:"institution_name"`
	DisplayName     *string `json:"display_name"`
	Type            string  `json:"type"`
	Subtype         *string `json:"subtype"`
	Balance         string  `json:"balance"`
	Currency        string  `json:"currency"`
	ToBase          float64 `json:"to_base"`
	BalanceAsOf     string  `json:"balance_last_update"`
	Status          string  `json:"status"`
}

func NewFromEnv() (*Client, error) {
	apiKey := strings.TrimSpace(os.Getenv(envAPIKey))
	if apiKey == "" {
		return nil, fmt.Errorf("%s is not set", envAPIKey)
	}

	baseURL, err := url.Parse(defaultBaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid default base url: %w", err)
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *Client) ListTransactions(ctx context.Context, params ListTransactionsParams) ([]Transaction, error) {
	if params.StartDate == "" {
		return nil, errors.New("start date is required")
	}
	if params.EndDate == "" {
		return nil, errors.New("end date is required")
	}
	if params.Limit <= 0 {
		params.Limit = 1000
	}

	all := make([]Transaction, 0, params.Limit)
	offset := 0

	for {
		q := url.Values{}
		q.Set("start_date", params.StartDate)
		q.Set("end_date", params.EndDate)
		if params.Status != "" {
			q.Set("status", params.Status)
		}
		q.Set("limit", strconv.Itoa(params.Limit))
		q.Set("offset", strconv.Itoa(offset))
		if params.IsPending != nil {
			q.Set("is_pending", strconv.FormatBool(*params.IsPending))
		} else if params.IncludePending {
			q.Set("include_pending", "true")
		}

		u := c.endpoint("/transactions")
		u.RawQuery = q.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}

		var resp listTransactionsResponse
		if err := c.doJSON(req, http.StatusOK, &resp); err != nil {
			return nil, err
		}

		all = append(all, resp.Transactions...)
		if !resp.HasMore {
			break
		}
		offset += len(resp.Transactions)
		if len(resp.Transactions) == 0 {
			return nil, errors.New("pagination indicated more results but received empty page")
		}
	}

	return all, nil
}

func (c *Client) GetTransaction(ctx context.Context, txID int64) (Transaction, error) {
	if txID <= 0 {
		return Transaction{}, errors.New("transaction id must be positive")
	}

	u := c.endpoint(path.Join("/transactions", strconv.FormatInt(txID, 10)))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Transaction{}, err
	}

	var tx Transaction
	if err := c.doJSON(req, http.StatusOK, &tx); err != nil {
		return Transaction{}, err
	}
	return tx, nil
}

func (c *Client) ListCategories(ctx context.Context) ([]Category, error) {
	u := c.endpoint("/categories")
	q := url.Values{}
	q.Set("format", "flattened")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Categories []Category `json:"categories"`
	}
	if err := c.doJSON(req, http.StatusOK, &resp); err != nil {
		return nil, err
	}

	return resp.Categories, nil
}

func (c *Client) ListTags(ctx context.Context) ([]Tag, error) {
	u := c.endpoint("/tags")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Tags []Tag `json:"tags"`
	}
	if err := c.doJSON(req, http.StatusOK, &resp); err != nil {
		return nil, err
	}
	return resp.Tags, nil
}

func (c *Client) ListManualAccounts(ctx context.Context) ([]ManualAccount, error) {
	u := c.endpoint("/manual_accounts")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		ManualAccounts []ManualAccount `json:"manual_accounts"`
	}
	if err := c.doJSON(req, http.StatusOK, &resp); err != nil {
		return nil, err
	}
	return resp.ManualAccounts, nil
}

func (c *Client) ListPlaidAccounts(ctx context.Context) ([]PlaidAccount, error) {
	u := c.endpoint("/plaid_accounts")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		PlaidAccounts []PlaidAccount `json:"plaid_accounts"`
	}
	if err := c.doJSON(req, http.StatusOK, &resp); err != nil {
		return nil, err
	}
	return resp.PlaidAccounts, nil
}

func (c *Client) UpdateTransaction(ctx context.Context, txID int64, params UpdateTransactionParams) (Transaction, error) {
	if params.CategoryID == nil && params.Note == nil && params.Date == nil {
		return Transaction{}, errors.New("must provide category_id, note, and/or date")
	}

	payload := map[string]any{}
	if params.CategoryID != nil {
		payload["category_id"] = *params.CategoryID
	}
	if params.Note != nil {
		payload["notes"] = *params.Note
	}
	if params.Date != nil {
		payload["date"] = *params.Date
	}

	return c.updateTransaction(ctx, txID, payload)
}

func (c *Client) MarkReviewed(ctx context.Context, txIDs []int64) ([]Transaction, error) {
	if len(txIDs) == 0 {
		return nil, errors.New("at least one transaction id is required")
	}

	updated := make([]Transaction, 0, len(txIDs))
	for _, txID := range txIDs {
		tx, err := c.updateTransaction(ctx, txID, map[string]any{"status": "reviewed"})
		if err != nil {
			return nil, fmt.Errorf("failed to mark transaction %d as reviewed: %w", txID, err)
		}
		updated = append(updated, tx)
	}
	return updated, nil
}

func (c *Client) SplitTransaction(ctx context.Context, txID int64, children []SplitTransactionChild) (Transaction, error) {
	if txID <= 0 {
		return Transaction{}, errors.New("transaction id must be positive")
	}
	if len(children) < 2 {
		return Transaction{}, errors.New("at least two child transactions are required")
	}
	for i, child := range children {
		if strings.TrimSpace(child.Amount) == "" {
			return Transaction{}, fmt.Errorf("child transaction %d amount is required", i+1)
		}
	}

	payload := map[string]any{
		"child_transactions": children,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Transaction{}, err
	}

	u := c.endpoint(path.Join("/transactions/split", strconv.FormatInt(txID, 10)))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return Transaction{}, err
	}

	var tx Transaction
	if err := c.doJSON(req, http.StatusCreated, &tx); err != nil {
		return Transaction{}, err
	}
	return tx, nil
}

func (c *Client) updateTransaction(ctx context.Context, txID int64, payload map[string]any) (Transaction, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return Transaction{}, err
	}

	u := c.endpoint(path.Join("/transactions", strconv.FormatInt(txID, 10)))
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), bytes.NewReader(body))
	if err != nil {
		return Transaction{}, err
	}

	var tx Transaction
	if err := c.doJSONWithStatuses(req, []int{http.StatusOK, http.StatusCreated}, &tx); err != nil {
		return Transaction{}, err
	}
	return tx, nil
}

func (c *Client) endpoint(p string) *url.URL {
	u := *c.baseURL
	u.Path = strings.TrimRight(c.baseURL.Path, "/") + p
	return &u
}

func (c *Client) doJSON(req *http.Request, expectedStatus int, out any) error {
	return c.doJSONWithStatuses(req, []int{expectedStatus}, out)
}

func (c *Client) doJSONWithStatuses(req *http.Request, expectedStatuses []int, out any) error {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if !containsStatus(expectedStatuses, resp.StatusCode) {
		return decodeAPIError(resp)
	}

	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}

func containsStatus(allowed []int, got int) bool {
	for _, status := range allowed {
		if status == got {
			return true
		}
	}
	return false
}

func decodeAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var e struct {
		Message string `json:"message"`
		Errors  []struct {
			ErrMsg string `json:"errMsg"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &e); err != nil {
		trimmed := strings.TrimSpace(string(body))
		if trimmed == "" {
			return fmt.Errorf("api request failed with status %d", resp.StatusCode)
		}
		return fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, trimmed)
	}

	parts := make([]string, 0, len(e.Errors)+1)
	if e.Message != "" {
		parts = append(parts, e.Message)
	}
	for _, detail := range e.Errors {
		if detail.ErrMsg != "" {
			parts = append(parts, detail.ErrMsg)
		}
	}
	if len(parts) == 0 {
		return fmt.Errorf("api request failed with status %d", resp.StatusCode)
	}
	return fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, strings.Join(parts, "; "))
}

type listTransactionsResponse struct {
	Transactions []Transaction `json:"transactions"`
	HasMore      bool          `json:"has_more"`
}
