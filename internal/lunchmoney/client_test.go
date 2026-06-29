package lunchmoney

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGetTransaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/transactions/123" {
			t.Fatalf("path = %s, want /transactions/123", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q", got)
		}

		_ = json.NewEncoder(w).Encode(Transaction{
			ID:     123,
			Amount: "10.0000",
			Payee:  "Test Payee",
		})
	}))
	defer server.Close()

	client := testClient(t, server)
	tx, err := client.GetTransaction(context.Background(), 123)
	if err != nil {
		t.Fatal(err)
	}
	if tx.ID != 123 || tx.Amount != "10.0000" || tx.Payee != "Test Payee" {
		t.Fatalf("transaction = %#v", tx)
	}
}

func TestSplitTransaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/transactions/split/123" {
			t.Fatalf("path = %s, want /transactions/split/123", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q", got)
		}

		var body struct {
			ChildTransactions []SplitTransactionChild `json:"child_transactions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.ChildTransactions) != 2 {
			t.Fatalf("child count = %d, want 2", len(body.ChildTransactions))
		}
		if body.ChildTransactions[0].Amount != "56.98" || body.ChildTransactions[1].Amount != "56.97" {
			t.Fatalf("child transactions = %#v", body.ChildTransactions)
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Transaction{
			ID:            123,
			Amount:        "113.9500",
			Payee:         "Hinodeya Ramen & Bar",
			IsSplitParent: true,
			Children: []Transaction{
				{ID: 124, Amount: "56.9800", Payee: "Hinodeya Ramen & Bar"},
				{ID: 125, Amount: "56.9700", Payee: "Hinodeya Ramen & Bar"},
			},
		})
	}))
	defer server.Close()

	client := testClient(t, server)
	tx, err := client.SplitTransaction(context.Background(), 123, []SplitTransactionChild{
		{Amount: "56.98"},
		{Amount: "56.97"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !tx.IsSplitParent || len(tx.Children) != 2 {
		t.Fatalf("transaction = %#v", tx)
	}
}

func testClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return &Client{
		apiKey:     "test-key",
		baseURL:    baseURL,
		httpClient: server.Client(),
	}
}
