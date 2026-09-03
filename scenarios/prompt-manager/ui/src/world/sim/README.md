# sim

The renderer-free world model: places, actors, signals, the actor state
machine, grid pathing, interpolation, the idle layer and the animation phase.
Deterministic from a seed at a fixed tick; tested in Node with no WebGL.

Import rule: `config` only. Never `three`, `react`, `@react-three/*`, or any
other world layer. ESLint enforces both (see `__lint__`).

Home is the desk: members rest at their desk seat and take outings to the
commons; `invariants.ts` names the rules every settled state must satisfy and
the tests and the smoke tool both run it.
