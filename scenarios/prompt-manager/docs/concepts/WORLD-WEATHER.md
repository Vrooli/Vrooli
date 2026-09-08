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
wetness and keeps the particle field around the camera target. A camera-centred
sky blends the resolved horizon and zenith colors and draws shaped procedural
clouds; quality profiles can disable clouds and scale or disable particles.
The HDR image supplies reflections, not the visible sky background.

`world.tuning.json.weather` is the runtime authority. `weather.json` is a catalogue
copy checked against it by a test. Snow uses pale-blue `#c7e3f2` shadow variation;
the catalogue's former magenta value was stale, not the running world's palette.

The HUD displays `<state> — health pressure <percent>%` in both 3D and 2D mode.
This text explains that degraded swarm health can produce rain. Diagnostics
also stamp the active weather in `data-weather` for read-only workflow tests.

## Civil time and quiet hours

World Settings exposes Midday, Evening, and Deep night presets, pause/play,
return to live time, and an expandable exact UTC input. Presets select civil
time in the configured clock timezone, freeze the presentation clock, and
switch lighting to follow that clock. Agent activity is independent of this
presentation override. A skipped spring-DST minute advances to the next valid
hour; explicit UTC remains available for exact instants.

Quiet hours fade in from 23:00 to 01:00, hold until 04:00, and fade out by
05:00. Local lamp and hearth emission reaches zero, while moon-colored key
and ambient fill keep the terrain readable. The same cyclic envelope reveals
a procedural Milky Way and the remaining stars within the selected quality
budget. Sun height, cloud coverage, and moon illumination gate sky visibility.
Named fixed lighting presets retain their ordinary appearance; Deep night
uses the shared clock so the atmosphere, wildlife, and events seek together.

During the central quiet hours, an additional ordinary meteor opportunity
every 90 seconds supplements the existing 180-second stream. The base event
identities and rare-fireball cooldown remain unchanged. The combined presenter
still admits at most two meteors; disabled ambient life, weather suppression,
and reduced motion also gate the extra stream. Seeking never replays a backlog.

The current campfire prop remains logs with coordinated emission. Dedicated
flames and a distinct ember stage, final visual tuning, and combined large-world
performance qualification remain part of the scene redesign.
