package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/chanian/cfrm-go/internal/config"
	"github.com/chanian/cfrm-go/internal/output"
	"github.com/chanian/cfrm-go/internal/solver"
	"github.com/chanian/cfrm-go/internal/tree"
)

func main() {
	configPath := flag.String("config", "config.json", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		panic(err)
	}

	builder := tree.Builder{
		PotSize: cfg.PotSize,
		MaxBets: cfg.Tree.StreetRiverBets,
	}
	root, err := builder.Build()
	if err != nil {
		panic(err)
	}

	lastPct := -1
	res, err := solver.RunWithProgress(cfg, root, func(iteration, total int) {
		if total <= 0 {
			return
		}
		pct := int(float64(iteration) * 100 / float64(total))
		if pct == lastPct && iteration < total {
			return
		}
		lastPct = pct
		const width = 30
		filled := int(float64(iteration) / float64(total) * width)
		if filled > width {
			filled = width
		}
		bar := strings.Repeat("=", filled) + strings.Repeat(" ", width-filled)
		fmt.Printf("\r[%s] %3d%% (%d/%d)", bar, pct, iteration, total)
		if iteration == total {
			fmt.Print("\n")
		}
	})
	if err != nil {
		panic(err)
	}

	out := filepath.Join("output", cfg.Output)
	if err := output.Write(out, cfg, res); err != nil {
		panic(err)
	}

	if res.Exploitability != nil {
		fmt.Printf(
			"wrote %s (infosets=%d, ev1=%.6f, ev2=%.6f, exploitability_p1=%.6f, exploitability_p2=%.6f, exploitability_total=%.6f, exploitability_total_pct=%.6f)\n",
			out,
			res.Infosets,
			res.EV1,
			res.EV2,
			res.Exploitability.P1,
			res.Exploitability.P2,
			res.Exploitability.Total,
			res.Exploitability.TotalPct,
		)
		return
	}
	fmt.Printf("wrote %s (infosets=%d, ev1=%.6f, ev2=%.6f)\n", out, res.Infosets, res.EV1, res.EV2)
}
