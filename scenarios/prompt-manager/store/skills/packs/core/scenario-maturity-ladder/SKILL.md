## Practice focus: Scenario Maturity Ladder

A scenario climbs a fixed ladder toward production-readiness. **Locate the current rung, then make the change that rung calls for** — which, at the higher rungs, means *large architectural and feature work*, not timid hygiene. This is the cross-dimension, scenario-level sibling of the per-dimension ladder inside `screaming-architecture-audit`.

The most common failure mode is timidity: an agent does competent low-risk cleanup (a lint fix, a tidier function) when the scenario actually needs a missing feature built or a cyclic architecture untangled. The ladder tells you when the bigger move *is* the right move.

Required reading:
- `prompt-manager skill read improvement-do-and-dont`

---

### **1. The rungs**

The ecosystem-manager controller works the **lowest unsatisfied** rung each loop. Low rungs are hard gates (you must clear them first); high rungs are soft (amplified, but a re-opened lower gate still pulls you back). Gates are "satisfied for now" and re-checked every loop, so the ladder oscillates naturally.

| Rung | Satisfied when | Typical change size |
|---|---|---|
| **R0 Runnable & green** | build passes, no failing tests, no blocker/error findings | small fixes — make it run |
| **R1 Safe & standards-clean** | no high/critical security, standards clean, deps sane | small–medium hardening |
| **R2 Evolvable architecture** | no import cycles, structure clean, contracts + docs map present | **large — structural/architectural refactors** |
| **R3 Features hardened** | coverage/tidiness/ui/visual/performance clean | **medium–large — build & harden features** |
| **R4 Capability progression** | operational targets ≥ the profile's threshold | **large — new capability / missing features** |

R0/R1 are about *not being broken*. R2–R4 are about *getting genuinely better* — and that is where the expected change size grows. **If the scenario is at R2 or R4, a large architectural change or a real new feature is the correct, expected move.** Don't shrink it into a cosmetic edit.

---

### **2. Locating the current rung**

1. Read the latest test/audit results (`vrooli scenario test <name>`, or the test-genie report) and the scenario's `problems`/`progress` docs (`knowledge-observatory-tools`).
2. Walk the rungs top-down (R0→R4) and stop at the **first one that isn't satisfied** — that's where the work is:
   - Build failing or tests red, or any blocker/error finding? → **R0**. Make it run/green first; nothing else matters yet.
   - Green but high/critical security or unclean standards/deps? → **R1**. Harden before building more.
   - Safe but import cycles / mislocated packages / missing docs map? → **R2**. This is an *architecture* loop — expect to move modules, break cycles, restructure. Small nibbles won't clear it.
   - Architecture sound but thin coverage, messy types, rough UI/perf? → **R3**. Build the feature out properly and harden it.
   - Everything hardened but operational targets below target? → **R4**. The scenario is missing capability — build the missing feature end-to-end (requirement → implementation → tests → validation).
3. Do the rung's work at the rung's scale. Then re-measure — closing a rung often re-opens a higher one (new feature work at R4 re-opens an R3 coverage gap), and that's the loop working as intended.

---

### **3. Right-sizing the change**

- **At R0/R1, prefer the smallest change that clears the gate.** These rungs reward precision, not ambition. Don't start an architecture rewrite while a test is red.
- **At R2/R3/R4, prefer the change that actually moves the scenario forward, even when it's big.** A missing operational target is not closed by a lint pass. An import cycle is not broken by renaming a variable. If the honest fix is "extract this package / build this endpoint / add this whole flow," do that.
- **Never substitute a lower-rung cosmetic change for the higher-rung work the scenario needs** just because it's safer. That's how a scenario stalls at "tidy but incomplete." If you find yourself polishing while a feature is missing, you're on the wrong rung.
- Every change still obeys `improvement-do-and-dont`: real progress, never gaming the measurement.

---

### **4. No known operational edge cases**

This is a judgment/reading skill — it changes no files itself. Pair it with the steer skill the controller selected for the current loop; this skill tells you *how big* the move should be, the steer skill tells you *what kind*.
