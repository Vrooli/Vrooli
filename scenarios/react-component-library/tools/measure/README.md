# React Component Library measurement harness

Run these scripts from the repository root while the scenario is running.
Browser scripts use `/usr/bin/google-chrome` at `http://localhost:23906`.

| Script | Measurement |
| --- | --- |
| `measure-story-roles.py` | Story roles, missing boundary stories, and enum-only boundaries. |
| `measure-story-counts.py` | Story counts, bare stories, and missing `args.fields`. |
| `measure-experience-claims.py` | Experience contracts, claims, and visual coverage. |
| `classify-gates.py` | Bookkeeping/perceptual gate counts and blocking status. |
| `measure-self-adoption.py` | UI files importing the published component library. |
| `measure-native-elements.py` | Native `<select>` and raw `<button>` usage. |
| `audit-painted-pixels.py` | Painted specimen nodes, dimensions, and verdicts. |
| `screenshot-routes.py` | Named route screenshots for retained evidence. |

The browser audit accepts `--output FILE` and otherwise writes
`/tmp/rcl-render-audit.json`. It uses a checked-in component list when one is
available and otherwise reads the live ComponentsService.
