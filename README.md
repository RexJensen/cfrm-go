# cfrm-go

A minimal Go CFRM solver for fixed limit single street spots.
Currently supports 2 toy game evaluators (1-card lowball, 1-card high).

## Motivations

There's a lot more excitement and activity in the home grown solver world. I hope this code gets some people (and their coding agents) a starting place.

## Disclaimer

This code was entirely written by a coding agent (gpt-codex-5.3-medium) and based on prompts, code snippets, and notes from my personal research on CFRM and solving mixed games.

https://proceedings.neurips.cc/paper/2007/file/08d98638c6fcd194a4b1e6992063e944-Paper.pdf

## Recommended Usage

Start here, have your agent (or code yourself) extensions on the evaluators... Build some cool UIs, and let me know how far along you get! I'll see you on the felt @ equilibrium.

## Run

```bash
go run ./cmd/cfrm -config config.json
```

Shows a terminal progress bar and writes `output/output.json`.

## Input format

`config.json` uses all-caps keys:

```json
{
  "POT_SIZE": 3,
  "ITERATIONS": 10000,
  "OUTPUT": "output.json",
  "EVALUATOR": "LOW_CARD",
  "TREE": {
    "STREET_RIVER_MAX_BETS": 4,
    "P1_RANGE": "A:1,K:1,...,2:1",
    "P2_RANGE": "A:1,K:1,...,2:1"
  }
}
```

- `EVALUATOR`: `HIGH_CARD` or `LOW_CARD`
- `STREET_RIVER_MAX_BETS`: max bet level allowed in the river toy game
- `P1_RANGE` / `P2_RANGE`: inline rank-weight ranges

## Output format

The solver writes JSON to `output/output.json` with this shape:

- `p1`: player-1 infoset buckets only
- `p2`: player-2 infoset buckets only
- `data`: all infoset buckets (`p1` + `p2`)
- `config`: evaluator name string (`evaluateHighCardAceLow` or `evaluateLowCardAceLow`)
- `configFile`: resolved config used for the run
- `stats`:
  - `ev`: overall EV summary (`evs`, `ev1`, `ev2`)
  - `exploitability`: BR-based exploitability:
    - `p1`, `p2`, `total`
    - `p1_pct`, `p2_pct`, `total_pct`
    - `br.p1`, `br.p2` (each has `evs`, `ev1`, `ev2`)
  - `isMeta`: per-infoset metadata (`allStrat`, `labels`)
  - `solver`: run stats (`iteration`, `total`, `pot`, `total_infosets`, `num_players`)

Each infoset bucket in `p1`, `p2`, and `data` maps to rows by hand:

```json
[
  "A",
  0.72,
  0.28,
  1.0,
  0.61,
  0.39
]
```

Row layout is:

1. `hand` (rank symbol)
2. strategy probabilities for each action in that infoset, in `stats.isMeta[infoset].labels` order
3. normalized frequency (max row frequency in bucket = 1.0)
4. `ev1` for this hand+infoset
5. `ev2` for this hand+infoset
