# Innerscene monitor screen-map v2

This sibling derivative package leaves the acquired and normalized v1 source package unchanged. It starts from the pinned v1 GLB and partitions the `Display glass` mesh into its two planar front triangles and the remaining 298 bevel triangles. A dedicated display-only `TEXCOORD_0` stream maps the four face corners to `[0,1]`; all original `POSITION`, `NORMAL`, UV0 accessor, and embedded-image bytes remain in the source BIN prefix. The original material record remains unchanged in JSON, and bevel references keep their original UV stream and neutral glass material. The screen primitive uses an exact material clone so Device Studio can replace only that slot with the app image.

The derivative is generic Innerscene CC0 geometry, not a named monitor. Active pixel bounds, manufacturer identity, physical dimensions, and physical qualification remain unknown. The display face is a glass proxy, not independently measured active pixels.

Recreate and verify the output from the repository root:

```sh
python3 scenarios/device-studio/assets/sources/innerscene-ultrawide-monitor-screenmap-v2/derive_monitor_screen_split.py
```

The script pins the original source SHA, v1 derivative SHA, source triangle/vertex identities, and source material chain. It verifies that 2 display triangles and 298 bevel triangles exactly partition the 300 source triangles, that the source BIN prefix is preserved, and that only the new screen UV stream adapts the display corners. The output and exact topology receipt are pinned in `monitor-screen-split-v2-manifest.json`.
