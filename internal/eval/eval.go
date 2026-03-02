package eval

import (
	"fmt"
	"strings"
)

type Evaluator string

const (
	EvaluatorHighCard Evaluator = "HIGH_CARD"
	EvaluatorLowCard  Evaluator = "LOW_CARD"
)

var order = []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "T", "J", "Q", "K"}
var indexByRank map[string]int

func init() {
	indexByRank = make(map[string]int, len(order))
	for i, r := range order {
		indexByRank[r] = i
	}
}

func RankOrder() []string {
	out := make([]string, len(order))
	copy(out, order)
	return out
}

func ParseRank(raw string) (string, error) {
	r := strings.ToUpper(strings.TrimSpace(raw))
	if _, ok := indexByRank[r]; !ok {
		return "", fmt.Errorf("invalid rank %q", raw)
	}
	return r, nil
}

func ParseEvaluator(raw string) (Evaluator, error) {
	v := Evaluator(strings.ToUpper(strings.TrimSpace(raw)))
	switch v {
	case EvaluatorHighCard, EvaluatorLowCard:
		return v, nil
	default:
		return "", fmt.Errorf("invalid evaluator %q", raw)
	}
}

func OutputName(kind Evaluator) string {
	if kind == EvaluatorHighCard {
		return "evaluateHighCardAceLow"
	}
	return "evaluateLowCardAceLow"
}

func Compare(kind Evaluator, p1, p2 string) int {
	i1 := indexByRank[p1]
	i2 := indexByRank[p2]
	if kind == EvaluatorHighCard {
		return compareHighCard(i1, i2)
	}
	return compareLowCard(i1, i2)
}
