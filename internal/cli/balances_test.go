package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"lunchmoney-cli/internal/lunchmoney"
)

func TestToBalancesViewRoundsTotalsForJSON(t *testing.T) {
	view := toBalancesView(
		[]lunchmoney.ManualAccount{
			{
				ID:          1,
				Name:        "Checking",
				Type:        "depository",
				Balance:     "100.00",
				Currency:    "usd",
				ToBase:      100,
				BalanceAsOf: "2026-05-22T01:00:00Z",
				Status:      "active",
			},
		},
		[]lunchmoney.PlaidAccount{
			{
				ID:              2,
				Name:            "Card 1",
				InstitutionName: "Bank",
				Type:            "credit",
				Balance:         "0.10",
				Currency:        "usd",
				ToBase:          0.1,
				BalanceAsOf:     "2026-05-22T02:00:00Z",
				Status:          "active",
			},
			{
				ID:              3,
				Name:            "Card 2",
				InstitutionName: "Bank",
				Type:            "credit",
				Balance:         "0.20",
				Currency:        "usd",
				ToBase:          0.2,
				BalanceAsOf:     "2026-05-22T03:00:00Z",
				Status:          "active",
			},
		},
		false,
	)

	out, err := json.Marshal(view.Totals)
	if err != nil {
		t.Fatal(err)
	}

	jsonText := string(out)
	if strings.Contains(jsonText, "000000000") || strings.Contains(jsonText, "999999999") {
		t.Fatalf("totals contain float artifacts: %s", jsonText)
	}

	if view.Totals.CreditBalances != 0.3 {
		t.Fatalf("credit balances = %v, want 0.3", view.Totals.CreditBalances)
	}
	if view.Totals.NetCashAfterCredit != 99.7 {
		t.Fatalf("net cash after credit = %v, want 99.7", view.Totals.NetCashAfterCredit)
	}
}
