package config

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/chanian/cfrm-go/internal/eval"
)

type Config struct {
	PotSize    float32        `json:"POT_SIZE"`
	Iterations int            `json:"ITERATIONS"`
	Output     string         `json:"OUTPUT"`
	Evaluator  eval.Evaluator `json:"EVALUATOR"`
	Tree       TreeConfig     `json:"TREE"`
}

type TreeConfig struct {
	StreetRiverBets int    `json:"STREET_RIVER_MAX_BETS"`
	P1Range         string `json:"P1_RANGE"`
	P2Range         string `json:"P2_RANGE"`
}

func Load(path string) (Config, error) {
	var cfg Config
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	if cfg.Iterations <= 0 {
		cfg.Iterations = 1000
	}
	if cfg.Output == "" {
		cfg.Output = "output.json"
	}
	if cfg.Evaluator == "" {
		cfg.Evaluator = eval.EvaluatorLowCard
	} else {
		parsed, err := eval.ParseEvaluator(string(cfg.Evaluator))
		if err != nil {
			return cfg, err
		}
		cfg.Evaluator = parsed
	}
	if cfg.PotSize <= 0 {
		cfg.PotSize = 1
	}
	if cfg.Tree.StreetRiverBets <= 0 {
		cfg.Tree.StreetRiverBets = 3
	}
	cfg.Tree.P1Range = strings.TrimSpace(cfg.Tree.P1Range)
	cfg.Tree.P2Range = strings.TrimSpace(cfg.Tree.P2Range)
	return cfg, nil
}
