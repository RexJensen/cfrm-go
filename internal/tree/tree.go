package tree

import "fmt"

type Action string

const (
	ActionBet   Action = "bet"
	ActionCheck Action = "check"
	ActionCall  Action = "call"
	ActionFold  Action = "fold"
	ActionRaise Action = "raise"
)

type NodeType int

const (
	Decision NodeType = iota
	Terminal
)

type TerminalKind int

const (
	Showdown TerminalKind = iota
	FoldP1
	FoldP2
)

type Node struct {
	Type     NodeType
	Player   int
	InfoKey  string
	Labels   []string
	Actions  []Action
	Children []*Node
	Wagers   [][2]float32

	Kind      TerminalKind
	P1Commit  float32
	P2Commit  float32
	PotBefore float32
}

type Builder struct {
	PotSize float32
	MaxBets int
}

func (b Builder) Build() (*Node, error) {
	if b.MaxBets <= 0 {
		return nil, fmt.Errorf("max bets must be > 0")
	}
	return b.buildDecision(1, 0, 0, 0, false, true, nil), nil
}

func (b Builder) target(level int) float32 {
	if level <= 0 {
		return 1
	}
	return float32(level)
}

func (b Builder) buildDecision(player, betLevel int, p1Commit, p2Commit float32, prevCheck bool, firstAction bool, hist []string) *Node {
	var pending float32
	if player == 1 {
		pending = p2Commit - p1Commit
	} else {
		pending = p1Commit - p2Commit
	}
	actions := legalActions(betLevel, pending, b.MaxBets, firstAction, prevCheck)
	labels := make([]string, len(actions))
	for i, a := range actions {
		labels[i] = string(a)
	}
	infoKey := buildInfoKey(player, hist)

	node := &Node{
		Type:    Decision,
		Player:  player,
		InfoKey: infoKey,
		Labels:  labels,
		Actions: actions,
	}

	for _, a := range actions {
		nextP1, nextP2 := p1Commit, p2Commit
		nextBetLevel := betLevel
		nextPrevCheck := false
		nextFirstAction := false
		hist2 := append(clone(hist), title(a))

		var child *Node
		delta := [2]float32{0, 0}
		switch a {
		case ActionBet:
			nextBetLevel = 1
			target := b.target(nextBetLevel)
			if player == 1 {
				delta[0] = target - nextP1
				nextP1 = target
			} else {
				delta[1] = target - nextP2
				nextP2 = target
			}
			child = b.buildDecision(other(player), nextBetLevel, nextP1, nextP2, false, nextFirstAction, hist2)
		case ActionCheck:
			if prevCheck {
				child = b.newShowdown(nextP1, nextP2)
			} else {
				nextPrevCheck = true
				child = b.buildDecision(other(player), nextBetLevel, nextP1, nextP2, nextPrevCheck, nextFirstAction, hist2)
			}
		case ActionCall:
			target := b.target(nextBetLevel)
			if player == 1 {
				delta[0] = target - nextP1
				nextP1 = target
			} else {
				delta[1] = target - nextP2
				nextP2 = target
			}
			child = b.newShowdown(nextP1, nextP2)
		case ActionFold:
			if player == 1 {
				child = b.newFoldP1(nextP1, nextP2)
			} else {
				child = b.newFoldP2(nextP1, nextP2)
			}
		case ActionRaise:
			nextBetLevel = betLevel + 1
			target := b.target(nextBetLevel)
			if player == 1 {
				delta[0] = target - nextP1
				nextP1 = target
			} else {
				delta[1] = target - nextP2
				nextP2 = target
			}
			child = b.buildDecision(other(player), nextBetLevel, nextP1, nextP2, false, nextFirstAction, hist2)
		}
		node.Children = append(node.Children, child)
		node.Wagers = append(node.Wagers, delta)
	}
	return node
}

func (b Builder) newShowdown(p1, p2 float32) *Node {
	return &Node{Type: Terminal, Kind: Showdown, P1Commit: p1, P2Commit: p2, PotBefore: b.PotSize}
}

func (b Builder) newFoldP1(p1, p2 float32) *Node {
	return &Node{Type: Terminal, Kind: FoldP1, P1Commit: p1, P2Commit: p2, PotBefore: b.PotSize}
}

func (b Builder) newFoldP2(p1, p2 float32) *Node {
	return &Node{Type: Terminal, Kind: FoldP2, P1Commit: p1, P2Commit: p2, PotBefore: b.PotSize}
}

func legalActions(betLevel int, pending float32, maxBets int, firstAction bool, prevCheck bool) []Action {
	if pending <= 0 {
		if firstAction {
			return []Action{ActionBet, ActionCheck}
		}
		if prevCheck {
			return []Action{ActionBet, ActionCheck}
		}
		return []Action{ActionCheck}
	}
	if betLevel < maxBets {
		return []Action{ActionCall, ActionFold, ActionRaise}
	}
	return []Action{ActionCall, ActionFold}
}

func buildInfoKey(player int, hist []string) string {
	prefix := fmt.Sprintf("P%d-", player)
	if len(hist) == 0 {
		return prefix
	}
	joined := ""
	for i, h := range hist {
		if i > 0 {
			joined += "-"
		}
		joined += h
	}
	return prefix + joined
}

func title(a Action) string {
	s := string(a)
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}

func other(p int) int {
	if p == 1 {
		return 2
	}
	return 1
}

func clone[T any](in []T) []T {
	out := make([]T, len(in))
	copy(out, in)
	return out
}
