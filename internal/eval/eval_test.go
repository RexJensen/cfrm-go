package eval

import "testing"

func TestCompareLowCardAceLow(t *testing.T) {
	if Compare(EvaluatorLowCard, "A", "K") != 1 {
		t.Fatalf("expected A to beat K in low-card")
	}
	if Compare(EvaluatorLowCard, "K", "A") != -1 {
		t.Fatalf("expected K to lose to A in low-card")
	}
	if Compare(EvaluatorLowCard, "7", "7") != 0 {
		t.Fatalf("expected tie")
	}
}

func TestCompareHighCardAceLow(t *testing.T) {
	if Compare(EvaluatorHighCard, "K", "A") != 1 {
		t.Fatalf("expected K to beat A in high-card")
	}
	if Compare(EvaluatorHighCard, "A", "K") != -1 {
		t.Fatalf("expected A to lose to K in high-card")
	}
	if Compare(EvaluatorHighCard, "7", "7") != 0 {
		t.Fatalf("expected tie")
	}
}
