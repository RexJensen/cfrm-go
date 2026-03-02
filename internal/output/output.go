package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chanian/cfrm-go/internal/config"
	"github.com/chanian/cfrm-go/internal/eval"
	"github.com/chanian/cfrm-go/internal/solver"
)

type meta struct {
	AllStrat vec      `json:"allStrat"`
	Labels   []string `json:"labels"`
}

type ev struct {
	Evs vec     `json:"evs"`
	EV1 float32 `json:"ev1"`
	EV2 float32 `json:"ev2"`
}

type brOut struct {
	P1 ev `json:"p1"`
	P2 ev `json:"p2"`
}

type exploitabilityOut struct {
	P1       float32 `json:"p1"`
	P2       float32 `json:"p2"`
	Total    float32 `json:"total"`
	P1Pct    float32 `json:"p1_pct"`
	P2Pct    float32 `json:"p2_pct"`
	TotalPct float32 `json:"total_pct"`
	BR       brOut   `json:"br"`
}

type solverStats struct {
	Iteration          int     `json:"iteration"`
	Total              int     `json:"total"`
	Pot                float32 `json:"pot"`
	TotalInfoSet       int     `json:"total_infosets"`
	TotalDecisionNodes int     `json:"total_decision_nodes"`
	NumPlayers         int     `json:"num_players"`
}

type stats struct {
	EV             ev                 `json:"ev"`
	Exploitability *exploitabilityOut `json:"exploitability"`
	IsMeta         map[string]meta    `json:"isMeta"`
	Solver         solverStats        `json:"solver"`
}

type payload struct {
	Data       map[string][]row `json:"data"`
	Config     string           `json:"config"`
	ConfigFile config.Config    `json:"configFile"`
	Stats      stats            `json:"stats"`
}

type row []any

type vec []float32

func (v vec) MarshalJSON() ([]byte, error) {
	return json.Marshal([]float32(v))
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
			AllStrat: vec(all),
			Labels:   labels,
		}
	}

	var exploitability *exploitabilityOut
	if res.Exploitability != nil {
		exploitability = &exploitabilityOut{
			P1:       res.Exploitability.P1,
			P2:       res.Exploitability.P2,
			Total:    res.Exploitability.Total,
			P1Pct:    res.Exploitability.P1Pct,
			P2Pct:    res.Exploitability.P2Pct,
			TotalPct: res.Exploitability.TotalPct,
			BR: brOut{
				P1: toEV(res.Exploitability.BR.P1),
				P2: toEV(res.Exploitability.BR.P2),
			},
		}
	}

	pl := payload{
		Data:       toRows(res.Data),
		Config:     eval.OutputName(cfg.Evaluator),
		ConfigFile: cfg,
		Stats: stats{
			EV:             ev{Evs: vec{res.EV1, res.EV2}, EV1: res.EV1, EV2: res.EV2},
			Exploitability: exploitability,
			IsMeta:         metaMap,
			Solver: solverStats{
				Iteration:          cfg.Iterations,
				Total:              cfg.Iterations,
				Pot:                cfg.PotSize,
				TotalInfoSet:       res.Infosets,
				TotalDecisionNodes: len(metaMap),
				NumPlayers:         2,
			},
		},
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	b, err := marshalPayload(pl)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, b, 0o644)
}

func marshalPayload(pl payload) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("{\n")
	if err := writeDataBlock(&buf, pl.Data); err != nil {
		return nil, err
	}
	buf.WriteString(",\n")

	aux := struct {
		Config     string        `json:"config"`
		ConfigFile config.Config `json:"configFile"`
		Stats      stats         `json:"stats"`
	}{
		Config:     pl.Config,
		ConfigFile: pl.ConfigFile,
		Stats:      pl.Stats,
	}
	auxBytes, err := json.MarshalIndent(aux, "  ", "  ")
	if err != nil {
		return nil, err
	}
	auxBytes = compactNamedVectors(auxBytes, "evs", "allStrat", "labels")
	if len(auxBytes) < 4 {
		return nil, fmt.Errorf("unexpected aux json length")
	}
	buf.Write(auxBytes[4 : len(auxBytes)-2])
	buf.WriteString("\n}\n")
	return buf.Bytes(), nil
}

func compactNamedVectors(in []byte, keys ...string) []byte {
	lines := strings.Split(string(in), "\n")
	set := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		set[k] = struct{}{}
	}
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		matched := false
		for k := range set {
			token := `"` + k + `": [`
			idx := strings.Index(line, token)
			if idx < 0 {
				continue
			}
			matched = true
			prefix := line[:idx] + token
			j := i + 1
			vals := make([]string, 0, 8)
			for ; j < len(lines); j++ {
				t := strings.TrimSpace(lines[j])
				if t == "]" || t == "]," {
					break
				}
				vals = append(vals, strings.TrimSuffix(t, ","))
			}
			if j >= len(lines) {
				out = append(out, line)
				break
			}
			suffix := "]"
			if strings.TrimSpace(lines[j]) == "]," {
				suffix = "],"
			}
			out = append(out, prefix+strings.Join(vals, ",")+suffix)
			i = j
			break
		}
		if !matched {
			out = append(out, line)
		}
	}
	return []byte(strings.Join(out, "\n"))
}

func writeDataBlock(buf *bytes.Buffer, data map[string][]row) error {
	buf.WriteString("  \"data\": {\n")
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		keyJSON, err := json.Marshal(k)
		if err != nil {
			return err
		}
		buf.WriteString("    ")
		buf.Write(keyJSON)
		buf.WriteString(": [\n")
		rows := data[k]
		for r := range rows {
			rowJSON, err := json.Marshal([]any(rows[r]))
			if err != nil {
				return err
			}
			buf.WriteString("      ")
			buf.Write(rowJSON)
			if r < len(rows)-1 {
				buf.WriteString(",")
			}
			buf.WriteString("\n")
		}
		buf.WriteString("    ]")
		if i < len(keys)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString("  }")
	return nil
}

func toRows(in map[string][][]any) map[string][]row {
	out := make(map[string][]row, len(in))
	for k, rows := range in {
		list := make([]row, len(rows))
		for i, r := range rows {
			list[i] = row(r)
		}
		out[k] = list
	}
	return out
}

func toEV(in solver.EVStats) ev {
	return ev{
		Evs: vec(in.Evs),
		EV1: in.EV1,
		EV2: in.EV2,
	}
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
