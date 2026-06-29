package cli

import (
	"reflect"
	"strings"
	"testing"

	"lunchmoney-cli/internal/lunchmoney"
)

func TestSplitAmountIntoParts(t *testing.T) {
	tests := []struct {
		name   string
		amount string
		parts  int
		want   []string
	}{
		{
			name:   "two way split distributes odd cent to first child",
			amount: "113.9500",
			parts:  2,
			want:   []string{"56.98", "56.97"},
		},
		{
			name:   "three way split distributes remainder left to right",
			amount: "113.9500",
			parts:  3,
			want:   []string{"37.99", "37.98", "37.98"},
		},
		{
			name:   "negative split preserves exact sum",
			amount: "-10.00",
			parts:  3,
			want:   []string{"-3.34", "-3.33", "-3.33"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitAmountIntoParts(tt.amount, tt.parts)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("splitAmountIntoParts() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSplitAmountIntoPartsRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name      string
		amount    string
		parts     int
		wantError string
	}{
		{
			name:      "too few parts",
			amount:    "10.00",
			parts:     1,
			wantError: "--parts must be at least 2",
		},
		{
			name:      "fractional cent parent",
			amount:    "10.0050",
			parts:     2,
			wantError: "fractional cents",
		},
		{
			name:      "would create zero amount child",
			amount:    "0.01",
			parts:     2,
			wantError: "non-zero cent amounts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := splitAmountIntoParts(tt.amount, tt.parts)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %q, want to contain %q", err, tt.wantError)
			}
		})
	}
}

func TestValidateSplitAmounts(t *testing.T) {
	got, err := validateSplitAmounts("113.9500", []string{"56.98", "56.97"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"56.98", "56.97"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("validateSplitAmounts() = %#v, want %#v", got, want)
	}

	got, err = validateSplitAmounts("3.5800", []string{"1.2345", "2.3455"})
	if err != nil {
		t.Fatal(err)
	}
	want = []string{"1.2345", "2.3455"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("validateSplitAmounts() = %#v, want %#v", got, want)
	}
}

func TestValidateSplitAmountsRejectsBadSum(t *testing.T) {
	_, err := validateSplitAmounts("113.9500", []string{"56.98", "56.98"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "split amounts sum to 113.96") {
		t.Fatalf("error = %q", err)
	}
}

func TestValidateSplittableTransactionRejectsExistingSplitOrGroup(t *testing.T) {
	splitParentID := int64(10)
	groupParentID := int64(20)

	tests := []lunchmoney.Transaction{
		{ID: 1, Amount: "10.00", IsSplitParent: true},
		{ID: 2, Amount: "10.00", SplitParentID: &splitParentID},
		{ID: 3, Amount: "10.00", IsGroupParent: true},
		{ID: 4, Amount: "10.00", GroupParentID: &groupParentID},
	}

	for _, tx := range tests {
		if err := validateSplittableTransaction(tx); err == nil {
			t.Fatalf("expected transaction %d to be rejected", tx.ID)
		}
	}
}
