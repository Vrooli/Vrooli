# scene

React Three Fiber components that draw the world: the ground (a fogged
terrain disc to the horizon with the lot as a low slab on top), places (one
instanced draw per slab kind: floors, walls, paths), props, trees, actors
(four instanced draws: contact discs, bodies, faces, extras), labels and
editor handles. Draw calls scale with kinds of thing, never with counts. They read simulation state
through refs inside `useFrame` and mutate Three objects directly; they never
call `setState` per frame and never own behaviour.

Import rule: `engine`, `sim`, `config`. Never `hud` or `data`.
