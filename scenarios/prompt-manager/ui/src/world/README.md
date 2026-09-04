# world

The 3D world module. Six layers with one-way dependencies, enforced by ESLint
(`import/no-restricted-paths`, see `eslint.config.js`) and proven by the
fixtures in `sim/__lint__` through `__lint__/layerRule.test.ts`.

| Layer | May import | Owns |
|---|---|---|
| `config` | nothing (zod only) | tuning, scene records, biome sets, weather presets |
| `sim` | `config` | seeded terrain, water, biomes, sites, navigation, actors, weather and views |
| `engine` | `config` | canvas, lighting rig, post chain, camera rig, quality governor, assets, diagnostics |
| `scene` | `engine`, `sim`, `config` | R3F views: terrain tiles, water, places, biome props, actors, weather and labels |
| `hud` | `sim`, `config`, `data` | summary strip, agent card, team panel, ticker, settings, 2D mode, editor chrome |
| `data` | `sim`, `config`, generated proto clients | WorldService client, feed stream, fallback poll, actions, runtime |

`index.tsx` is the route component and the only module that composes `scene`
and `hud`. Nothing outside `src/world` imports `scene` or `hud`.

Architecture: `../../docs/concepts/WORLD-ARCHITECTURE.md`. Levers:
`../../docs/reference/configuration.md` (world tuning section). Sim rules:
`../../docs/concepts/WORLD-SIM.md`. HUD: `../../docs/concepts/WORLD-HUD.md`.
Assets: `../../docs/guides/WORLD-ASSETS.md`.
Terrain: `../../docs/concepts/WORLD-TERRAIN.md`. Weather:
`../../docs/concepts/WORLD-WEATHER.md`.
