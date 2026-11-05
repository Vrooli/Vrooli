# Smart display v7 reviewed starter

This is an explicit local starter install. Selecting **Reviewed generic smart display v7** bootstraps the fixed v7 candidate inputs, reconstructs the retained qualification package from the embedded source bytes, and passes it through the existing qualification-package importer. The installer creates no render job and performs no model or provider call. The exact original package remains in E5 evidence; this compact certificate plus its native PNG and the already embedded source files reconstruct the same inner `package_sha256` (`7139e503c471e724789f61733712ff3f016ed33e841605f197574b338c3e1bd2`). This inner digest covers canonical package contents and differs from the SHA-256 of the external JSON package file.

The retained review is scoped to `visual_setup`, dimension state `source_geometry_envelope`, display `main_display`, static motion, the v7 tabletop support, and the exact scene, motion, preset, candidate, Library revisions, and native output below. The source geometry envelope remains unknown as a measured device dimension. This does not approve other poses, motions, product identities, or dimensions.

- Candidate `generic-smart-display-candidate-v7@1`, SHA-256 `a3b928a5583afb0c38ac367868f3588a413881dad111ad82ea34af8fd551355c`.
- Scene `generic-smart-display-study-candidate-v7@1`, SHA-256 `faa23c177547e2b0dba3293364225f2123ea93f3a76d834b3980b19fe033abcd`.
- Motion `phone-still-identity@1`, SHA-256 `45e46e09f0f01bd04b71d24993e43ada9dee199d294429f259be8024aaf7d236`.
- Preset `generic-smart-display-tabletop-still-v7@1`, SHA-256 `903e309d1bbd1af7be33c993bb92e7d4a3cef4929c50f4e5f1572f104ebedee0`.
- Native still SHA-256 `44b75097389ae53ecd1bf87b47e93fee43b9fb74cd33c381966e767d3e1c73fb`; originating job `e83ef49d-62ee-46aa-b65c-cb6d470e2782`.

The package's hashes detect changed bytes; they are not a cryptographic reviewer signature. Source review records remain the Library's existing `edit,render` scope. This local embedded install does not change those source review permissions or enable separate source export/publication.

See [NOTICE.md](NOTICE.md) for the exact six source artifacts, attribution, license records, and included license text.
