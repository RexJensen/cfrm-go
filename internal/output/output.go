package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/chanian/cfrm-go/internal/config"
	"github.com/chanian/cfrm-go/internal/eval"
	"github.com/chanian/cfrm-go/internal/solver"
)

type meta struct {
	AllStrat []float32 `json:"allStrat"`
	Labels   []string  `json:"labels"`
}

type ev struct {
	Evs []float32 `json:"evs"`
	EV1 float32   `json:"ev1"`
	EV2 float32   `json:"ev2"`
}

type solverStats struct {
	Iteration    int     `json:"iteration"`
	Total        int     `json:"total"`
	Pot          float32 `json:"pot"`
	TotalInfoSet int     `json:"total_infosets"`
	NumPlayers   int     `json:"num_players"`
}

type stats struct {
	EV             ev              `json:"ev"`
	Exploitability any             `json:"exploitability"`
	IsMeta         map[string]meta `json:"isMeta"`
	Solver         solverStats     `json:"solver"`
}

type payload struct {
	P1         map[string][][]any `json:"p1"`
	P2         map[string][][]any `json:"p2"`
	Data       map[string][][]any `json:"data"`
	Config     string             `json:"config"`
	ConfigFile config.Config      `json:"configFile"`
	Stats      stats              `json:"stats"`
}

func Write(outPath string, cfg config.Config, res solver.Result) error {
	metaMap := map[string]meta{}
	for k, rows := range res.Data {
		labels := res.Labels[k]
		all := make([]float32, len(labels))
		var totalFreq float32
		for _, row := range rows {
			freq := num(row[1+len(labels)])
			totalFreq += freq
		}
		if totalFreq > 0 {
			for _, row := range rows {
				freq := num(row[1+len(labels)])
				w := freq / totalFreq
				for i := range labels {
					all[i] += w * num(row[1+i])
				}
			}
		}
		metaMap[k] = meta{
			AllStrat: all,
			Labels:   labels,
		}
	}

	p1 := map[string][][]any{}
	p2 := map[string][][]any{}
	for k, rows := range res.Data {
		if strings.HasPrefix(k, "P1-") {
			p1[k] = rows
		} else if strings.HasPrefix(k, "P2-") {
			p2[k] = rows
		}
	}

	pl := payload{
		P1:         p1,
		P2:         p2,
		Data:       res.Data,
		Config:     eval.OutputName(cfg.Evaluator),
		ConfigFile: cfg,
		Stats: stats{
			EV:             ev{Evs: []float32{res.EV1, res.EV2}, EV1: res.EV1, EV2: res.EV2},
			Exploitability: res.Exploitability,
			IsMeta:         metaMap,
			Solver: solverStats{
				Iteration:    cfg.Iterations,
				Total:        cfg.Iterations,
				Pot:          cfg.PotSize,
				TotalInfoSet: res.Infosets,
				NumPlayers:   2,
			},
		},
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(pl, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, b, 0o644)
}

func num(v any) float32 {
	switch x := v.(type) {
	case float32:
		return x
	case float64:
		return float32(x)
	default:
		return 0
	}
}
