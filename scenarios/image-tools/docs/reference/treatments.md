# Deterministic treatment operations

The treatment family is a model-free extension of the `image-tools ops`
surface. Treatments are pure pixel transforms: they do not download weights,
call `ai-gateway`, read the clock, or use process-global randomness.

The seven tier-1 operations are:

| Operation | Purpose | Reproducibility input |
|---|---|---|
| `duotone` | Linear-light luminance mapped to two or three inks | parameters |
| `posterize` | Fixed-level luminance quantization | parameters |
| `halftone` | Rotated dot screen | parameters |
| `dither_ordered` | 4×4 Bayer threshold screen | parameters |
| `dither_diffusion` | Floyd–Steinberg error diffusion | parameters |
| `grain` | Film-like seeded noise | parameters + seed |
| `scrim` | Directional contrast wash | parameters |

The golden fixtures in `api/internal/treatments/testdata/golden/` are
byte-compared in separate executions. Higher-level scenarios should resolve
brand ink values before submitting these operations; an unresolved palette
slot is an error, never an implicit default.
