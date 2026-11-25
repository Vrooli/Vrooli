# Book art and print preparation

Image Tools owns the five book-art generation recipes and the deterministic
`print_prep` pixel operation. Their capability declarations live in
`api/internal/capability/manifests.seed.json`; generated catalogs and CLI
commands are derived from that file. Preset operation ids are
`book_art_line_art`, `book_art_picture_book`, `book_art_comic_ink`,
`book_art_cookbook`, and `book_art_cover_art`.

Each preset wraps the caller's subject prompt in a compositional/style block,
supplies a negative prompt, a 1024px canvas, a sampling step count, a fixed
default seed, and an ordered post-step declaration. Supplying a nonzero seed
overrides the fixed default. Book producers should derive a stable per-figure
seed from their own immutable figure record and pass it explicitly; reusing the
preset seed for every figure is deterministic but produces the same stochastic
starting state. Prompts reserve cover title space and prohibit generated
lettering; typeset copy remains outside the image.

## Consumer route

The supported stage contract is:

1. Submit the chosen `book_art_*` AI operation with the subject description and
   an explicit per-figure seed. The engine expands it to `text_to_image` while
   preserving normal image-tools admission, safety, model selection and job
   provenance.
2. Retrieve its retained output and submit `upscale` with `scale: 2` if the
   source resolution needs the declared 2× enlargement.
3. Submit `print_prep` with the book's trim box (`width_in`, `height_in`),
   `dpi: 300`, and a chosen `ink_mode`. Pass `threshold` for 1-bit ink.
4. Retain the returned PNG and `resolved_params` (`dpi`, `width_px`,
   `height_px`, `ink_coverage`) in the consuming book's asset manifest.

`print_prep` trims to the requested aspect ratio using `gravity` (center by
default), resamples with Lanczos to the exact physical box, writes a PNG
`pHYs` chunk for the requested DPI, and reports the fraction of output pixels
that carry ink. In `1bit` mode, non-paper pixels become black with source alpha;
paper pixels become transparent. The PNG uses RGBA storage as a transport for
this binary ink mask, so it is 1-bit in visual semantics, not in PNG bit depth.
`colour` mode preserves colour and alpha. The operation rejects missing trim
dimensions, DPI outside 72–1200, and output boxes above 100 million pixels.

## Current orchestration boundary

The capability manifest carries `post_steps` in order, but image-tools still
submits one durable operation per job. No current recipe runner automatically
chains a preset result into upscale and print prep. A consumer must execute the
four stages above and carry the returned blob reference between requests. The
`runs/niche-book-production-20261123/production` book engine must then record
the resulting artifact hash, model, prompt, explicit seed, rights, alt text,
and resolved print-prep report in its own figure manifest. Saved recipe replay
or a single orchestration job is the remaining integration gap.

No image was generated or model installed as part of this implementation.
