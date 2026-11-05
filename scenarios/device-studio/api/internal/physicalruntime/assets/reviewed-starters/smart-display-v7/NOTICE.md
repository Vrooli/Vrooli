# Source and license notices

This reviewed starter packages six source artifacts. Their original IDs, immutable Library revisions, sizes, SHA-256 values, and source bundle review hashes are pinned in `qualification-certificate-v1.json`.

| Source artifact | Bytes | SHA-256 | License and attribution |
|---|---:|---|---|
| `vrooli-generic-smart-display-concept-v7` | 346,044 | `e540c0dea92fe9217d2ac3833f3e89f0b77346e4ad22744d2e758b74566d4dd5` | Vrooli-authored generic device concept, AGPL-3.0-only. Its source is v6 SHA-256 `0b77cc12cf175c5c2d8640e0b81b3b1d5ac216233db3fc1fff4ba2d8b4f428e4`; v7 removes only the 192 coplanar front-cap triangles. Generator and derivation manifest: `assets/sources/generic-smart-display-v7-20261021/` and `api/internal/physicalruntime/assets/smart-display-v7/`. |
| `laptop-study-room-no-cup-glb` | 420,960 | `68a8ad34f5af76f681e9aab7337ce80514221bda333c6713b58f624556660f56` | Vrooli-authored room/table scene derivative, AGPL-3.0-only. It omits the cup nodes from the Vrooli phone-study environment and does not include the separately licensed Jack Baeten laptop model. Source derivation is in `api/internal/physicalruntime/laptop_starter.go`. |
| ambientCG Travertine001 Color | 16,152,869 | `e14668f66d61b73c1b1bd1df6150449244ea9c766af12908e3f522c6c93b09c8` | ambientCG, CC0-1.0. |
| ambientCG Travertine001 Displacement/Height | 355,941 | `64001fed9f67bbf9af0fccdf079fccbb0e01cba369c9c0319009b3c9aa2cabaf` | ambientCG, CC0-1.0; used only as subtle bump, not displacement. |
| ambientCG Travertine001 NormalGL | 5,152,752 | `68419c7540b71608d34ef103d6ae839f7144128784b3b03af44c95b1db8b168b` | ambientCG, CC0-1.0. |
| ambientCG Travertine001 Roughness | 5,302,649 | `a8f053413e46dacce39e601d1ceb5ae615cd7f91e1f270db386e6c6470cf9b90` | ambientCG, CC0-1.0. |

The ambientCG source is <https://ambientcg.com/view?id=Travertine001>. Its captured license page, acquisition date, source archive SHA-256 `10ca543f7b24a51f51c1b9844189c1462bb75d1c5e0431392a0aefed56da8e0b`, and file hashes are retained at `docs/internal/goal/evidence/E3/scene-studies/materials/Travertine001-acquisition.json` and `license-page.html`. The maps are stored in `api/internal/physicalruntime/assets/phone-table/textures/` under ambientCG's CC0 1.0 dedication; the Vrooli-authored room geometry is not CC0.

The full GNU Affero General Public License v3 text is included as `AGPL-3.0.txt` and at the repository root `LICENSE`. The CC0 source and license links above preserve attribution and provenance; the CC0 dedication text is available at <https://creativecommons.org/publicdomain/zero/1.0/legalcode>.
