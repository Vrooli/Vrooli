# data

Everything that talks to the server: the WorldService client, the feed stream
with reconnect, the fallback poll, the signal adapters, the actions the HUD
calls, and the runtime that steps the sim on a clock.

Import rule: `sim`, `config` and the generated proto clients. Never `scene`,
`hud` or `engine`.
