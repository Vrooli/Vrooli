---
name: "development-toolchain-validator"
description: "Validate the development toolchain by running skills against pristine golden scenarios: register goldens, store skill manifests, and read per-skill verdicts, coverage and fitness reports for a golden."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["development-toolchain-validator", "toolchain", "golden-scenario", "skills-validation", "skill-fitness", "coverage", "goldens"]
  requires:
    scenarios: ["development-toolchain-validator"]
    commands: ["development-toolchain-validator goldens list"]
  origin:
    kind: "authored"
---

## development-toolchain-validator

Goldens are generated from pinned templates; verdicts are recorded per (skill or tool, golden).

| You want | Command |
|---|---|
| Which goldens exist | `development-toolchain-validator goldens list`, `development-toolchain-validator goldens get` |
| Latest verdicts for a golden | `development-toolchain-validator report golden-summary` |
| Which skills are covered | `development-toolchain-validator report coverage` |
| Is a skill trustworthy across goldens | `development-toolchain-validator report skill-fitness` |
| History of one tuple | `development-toolchain-validator report tuple-history` |
| Refresh a golden | `development-toolchain-validator goldens regenerate` |

Run `development-toolchain-validator <group> <command> --help` for flags.
