# Pre-plan render matrix

The starting state this plan repairs, captured 2026-08-12 before any edit.

**Reproduce:** with `backdrop-studio` and `image-tools` running,

```bash
for s in $(backdrop-studio catalog list --json | jq -r '.styles[].id'); do
  backdrop-studio render submit --style "$s" --seed 7 --json >/dev/null 2>&1 \
    && echo "PASS $s" || echo "FAIL $s"
done
```

**Result: 4 pass, 12 fail** out of 16 seeded styles, with no brand bound —
which is what a CLI caller always had, because `render submit` had no way to
supply one.

| Style | Result | Detail |
|---|---|---|
| `ascii-field` | PASS | rendered 1600x1000 |
| `city-pop-horizon` | FAIL | render: treatment chain: image-tools: posterize returned 422 Unprocessable Entity: {"code":"invalid_request", "message":"treatments: invalid color \"$brand.primary\""} |
| `constructivist-figure` | FAIL | render: synthesized inference capability: image-tools inference returned no job id |
| `cyanotype-arcade` | FAIL | render: treatment chain: image-tools: duotone returned 422 Unprocessable Entity: {"code":"invalid_request", "message":"treatments: invalid color \"$brand.primary\""} |
| `demoscene-terrain` | FAIL | render: treatment chain: image-tools: dither_ordered returned 422 Unprocessable Entity: {"code":"invalid_request", "message":"treatments: invalid color \"$brand.primary\""} |
| `engraved-colonnade` | PASS | rendered 1600x1000 |
| `guided-botanical` | FAIL | render: guided inference capability: image-tools inference returned no job id |
| `memphis-weave` | FAIL | render: treatment chain: image-tools: posterize returned 422 Unprocessable Entity: {"code":"invalid_request", "message":"treatments: invalid color \"$brand.primary\""} |
| `op-art-interior` | PASS | rendered 1600x1000 |
| `riso-horizon` | FAIL | render: treatment chain: image-tools: dither_diffusion returned 422 Unprocessable Entity: {"code":"invalid_request", "message":"treatments: invalid color \"$brand.primary\""} |
| `solar-bloom-horizon` | FAIL | render: treatment chain: image-tools: scrim returned 422 Unprocessable Entity: {"code":"invalid_request", "message":"treatments: invalid color \"$brand.primary\""} |
| `stipple-massif` | PASS | rendered 1600x1000 |
| `swiss-contour` | FAIL | render: treatment chain: image-tools: posterize returned 422 Unprocessable Entity: {"code":"invalid_request", "message":"treatments: invalid color \"$brand.primary\""} |
| `technical-field` | FAIL | render: treatment chain: image-tools: duotone returned 422 Unprocessable Entity: {"code":"invalid_request", "message":"treatments: invalid color \"$brand.primary\""} |
| `ukiyo-tide` | FAIL | render: treatment chain: image-tools: posterize returned 422 Unprocessable Entity: {"code":"invalid_request", "message":"treatments: invalid color \"$brand.primary\""} |
| `vaporwave-drift` | FAIL | render: treatment chain: image-tools: scrim returned 422 Unprocessable Entity: {"code":"invalid_request", "message":"treatments: invalid color \"$brand.primary\""} |

## What the failures were

**Ten colour failures.** `imageengine.mergedParams` failed *open*: when the
palette lookup missed it wrote the literal slot string onto the wire, so
image-tools answered `422 invalid color "$brand.primary"`. Every Go unit test
passed throughout, because the one contract test resolved parameters against a
bound brand — the path a CLI caller never took.

**Two generation failures.** Recorded by the audit as a missing inference
capability. That attribution was wrong. image-tools' REST submit edge returns
`job_id` (protojson proto names) while its Connect edge returns `resultRef`
(camelCase); backdrop-studio decoded `jobId` and therefore discarded every job
id it was given. `sd-1.5/local-gpu` was generating in ~15s the whole time.

## After Phase 1

16 pass, 0 fail, with no brand bound; 16 pass, 0 fail with a brand bound. No
response body contains the string `$brand.`.
