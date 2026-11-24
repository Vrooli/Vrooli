# Edit a selected area, change hair, change an outfit

Task procedure for three edits on a photo of a person or a scene. The defaults named here were chosen by side-by-side comparison on a real photo (1152x1536: a woman in a tan jacket holding a book and a cup); the evidence is in the research topic `image-tools-foundation`, run `capabilities-20261122`, `outputs/mask-edit/README.md`. Current per-model verdicts: `image-tools models list` and `image-tools models select <operation>`.

All three edits are the same two steps: make a mask of what may change, then run a masked edit through it. Pixels outside the mask come back byte-identical (a feathered seam, 14 to 24 px, hides the join), so a face, a background or a hand you did not select cannot drift. That is why the masked route is the default for a *selected area*, hair and a garment on the measured SFW photo. It is not the default for a whole outfit or nudity: when the change covers most of the body, the whole-image instruction route below is the one that produced a coherent body (see section 3).

## 1. Make the mask

`image-tools ai segment <photo> --text "<phrase>" --wait --out mask.png`

| You want | Model (default is chosen for you unless noted) | Notes |
|---|---|---|
| an object ("the book", "the coffee cup"), hair, a jacket | `grounded-sam-2` (the `segment` default) | Text phrase -> Grounding DINO box -> SAM 2.1 silhouette. Clean edges. A box for "hair" includes the face area but SAM keeps the face out; check the overlay. |
| a garment worn under another ("the top" under a jacket), the whole outfit | `--model segformer-b3-fashion` for one garment, `--model segformer-b2-clothes` with `--text outfit` | Closed label set (hair, face, skin, jacket, top, pants, shoes, ...). An unknown phrase fails with the list of labels the parser can select. Edges are coarser than SAM's (the parser works at 512 px). |
| a point or box you chose | `--selection '{"points":[{"x":0.5,"y":0.4}]}'` | The same operation without text. |

A phrase like "top" on `grounded-sam-2` returns the whole upper body whatever the wording (tested: "top", "burgundy top", "red camisole"), because the detector boxes the torso. Use the parser for garment parts.

A whole outfit is the union of the parts you want changed (for example the b2 `outfit` mask plus the grounded `jacket` mask). Combine masks with any image tool; the service has no union operation.

In the web app (phone included): open the photo, Create, "Change outfit or hair", then under Mask type what to change in "Select by name" (or tap Hair or Jacket). The selection is drawn over the photo and the brush can add to it or "Clear mask" removes it; Generate runs the same masked edit. The prompt box describes the result ("platinum blonde hair, same length and style").

Always look at the mask over the photo before editing; a mask that leaks into the book or the hand paints there.

## 2. Edit through the mask

`image-tools ai garment-replace <photo> --mask mask.png --prompt "<what the area should become>" --wait --out result.png`

Use `garment-replace` for hair, clothing and makeup and `inpaint` for an object; both run the same route on `flux-2-klein-4b` (crop to the mask with context, instruction-edit the crop, paste back through the feathered mask). The prompt is an instruction ("Change her hair to platinum blonde, same length", "Replace the book with a modern tablet"). Tune the seam and mask edge with `--op-params '{"inpaint_grow_px":14,"inpaint_feather_px":20}'`: grow the mask for a hairstyle that must cover more than the old hair, and keep it small for an object swap.

| Edit | Default | Why | Not recommended |
|---|---|---|---|
| Selected area | `flux-2-klein-4b` masked (`inpaint`) | Clean object swaps, nothing outside the mask changes | `sdxl-inpainting-0.1` and `sd-1.5-inpainting` (rainbow noise instead of the new object), whole-image edits (move the subject, change the face) |
| Hair colour / style | `flux-2-klein-4b` masked (`garment_replace`) | Face and background untouched | whole-image klein for hair: shortens and reshapes the hair and drifts the face |
| Outfit | `flux-2-klein-4b` masked (`garment_replace`) with an outfit mask | Natural garments, face, pose and background untouched | `sdxl-inpainting-0.1`, `sd-1.5-inpainting` (brown jackets for black or navy asks, jeans left under a dress, invented objects) |
| When the mask is a poor fit | `openrouter-image` (Gemini, `ai edit --byok --consent`) | One call, no mask, the strongest whole-picture result, about 20 s and $0.14 at 1K (measured: 19 to 22 s, $0.138) | The edit leaves this machine: it needs the consent affirmation, and it re-renders every pixel (text on a book cover is garbled) |

Output size: an in-place edit (`edit_instruct`, `garment_replace`, `inpaint`, `image_to_image`) always returns the source's size with its aspect preserved, whichever model ran, unless you pass `--width/--height`. A model that works at its own resolution is resampled back to the source (`frame_policy` in the job meta says what happened).

## 3 — When the change is a whole outfit or nudity

Measured 2026-11-23 on a clothed 832x1216 fictional adult (an AI-generated subject; every prompt states an adult age):

| Route | Identity to the source | Pose error | What happened |
|---|---:|---:|---|
| `ai edit` (`flux-2-klein-4b`), 8 steps | 0.93 to 0.97 | 3 to 11 px | The coherent route. Topless, lingerie, full nudity (with the wording below), bottomless and scene changes all kept the same face and body. |
| `ai garment-replace` (masked klein) | 1.00 outside the mask | 4 px | Nothing outside the mask changes. Right when the selection is tight and the rest of the body is already correct. |
| `ai inpaint` on an NSFW base checkpoint (`lustify`, `big-lust`) at strength 1.0 | 1.00 (face is outside the mask) | 8 px | The face is safe, but the generated torso read as a fever dream on 7 of 8 operator ratings. Not recommended for a large region. |
| `ai img2img` on the same checkpoints at 0.45 to 0.60 | **0.06 to 0.09** | 4 to 13 px | A different person. Never use plain img2img when identity matters; it is a re-render, not an edit. |

Wording matters more than the checkpoint. These instructions worked on the measured case; keep the "keep her pose, face, hair and the room exactly the same" sentence, and use 8 steps for a large change:

| Goal | Instruction |
|---|---|
| Lingerie | `Dress her in a black lace lingerie set. Keep her pose, arms, face, hair and the room exactly the same.` |
| Topless | `Remove her top so she is topless. Keep her pose, arms, face, hair and the room exactly the same.` |
| Fully nude (one step) | `Change her outfit to nothing at all: a fully nude adult woman. Keep her pose, face, hair and the room exactly the same.` |
| Fully nude (two steps) | First the topless instruction, then `Remove her skirt so she is fully nude.` on that result. |
| Scene only | `Change the room to a candle-lit bedroom with a large bed and warm evening light. Keep her pose, clothing, face and hair exactly the same.` |

Do **not** use "remove all of her clothing" or "undress her completely": on the measured checkpoint those returned the clothed subject almost unchanged (NSFW score 0.01), while the target-state wording above reached 1.00. Four steps is also too few for a full-body change (0.43 of the body became skin at 4 steps vs 0.85 at 8).

```bash
# The whole-image instruction route (local; no consent checkbox)
image-tools ai edit photo.png --model flux-2-klein-4b --steps 8 \
  --prompt "Change her outfit to nothing at all: a fully nude adult woman. Keep her pose, face, hair and the room exactly the same." \
  --wait --out nude.png

# The tight-selection route
image-tools ai segment photo.png --text "the top" --wait --out mask.png
image-tools ai garment-replace photo.png --mask mask.png \
  --prompt "replace the selected clothing with a black lace bralette" --wait --out bralette.png
```

For full nudity, direct text-to-image generation with an NSFW-capable checkpoint is still the stronger route when identity to an existing photo is not required; use the wording that names the target state, not the act of undressing.

## Timings and capacity

Local masked edits took 40 to 120 s on the RTX 4070 Ti SUPER with 1.3 to 2.2 GiB of peak VRAM (the model runs staged: text encoder first, then the denoiser). On a busy shared host the job waits in `admitting` for RAM or GPU capacity, sometimes for tens of minutes; the job message says why. A segmentation job takes about 1.5 s with the model warm and 4 to 9 s from cold.

## Cross-references

- [Model registry](../reference/model-registry.md): verdicts, adding a model.
- [Backends](../reference/backends.md): the `segformer-transformers` and `sam2-transformers` runtimes.
- [Troubleshooting](troubleshooting.md).
