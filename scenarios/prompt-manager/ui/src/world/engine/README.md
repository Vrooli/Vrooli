# engine

Rendering infrastructure with no knowledge of the world's meaning: the Canvas,
the lighting rig (key light, HDRI + Lightformers, sky, fog), the post chain
(N8AO, bloom, AgX), the Explore and walking camera rig,
the quality governor, the diagnostics probe/overlay and asset URL helpers.

Home, stored poses, and focus use the same footprint framing solver.
Focus preserves the current viewing angles and accounts for elevated boxes.
Follow commands are gated by target displacement and translate immediately,
preserving an in-progress user orbit or dolly. Stop detaches Follow. Automatic
pose commands reset magnification and focal offset; manual input cancels an
in-flight pose route before applying its first delta.

`frameDistance` is the single distance solver; focus supplies its target bounds
through `poseForBox`. The former library box-fitting path was removed.
The Camera toolbar exposes Home, Frame selection, zoom, directional pan/orbit,
top/front/isometric views, and an Orbit/Pan drag tool. Its blue ring shows the
current orbit pivot. Select the input device explicitly; event frequency is not
a reliable device detector. Mouse-button and touchscreen maps use `camera.input`;
the operator profile owns wheel/trackpad input in `camera/navigation.ts`:

| Input | Action |
| --- | --- |
| Left mouse drag / one-finger touch | Selected Orbit or Pan tool |
| Middle or right mouse drag | Pan |
| Mouse wheel | Dolly (move the camera), with a fixed lens |
| Trackpad two-finger scroll | Pan in both screen axes |
| Trackpad pinch / Ctrl-wheel | Dolly through the same surface guard as wheel input |
| Two-finger touchscreen gesture | Dolly and pan |
| Three-finger touchscreen gesture | Pan |
| Arrows, with world keyboard focus | Orbit |
| WASD, with world keyboard focus | Pan left/right/up/down |
| + / - | Dolly in/out |

Manual keyboard and wheel motion is immediate. Pointer drag is immediate by
default; optional 60 ms smoothing ends on pointer release/cancellation or focus
loss. Reduced motion disables that smoothing. `navigationDelta` preserves
ordinary elapsed time through 250 ms frames and treats longer gaps as a 50 ms
resume step. The controller integrates ordinary long frames in steps of at most
50 ms, applying collision constraints at each step. Automatic route duration
therefore does not stretch below 20 FPS.

Device choice, sensitivity, invert options and drag smoothing persist in this
browser under `prompt-manager.world.navigation.v1`; invalid stored values fall
back to bounded defaults. The existing account zoom-target preference still
owns pointer/center targeting. Navigation telemetry updates the toolbar without
re-rendering the world composition.

## Walking

First person and Third person share one grounded visitor body, separate from
the autonomous agents. WASD/arrows walk; Shift runs; Space or the Jump button
jumps; drag looks around. Jumping uses gravity, swept ceiling clearance and ground
landing. Holding Space does not repeat jumps, and airborne double jumps are disabled. Explicit
mouse capture enables continuous look. Escape releases capture, and a subsequent
Escape returns to Explore. The Return button always restores the saved Explore
position, target, and polar limits. Home restores the complete home view instead.
Text inputs, dialogs, focus loss, and mode switches release held movement.

`Walker` takes injected ground and collision queries. The composition layer
supplies terrain height and body-footprint eligibility from shoreline, slope,
and world-boundary checks. Movement uses bounded spatial steps and wall sliding;
it cannot skip a narrow blocked region during a slow frame. A vertical capsule is
swept conservatively against structural and rendered furniture bounds, including
transformed instances. Rounded corners can stop slightly early. The third-person
boom retracts against those obstacles and terrain. This is grounded walking,
with jumping, but without flight. Tunable operator defaults are in `config/navigation.ts`.

With mouse capture active, the centre marker is the picking target; frozen OS cursor coordinates do not control agent selection.
Walking mode selection focuses the canvas immediately. A click selects an agent;
a look-drag does not. In walking modes, selection invites the agent to a navigable
spot about two metres in front of the visitor, facing the visitor on arrival.
The simulation routes this presentation movement separately from live run status.
An invitation also advances an otherwise frozen demo roster until dismissal.
End conversation, selecting another agent, or returning to Explore releases the
invitation. An unreachable agent stays in place rather than crossing blocked cells.

Visitors step over objects up to 45 cm high using an up/across/down body sweep,
including headroom checks. A low lintel limits the lift to available clearance;
a small floor lip does not demand the maximum lift. Spawn resolves low supporting
slabs before looking for a different location. Terrain and shoreline checks remain enforced; rendered
colliders own furniture clearance instead of the coarser agent navigation grid.

Geometry may declare `cameraObstacle: 'box'` or `cameraObstacle: 'triangles'`.
Triangle surfaces use cached convex half-spaces expanded by the camera sphere or
vertical body capsule, with each instance transform applied. This preserves holes
and the empty space under sloped roofs. Geometry is immutable while cached. The
bounds/outline diagnostics remain conservative boxes around the tested primitives.

A committed world change checks the visitor position against the new geometry.
If no nearby safe site remains, walking exits with a visible explanation. Failed
pointer capture leaves drag-to-look and on-screen movement buttons available.

Run focused camera/HUD tests through Vitest, then scoped Test Genie phases.
`scripts/world-smoke/navigation.mjs` exercises the built UI with synthetic agents
and retains a JSON verdict plus screenshots. `scripts/world-smoke/camera.mjs`
retains focus/follow, automatic route, collision, and editor-cancellation coverage.
Browser wheel replay validates event semantics; physical OS trackpad momentum
and hardware-specific gesture feel still require a physical-device trial.

Capture mode exposes a fixed-time canvas snapshot after live performance
measurement. It runs the normal local frame callbacks and post chain, freezes
animation time, and excludes DOM chrome from golden pixels. The smoke tool also
retains a full-page screenshot and separately gates runtime and loading errors.

Environment owns the outdoor sky background. Period effects set exposure and
indoor background colors; they must not overwrite the outdoor cube texture.
The diagnostic attribution pass counts the background once, excludes it from
isolated child renders, and restores it even when measurement fails.

Import rule: `config` only. Never `sim`, `scene`, `hud` or `data`.

Adjustable values come from `config`, including material, post-processing,
lighting, camera, and diagnostics settings. The literal gate permits only
documented structural constants such as unit conversions and hash mathematics.
