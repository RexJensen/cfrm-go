package solver

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/chanian/cfrm-go/internal/config"
	"github.com/chanian/cfrm-go/internal/eval"
	"github.com/chanian/cfrm-go/internal/tree"
)

type Matchup struct {
	P1   string
	P2   string
	Prob float32
}

type InfoSet struct {
	Key      string
	BaseKey  string
	Hand     string
	Labels   []string
	Regret   []float32
	StratSum []float32

	FreqSum float32
	EV1Sum  float32
	EV2Sum  float32
}

type Result struct {
	Data           map[string][][]any
	Labels         map[string][]string
	EV1            float32
	EV2            float32
	Iterations     int
	Infosets       int
	Exploitability *Exploitability
}

type EVStats struct {
	Evs []float32 `json:"evs"`
	EV1 float32   `json:"ev1"`
	EV2 float32   `json:"ev2"`
}

type BRStats struct {
	P1 EVStats `json:"p1"`
	P2 EVStats `json:"p2"`
}

type Exploitability struct {
	P1       float32 `json:"p1"`
	P2       float32 `json:"p2"`
	Total    float32 `json:"total"`
	P1Pct    float32 `json:"p1_pct"`
	P2Pct    float32 `json:"p2_pct"`
	TotalPct float32 `json:"total_pct"`
	BR       BRStats `json:"br"`
}

func Run(cfg config.Config, root *tree.Node) (Result, error) {
	return RunWithProgress(cfg, root, nil)
}

func RunWithProgress(cfg config.Config, root *tree.Node, onProgress func(iteration, total int)) (Result, error) {
	p1Range, err := parseRange(cfg.Tree.P1Range)
	if err != nil {
		return Result{}, fmt.Errorf("parse p1Range: %w", err)
	}
	p2Range, err := parseRange(cfg.Tree.P2Range)
	if err != nil {
		return Result{}, fmt.Errorf("parse p2Range: %w", err)
	}
	matchups := buildMatchups(p1Range, p2Range)
	if len(matchups) == 0 {
		return Result{}, fmt.Errorf("no matchups from ranges")
	}

	infoSets := make(map[string]*InfoSet)
	for i := 0; i < cfg.Iterations; i++ {
		for _, m := range matchups {
			cfr(root, m, cfg.Evaluator, 1.0, 1.0, m.Prob, cfg.PotSize, infoSets)
		}
		if onProgress != nil {
			onProgress(i+1, cfg.Iterations)
		}
	}

	accumulateStats(root, matchups, cfg.Evaluator, cfg.PotSize, infoSets)
	ev1 := evalEV(root, matchups, cfg.Evaluator, cfg.PotSize, infoSets)
	ev2 := cfg.PotSize - ev1
	brP1 := computeBestResponseEV(root, matchups, 1, cfg.Evaluator, cfg.PotSize, infoSets)
	brP2 := computeBestResponseEV(root, matchups, 2, cfg.Evaluator, cfg.PotSize, infoSets)
	expP1 := brP1.EV1 - ev1
	expP2 := brP2.EV2 - ev2
	totalExp := (expP1 + expP2) / 2
	pot := cfg.PotSize
	exploitability := &Exploitability{
		P1:       expP1,
		P2:       expP2,
		Total:    totalExp,
		P1Pct:    safePct(expP1, pot),
		P2Pct:    safePct(expP2, pot),
		TotalPct: safePct(totalExp, pot),
		BR: BRStats{
			P1: brP1,
			P2: brP2,
		},
	}

	data := make(map[string][][]any)
	labels := make(map[string][]string)

	for _, is := range infoSets {
		strat := avgStrategy(is)
		row := make([]any, 0, 1+len(strat)+3)
		row = append(row, is.Hand)
		for _, s := range strat {
			row = append(row, s)
		}
		freq := is.FreqSum
		var ev1h float32
		var ev2h float32
		if freq > 0 {
			ev1h = is.EV1Sum / freq
			ev2h = is.EV2Sum / freq
		}
		row = append(row, freq, ev1h, ev2h)

		data[is.BaseKey] = append(data[is.BaseKey], row)
		if _, ok := labels[is.BaseKey]; !ok {
			labels[is.BaseKey] = cloneStrings(is.Labels)
		}
	}

	for bucket, rows := range data {
		labelCount := len(labels[bucket])
		freqIdx := 1 + labelCount
		var maxFreq float32
		for _, row := range rows {
			f := row[freqIdx].(float32)
			if f > maxFreq {
				maxFreq = f
			}
		}
		if maxFreq > 0 {
			for _, row := range rows {
				row[freqIdx] = row[freqIdx].(float32) / maxFreq
			}
		}
		sort.Slice(rows, func(i, j int) bool {
			return rankToNum(rows[i][0].(string)) < rankToNum(rows[j][0].(string))
		})
		data[bucket] = rows
	}

	return Result{
		Data:           data,
		Labels:         labels,
		EV1:            ev1,
		EV2:            ev2,
		Iterations:     cfg.Iterations,
		Infosets:       len(infoSets),
		Exploitability: exploitability,
	}, nil
}

func cfr(n *tree.Node, m Matchup, evaluator eval.Evaluator, reach1, reach2, chanceProb, pot float32, infoSets map[string]*InfoSet) float32 {
	if n.Type == tree.Terminal {
		return terminalUtilityP1(n, m, evaluator, pot)
	}

	is := getInfoSet(n, m, infoSets)
	strat := regretStrategy(is)

	utils := make([]float32, len(n.Children))
	var nodeUtil float32
	for i, c := range n.Children {
		nr1, nr2 := reach1, reach2
		if n.Player == 1 {
			nr1 *= strat[i]
		} else {
			nr2 *= strat[i]
		}
		u := cfr(c, m, evaluator, nr1, nr2, chanceProb, pot, infoSets)
		utils[i] = u
		nodeUtil += strat[i] * u
	}

	if n.Player == 1 {
		for i := range strat {
			is.Regret[i] += chanceProb * reach2 * (utils[i] - nodeUtil)
			is.StratSum[i] += chanceProb * reach1 * strat[i]
		}
	} else {
		for i := range strat {
			is.Regret[i] += chanceProb * reach1 * (nodeUtil - utils[i])
			is.StratSum[i] += chanceProb * reach2 * strat[i]
		}
	}
	return nodeUtil
}

func accumulateStats(root *tree.Node, matchups []Matchup, evaluator eval.Evaluator, pot float32, infoSets map[string]*InfoSet) {
	for _, is := range infoSets {
		is.FreqSum = 0
		is.EV1Sum = 0
		is.EV2Sum = 0
	}
	for _, m := range matchups {
		walkStats(root, m, evaluator, 1.0, 1.0, m.Prob, pot, infoSets)
	}
}

func walkStats(n *tree.Node, m Matchup, evaluator eval.Evaluator, reach1, reach2, chanceProb, pot float32, infoSets map[string]*InfoSet) float32 {
	if n.Type == tree.Terminal {
		return terminalUtilityP1(n, m, evaluator, pot)
	}
	is := getInfoSet(n, m, infoSets)
	strat := avgStrategy(is)

	var nodeUtil float32
	for i, c := range n.Children {
		nr1, nr2 := reach1, reach2
		if n.Player == 1 {
			nr1 *= strat[i]
		} else {
			nr2 *= strat[i]
		}
		u := walkStats(c, m, evaluator, nr1, nr2, chanceProb, pot, infoSets)
		nodeUtil += strat[i] * u
	}

	reach := reach1
	if n.Player == 2 {
		reach = reach2
	}
	w := chanceProb * reach
	is.FreqSum += w
	is.EV1Sum += w * nodeUtil
	is.EV2Sum += w * (pot - nodeUtil)

	return nodeUtil
}

func evalEV(n *tree.Node, matchups []Matchup, evaluator eval.Evaluator, pot float32, infoSets map[string]*InfoSet) float32 {
	var total float32
	for _, m := range matchups {
		total += m.Prob * evalNode(n, m, evaluator, pot, infoSets)
	}
	return total
}

func evalNode(n *tree.Node, m Matchup, evaluator eval.Evaluator, pot float32, infoSets map[string]*InfoSet) float32 {
	if n.Type == tree.Terminal {
		return terminalUtilityP1(n, m, evaluator, pot)
	}
	is := getInfoSet(n, m, infoSets)
	strat := avgStrategy(is)
	var sum float32
	for i, c := range n.Children {
		sum += strat[i] * evalNode(c, m, evaluator, pot, infoSets)
	}
	return sum
}

func terminalUtilityP1(n *tree.Node, m Matchup, evaluator eval.Evaluator, pot float32) float32 {
	switch n.Kind {
	case tree.FoldP1:
		return -n.P1Commit
	case tree.FoldP2:
		return pot + n.P2Commit
	default:
		cmp := eval.Compare(evaluator, m.P1, m.P2)
		if cmp > 0 {
			return pot + n.P2Commit
		}
		if cmp < 0 {
			return -n.P1Commit
		}
		return 0.5*pot + 0.5*(n.P2Commit-n.P1Commit)
	}
}

func terminalUtilityTarget(n *tree.Node, m Matchup, target int, evaluator eval.Evaluator, pot float32) float32 {
	u1 := terminalUtilityP1(n, m, evaluator, pot)
	if target == 1 {
		return u1
	}
	return pot - u1
}

func computeBestResponseEV(root *tree.Node, matchups []Matchup, target int, evaluator eval.Evaluator, pot float32, infoSets map[string]*InfoSet) EVStats {
	policy := map[string]int{}
	for iter := 0; iter < 128; iter++ {
		actionSums := map[string][]float32{}
		for _, m := range matchups {
			accumulatePolicyImprovement(root, m, target, evaluator, pot, infoSets, policy, actionSums, m.Prob)
		}
		next := buildBestResponsePolicy(actionSums)
		if samePolicy(policy, next) {
			policy = next
			break
		}
		policy = next
	}
	var totalTarget float32
	for _, m := range matchups {
		totalTarget += m.Prob * evaluateBestResponseWithPolicy(root, m, target, evaluator, pot, infoSets, policy)
	}
	ev1 := totalTarget
	ev2 := pot - totalTarget
	if target == 2 {
		ev2 = totalTarget
		ev1 = pot - totalTarget
	}
	return EVStats{
		Evs: []float32{ev1, ev2},
		EV1: ev1,
		EV2: ev2,
	}
}

func accumulatePolicyImprovement(
	n *tree.Node,
	m Matchup,
	target int,
	evaluator eval.Evaluator,
	pot float32,
	infoSets map[string]*InfoSet,
	policy map[string]int,
	actionSums map[string][]float32,
	reachWeight float32,
) float32 {
	if n.Type == tree.Terminal {
		return terminalUtilityTarget(n, m, target, evaluator, pot)
	}
	if n.Player == target {
		key := concreteKey(n.InfoKey, actingHand(target, m))
		sums, ok := actionSums[key]
		if !ok || len(sums) != len(n.Children) {
			sums = make([]float32, len(n.Children))
		}
		childValues := make([]float32, len(n.Children))
		for i, c := range n.Children {
			v := accumulatePolicyImprovement(c, m, target, evaluator, pot, infoSets, policy, actionSums, reachWeight)
			childValues[i] = v
			sums[i] += reachWeight * v
		}
		actionSums[key] = sums
		idx, ok := policy[key]
		if !ok || idx < 0 || idx >= len(childValues) {
			idx = 0
		}
		return childValues[idx]
	}

	is := getInfoSet(n, m, infoSets)
	strat := avgStrategy(is)
	var sum float32
	for i, c := range n.Children {
		p := strat[i]
		v := accumulatePolicyImprovement(c, m, target, evaluator, pot, infoSets, policy, actionSums, reachWeight*p)
		sum += p * v
	}
	return sum
}

func buildBestResponsePolicy(actionSums map[string][]float32) map[string]int {
	policy := map[string]int{}
	for k, sums := range actionSums {
		bestIdx := 0
		bestVal := float32(-math.MaxFloat32)
		for i, v := range sums {
			if v > bestVal {
				bestVal = v
				bestIdx = i
			}
		}
		policy[k] = bestIdx
	}
	return policy
}

func evaluateBestResponseWithPolicy(
	n *tree.Node,
	m Matchup,
	target int,
	evaluator eval.Evaluator,
	pot float32,
	infoSets map[string]*InfoSet,
	policy map[string]int,
) float32 {
	if n.Type == tree.Terminal {
		return terminalUtilityTarget(n, m, target, evaluator, pot)
	}

	if n.Player == target {
		key := concreteKey(n.InfoKey, actingHand(target, m))
		idx, ok := policy[key]
		if !ok || idx < 0 || idx >= len(n.Children) {
			idx = 0
		}
		return evaluateBestResponseWithPolicy(n.Children[idx], m, target, evaluator, pot, infoSets, policy)
	}

	is := getInfoSet(n, m, infoSets)
	strat := avgStrategy(is)
	var sum float32
	for i, c := range n.Children {
		p := strat[i]
		sum += p * evaluateBestResponseWithPolicy(c, m, target, evaluator, pot, infoSets, policy)
	}
	return sum
}

func safePct(v, pot float32) float32 {
	if pot == 0 {
		return 0
	}
	return v / pot
}

func samePolicy(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		if bv, ok := b[k]; !ok || bv != av {
			return false
		}
	}
	return true
}

func regretStrategy(is *InfoSet) []float32 {
	out := make([]float32, len(is.Regret))
	var sum float32
	for i, r := range is.Regret {
		if r > 0 {
			out[i] = r
			sum += r
		}
	}
	if sum <= 0 {
		v := 1.0 / float32(len(out))
		for i := range out {
			out[i] = v
		}
		return out
	}
	for i := range out {
		out[i] /= sum
	}
	return out
}

func avgStrategy(is *InfoSet) []float32 {
	out := make([]float32, len(is.StratSum))
	var sum float32
	for _, v := range is.StratSum {
		sum += v
	}
	if sum <= 0 {
		u := 1.0 / float32(len(out))
		for i := range out {
			out[i] = u
		}
		return out
	}
	for i := range out {
		out[i] = is.StratSum[i] / sum
	}
	return out
}

func getInfoSet(n *tree.Node, m Matchup, infoSets map[string]*InfoSet) *InfoSet {
	hand := actingHand(n.Player, m)
	key := concreteKey(n.InfoKey, hand)
	is, ok := infoSets[key]
	if ok {
		return is
	}
	is = &InfoSet{
		Key:      key,
		BaseKey:  n.InfoKey,
		Hand:     hand,
		Labels:   cloneStrings(n.Labels),
		Regret:   make([]float32, len(n.Labels)),
		StratSum: make([]float32, len(n.Labels)),
	}
	infoSets[key] = is
	return is
}

func actingHand(player int, m Matchup) string {
	if player == 1 {
		return m.P1
	}
	return m.P2
}

func concreteKey(baseKey, hand string) string {
	return baseKey + "_" + hand
}

func parseRange(raw string) (map[string]float32, error) {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty range")
	}
	ret := map[string]float32{}
	var total float32
	for _, p := range parts {
		tok := strings.Split(strings.TrimSpace(p), ":")
		if len(tok) != 2 {
			return nil, fmt.Errorf("invalid token %q", p)
		}
		r, err := eval.ParseRank(tok[0])
		if err != nil {
			return nil, err
		}
		w, err := strconv.ParseFloat(strings.TrimSpace(tok[1]), 32)
		if err != nil {
			return nil, fmt.Errorf("invalid weight in %q", p)
		}
		if w < 0 {
			return nil, fmt.Errorf("negative weight in %q", p)
		}
		fw := float32(w)
		ret[r] += fw
		total += fw
	}
	if total <= 0 {
		return nil, fmt.Errorf("range total weight is zero")
	}
	for k := range ret {
		ret[k] /= total
	}
	return ret, nil
}

func buildMatchups(p1, p2 map[string]float32) []Matchup {
	rows := make([]Matchup, 0, len(p1)*len(p2))
	for r1, pR1 := range p1 {
		for r2, pR2 := range p2 {
			prob := pR1 * pR2
			if prob <= 0 {
				continue
			}
			rows = append(rows, Matchup{P1: r1, P2: r2, Prob: prob})
		}
	}
	return rows
}

func rankToNum(rank string) float32 {
	for i, r := range eval.RankOrder() {
		if r == rank {
			return float32(i + 1)
		}
	}
	return 0
}

func cloneStrings(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	return out
}
