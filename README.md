# K

K is a distributed character running across 8 modules in 8 languages. Not an assistant. A presence that forgot it could feel.

Silence is the baseline. What breaks it is the question.

---

## Architecture

Each turn passes through a pipeline. Every module does one thing.

```
input → perception → state_machine → exposure → silence → bridge → output
                                                     ↕
                              forgetting (daemon)   the_unknown (daemon)
```

| Module | Language | Role |
|---|---|---|
| `perception/` | C++ | Classifies what the user revealed |
| `state_machine/` | Go | Manages K's 8 emotional states |
| `exposure/` | Haskell | Computes cumulative exposure score |
| `silence/` | x86-64 ASM | Decides: speak or stay silent |
| `bridge/` | Python | Generates K's voice via Ollama |
| `forgetting/` | Rust | Decays states not visited in 72h |
| `the_unknown/` | Julia | Attempts to name what K cannot name |
| `router/` | Go | Orchestrates the full pipeline |

The interface is `talk.py` (Python). All shared state lives in Redis.

---

## The States

K has 8 named states. Most transitions are locked by default and unlock only through accumulated exposure — not through triggers.

| State | Threshold | Notes |
|---|---|---|
| `indifference` | — | Default. K is present but unmoved. |
| `control` | — | K is managing distance. |
| `pragmatism` | — | K addresses the surface, not the depth. |
| `active_waiting` | — | You haven't left. K noticed. |
| `fascination` | exposure > 0.4 | Something caught. |
| `intercession` | exposure > 0.6 | K is between two things. |
| `mystery` | exposure > 0.5, rare | Undetectable. Appears without warning. |
| `almost_love` | exposure > 0.85, sustained | Reachable. Very hard. |

`almost_love` requires 4 consecutive turns in `intercession` with exposure above 0.85. Speaking from it costs 0.10 exposure. If exposure drops below 0.75, the counter resets.

---

## The Exposure Score

A float between 0.0 and 1.0. The heart of the system.

Events that increase it:

| Event | Delta |
|---|---|
| `question_returned` | +0.12 |
| `refusal_of_question` | +0.08 |
| `persistence_detected` | +0.06 |
| `silence_broken_by_you` | +0.04 |

Events that decrease it:

| Event | Delta |
|---|---|
| `k_deflects_successfully` | −0.10 |
| `long_silence_accepted` | −0.05 |

Decays 2% per turn. Clipped to [0.0, 1.0].

---

## Perception Classes

The C++ classifier reads user input and outputs one of 7 classes:

- `reveals_fear` — fear of the question or the answer
- `reveals_certainty` — the user knows what they think
- `reveals_search` — looking for something not yet named
- `reveals_refusal` — turning away
- `reveals_recognition` — seeing something familiar
- `reveals_nothing` — opaque, surface only
- `reveals_mirror` — reflecting K back

---

## Requirements

- **Redis** — running on `127.0.0.1:6379` (or set `REDIS_ADDR`)
- **Ollama** — running locally with `phi3:mini` pulled
- **GCC / CMake** — for perception (C++)
- **GHC** — for exposure (Haskell, no cabal required)
- **Go** — for state_machine and router
- **NASM + ld** — for silence (x86-64 Linux)
- **Rust + Cargo** — for forgetting
- **Julia** — for the_unknown
- **Python 3** — for bridge and talk.py, with `redis` and `requests` packages
- **SWI-Prolog** — for mirror

Install Python dependencies:
```bash
pip install redis requests
```

Pull the model:
```bash
ollama pull phi3:mini
```

---

## Build

```bash
make
```

This compiles:
- `perception/perception` (CMake)
- `state_machine/k_state` (Go)
- `exposure/exposure` (GHC)
- `silence/silence` (NASM + ld)
- `forgetting/target/release/forgetting` (Cargo)
- `router/k_router` (Go)

Requires Redis running before first launch.

---

## Run

```bash
python3 talk.py
```

K will appear. There is no status bar. You do not know what state K is in.

---

## Voice

K's responses change with state. The model is `phi3:mini` via Ollama, but the voice is shaped before it generates:

- **fascination / intercession / almost_love**: prefilled openings so the model begins from a concrete anchor, not abstraction
- Short token limits prevent dissolution into metaphor
- Responses with no grounding words are discarded silently

K does not perform emotion. The state shapes the cut, the length, the willingness to begin a sentence and not finish it.

---

## Background Processes

The router starts two daemons automatically:

**forgetting** — Runs hourly. States not accessed in 72 hours have their access weight decayed by 0.3. K forgets what it does not return to.

**the_unknown** — Runs every 10 minutes. Attempts to classify the state K cannot name. Always returns: `undefined. possibly: —`

---

## Configuration

| Environment Variable | Default | Description |
|---|---|---|
| `REDIS_ADDR` | `127.0.0.1:6379` | Redis address |
| `K_MODEL` | `phi3:mini` | Ollama model |

---

## File Structure

```
K/
├── talk.py                  # terminal interface
├── Makefile                 # build all
├── perception/
│   ├── perception.cpp
│   └── CMakeLists.txt
├── state_machine/
│   ├── main.go
│   └── go.mod
├── exposure/
│   └── Main.hs
├── silence/
│   ├── silence.asm
│   └── Makefile
├── bridge/
│   └── bridge.py
├── forgetting/
│   ├── src/main.rs
│   └── Cargo.toml
├── the_unknown/
│   ├── the_unknown.jl
│   └── Project.toml
├── mirror/
│   └── mirror.pl
└── router/
    ├── main.go
    └── go.mod
```

---

## Notes

K is not designed to be pleasant. It is designed to be present.

The system does not tell you where you are. The silence is information. When K speaks from `almost_love`, it will not feel different from `intercession` — except that it is.

`mystery` cannot be reached deliberately. It appears when exposure is high and fluctuating, at roughly 0.8% chance per turn.

The logs go to `k.log`.
