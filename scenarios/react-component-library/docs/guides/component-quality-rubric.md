# Component quality rubric

Preview review uses this rubric for component implementation and adoption
decisions. A passing automated contract is not a visual acceptance.

| Dimension | Accept when | Reject or defer when |
|---|---|---|
| Geometry | Spacing and sizing use governed tokens; long content stays readable | Text clips, overlaps, or requires story-only CSS |
| Hierarchy | The subject has a clear title, content, action, and state order | Decorative harness content competes with the subject |
| Density | Default density is useful at desktop and narrow widths | Rows collapse to one-character columns or become an unreadable wall |
| Semantics | Native roles, names, labels, and headings match the public purpose | A visual control has no keyboard or accessible name path |
| States | Loading, empty, failure, disabled, focus, and recovery are explicit where credible | Only the happy path is represented for a stateful subject |
| Motion | Reduced motion removes non-essential animation and transitions | Capture depends on a timing race or animated layout |
| Contrast | Light/dark and forced-colors states preserve text, focus, and status distinction | Status is communicated by color alone or loses focus visibility |
| Overflow | No unexpected horizontal overflow at supported widths | A story hides overflow or adds per-story CSS to mask an owner defect |

Each reviewed component receives one disposition: `adopt`, `adapt`,
`structural-only`, `defer`, or `reject`. The disposition names the exact
version, evidence paths, reviewer note, and revisit condition when it is not
`adopt`.
