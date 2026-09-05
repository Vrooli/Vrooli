# World weather

Weather belongs to the deterministic simulation. `WorldState.weather` stores
the current state, its expiry time, and smoothed swarm-health pressure. Seeded
transitions choose among `clear`, `cloudy`, `rain`, and `snow`; seasonal gates
can exclude snow.

Pressure combines recent failed runs, the share of actors in the failed state,
and expired gatherings. The smoothing lever prevents one event from changing
the sky abruptly. URL parameters can pin `weather` and `pressure` for tests and
diagnostics. Production operation uses the simulation value.

Each preset in `world.tuning.json.weather` supplies lighting multipliers, cloud coverage,
wetness, wind, and particle rate. The scene applies lighting and terrain
wetness, places the cloud plane at the configured altitude, and keeps the
particle field around the camera target. Quality profiles scale or disable
particles.

`world.tuning.json.weather` is the runtime authority. `weather.json` is a catalogue
copy checked against it by a test. Snow uses pale-blue `#c7e3f2` shadow variation;
the catalogue's former magenta value was stale, not the running world's palette.

The HUD displays `<state> — health pressure <percent>%` in both 3D and 2D mode.
This text explains that degraded swarm health can produce rain. Diagnostics
also stamp the active weather in `data-weather` for read-only workflow tests.
