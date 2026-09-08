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
05:00. Local lamp and hearth emission reaches zero, while phase-dependent moonlight
and a bounded ambient fill keep the terrain readable. The same cyclic envelope reveals
a procedural Milky Way and the remaining stars within the selected quality
budget. Sun height, cloud coverage, and moon illumination gate sky visibility.
Named fixed lighting presets retain their ordinary appearance; Deep night
uses the shared clock so the atmosphere, wildlife, and events seek together.

The visible moon has a textured grey surface and an illuminated phase derived
from the presentation date. Its unlit side blends into the sky. The directional
key follows the same sun/moon directions as the visible bodies. Moonlight fades
with the square of the illuminated fraction, cloud cover, and proximity to the
horizon; a moon below the horizon casts no direct light. Ambient fill remains
available on moonless nights. Sun height also drives the late-evening transition
to night sky and fog colors, so a low sun cannot leave a bright pink night sky.
Shadow refreshes follow bounded clock intervals and explicit time changes.

During the central quiet hours, an additional ordinary meteor opportunity
every 90 seconds supplements the existing 180-second stream. The base event
identities and rare-fireball cooldown remain unchanged. The combined presenter
still admits at most two meteors; disabled ambient life, weather suppression,
and reduced motion also gate the extra stream. Seeking never replays a backlog.

Campfires combine the retained log prop with a stone ring, crossed flame cards,
a separate ember bed and rising smoke. Flames stop before the last embers cool;
smoke follows active flames. Wet weather extinguishes exposed fires. Frozen time
retains a stable pose, while reduced motion suppresses moving smoke and retains
static flames/embers. Effects do not block walking or agent picking. Final visual
tuning and combined large-world performance qualification remain pending.
