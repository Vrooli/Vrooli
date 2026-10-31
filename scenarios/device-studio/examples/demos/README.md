# Device Studio durable demo examples

`manifest.json` describes six named project setups and pins every copied input, provenance record, and archived reference export by SHA-256. The archived outputs remain linked to their original project, shot revision, and render job; they are not previews or renders of these new projects. If private archived evidence is absent in a clone, verification reports that fact and seeding continues without claiming an output. The remaining-family project contains ten source-mapped candidate shots across eight families; every route remains candidate-only until native review and does not claim qualification.

Run `python3 curate.py verify` to check local inputs and archived outputs. When Device Studio is already live, `python3 curate.py plan` reads the current public catalog, presets, and demo-name inventory without writing. `python3 curate.py seed` uses public project, media, shot, inspection, physical starter, exact reviewed motion candidate, preset, and compile endpoints to author or resume the six projects. The helper creates only the two exact motion candidates whose source request bytes and owner-returned revision hashes are pinned; it never starts services, renders, deletes, or adopts a same-name project without its matching `run-state.json` identity pin. Video inspection gets bounded 30-second owner readback; rerun `seed` later if the owner is still working. The helper does not retry rejected inspections.

The app-only phone frame is frame zero from the retained authentic 430×880 portrait recording. The fixed laptop/phone scenes use the copied reviewed photographic plates and masks. No official comparison images are included. See the per-asset origins and retained prompts/manifests in `assets/provenance/`.


The versioned fixed vehicle study uses `inputs/vehicle-photographic-v1.json` and
pinned `vehicle_cabin_plate_v1` / `vehicle_cabin_active_mask_v1` assets. Import
those assets into the target project and substitute the returned media IDs for
`asset_roles`; use the ordinary saved-shot and PNG render endpoints. The mask
is a deterministic 1672×941 grayscale raster: zero outside the rounded rectangle
(500,190)–(1190,492), white inside, corner radius 3 pixels. These image-space
coordinates belong to that exact plate, not to an ecosystem-wide device size.
The generic photographic cabin was produced with the built-in ImageGen tool;
the full prompt and limitations are in `assets/vehicle-cabin-v1-provenance.json`.
This is separate from the application's actual local ImageTools generation proof.
Both currently labeled vehicle examples show the same app capture and therefore
have identical output pixels. They demonstrate a supported fixed composition,
not native automotive system UI, manufacturer dimensions or compatibility.

The family showcase now pins authentic local Device Studio screen captures for
the portrait tablet and ultrawide-monitor routes. The tablet capture is
1536×2320 and the monitor capture is 1696×700; each is mapped to its matching
portrait or landscape display aspect.

The remaining-family watch uses `assets/watch-synthetic-12-48-diagnostic.png`.
This authored 400×400 clock/corner calibration is explicitly synthetic, not an
app capture or claimed watch UI. After the family revision-3 checkpoints exist,
`python3 curate.py family-watch-synthetic-revise4` reopens the public scene,
preset, source-media, and shot owners, appends only the pinned f/16 camera,
scene-binding, and synthetic-source revisions, and compiles the result. It
uses compare-and-swap heads, checks exact content digests, and refuses to adopt
an unexpected project/source/catalog revision. It does not render; the retained
native PNG receipt identifies the exact revision and job that produced it.

From this directory, use the following operator sequence against an already-live
Device Studio API:

1. `python3 curate.py verify` checks copied inputs, provenance and any archived
   references present in the workspace.
2. `python3 curate.py family-revise2` reopens current public family routes and
   appends/reopens the approved tablet and monitor setup revisions. It uploads
   only manifest-pinned sources through the public project-media owner and
   refuses ambiguous copies.
3. `python3 curate.py family-optix-revise3` appends or reopens the separately
   checkpointed renderer-backend revisions.
4. `python3 curate.py family-watch-synthetic-revise4` reopens the exact watch
   scene, preset, source-media and shot revisions, checks the accepted r4 recipe,
   then compiles. On an unchanged r4 checkpoint it performs no catalog or shot
   save. It stops if a public owner head differs from its checkpoint.

These commands never start Device Studio or submit renders. Native stills are
separate, explicitly coordinated render-owner operations. The retained watch
PNG is accepted only as a synthetic 12:48 static calibration, not real watch UI.
Tablet portrait and monitor outputs are parent-reviewed for their exact static
setup and app mapping; tablet fine-print readability and landscape qualification
are not claimed.

The tablet and monitor revision-3 OptiX outputs are retained under
`docs/internal/goal/evidence/E4/family-native-review/primary-showcase-optix-tablet-monitor-rev3/`.
Their public render job IDs, recipe hashes, verified PNG hashes, dimensions, and
bounded root visual reviews are recorded in `delivery-receipt.json`. Tablet
review is portrait-only and excludes fine-print readability and landscape
qualification; monitor review is limited to the matching screenshot and saved
static setup.
