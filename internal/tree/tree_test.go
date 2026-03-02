package tree

import "testing"

func TestRootActions(t *testing.T) {
	b := Builder{PotSize: 3, MaxBets: 3}
	root, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	if root.Player != 1 {
		t.Fatalf("expected player 1 to act first")
	}
	if len(root.Actions) != 2 || root.Actions[0] != ActionBet || root.Actions[1] != ActionCheck {
		t.Fatalf("unexpected root actions: %+v", root.Actions)
	}
}
