---
name: "local-experimentation"
description: "Compare a candidate tool, model or technique against the current baseline on this host: freeze the evaluation, admit resource and acquisition costs before trials, run and clean up through the owners, and decide adopt, investigate or reject from retained evidence."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["experiment", "benchmark", "capacity", "reproducibility", "local"]
  icon: "flask-conical"
  status: "active"
  revision: 16
  createdAt: "2026-10-20T00:00:00Z"
  updatedAt: "2026-11-15T08:00:00Z"
  requires:
    commands: ["vrooli capacity", "scenario-dependency-analyzer deps"]
  origin:
    kind: "authored"
---
## Local experimentation

Use a controlled comparison when a choice is expensive to reverse: a default model, a runtime, a
storage format, a production technique. For a choice you can make by reading two outputs, decide
by inspection and build. The goal is a decision you can reproduce, with failures and untested
routes kept, not a pile of metrics.

Owners keep their mechanics: capacity admission (`docs/configuration/host/capacity.md`),
acquisition (`docs/package-governance.md`, Scenario Dependency Analyzer), lifecycle and media
owners. This skill adds judgment; it creates no manager. Use `adaptive-research` when the next
question must come from evidence, and `scientific-debugging` for a known defect.

### 1. Fix the question before any output

- Recall prior experiments (`search-hub query "<intent>" --type library,record,skill,doc`, or
  `rg` over the topic's runs).
- State a falsifiable hypothesis and the baseline. Freeze inputs, metrics, acceptance criteria,
  candidate and retry limits, and the stop time before you look at results. Check the evaluator
  against the real outcome the operator cares about; a technical proxy keeps its narrow meaning.
- Budget each route: trials and repetitions, wall time, tokens, peak GPU, RAM and disk including
  downloads, unpacking, caches and outputs. Stop on failed admission, new terms or authority,
  OOM, interference, the repeated-failure limit or an exhausted budget.

If you cannot fix these, the decision is `investigate`; do not start a heavy trial.

### 2. Admit the exact route

- Pin inputs and dependency versions. Acquire tools, models and data through Scenario Dependency
  Analyzer or the resource's owner; routine in-scope acquisition needs no extra confirmation.
  Record source, version, terms, expanded bytes and where it installs.
- Read terms against the research use; a research-only license may still forbid the product.
- Admit heavy work through the capacity owner with fresh readings. A warning, a recorded claim,
  an enforced refusal and your own safety hold mean different things. Unknown demand or a missing
  required admission is `not_tested`, never a rejection.
- Never acquire weights for a route that already fails memory, disk, terms or authority.

### 3. Run trials

- Submit, watch, cancel and fetch through the owner; record each operation's identity before
  waiting. Serialize when overlap would break capacity or comparability.
- Count refusals before execution separately from executed trials. Retry a refused slot only when
  evidence shows the failed condition changed. Amend the protocol only prospectively, keeping the
  earlier results.
- Measure the declared outcome and costs; separate setup, trial and selection costs; label
  sampled peaks.
- Check outputs against acceptance and then judge their usefulness. A decodable file is not
  acceptance. Recheck an apparent winner on fresh runs before claiming it.

### 4. Decide and clean up

| Evidence | Decision |
|---|---|
| A required check (effect, acceptance, permission, admission, cleanup) is unverified, or the route is `not_tested` | `investigate`: name the gap, owner and next check |
| Comparable trials fail a preset criterion | `reject`: keep the failure and the baseline |
| Comparable trials pass, recover cleanly and beat the baseline | `adopt`: name the smallest maintained integration, and build it when the commission includes it |

Keep accepted outputs, inputs, hashes, receipts and the decision in the run folder (never `/tmp`,
which is cleared after three days). Remove only task-owned assets through their owners; never
clear global caches, shared models or unrelated processes. Report what remains. For experiments on
copied production code, source access with credentials, or unattended runs, read
[trials](references/trials.md).

### Traps

| Trap | Instead |
|---|---|
| Admitting against nominal GPU size or yesterday's free memory | Fresh readings plus owner admission |
| Installing first and measuring disk afterwards | Budget peak storage before acquisition |
| Changing prompts, seeds or acceptance after one result | Freeze them; a change is a new experiment |
| Retrying an uncertain job or killing every matching process | Reconcile the exact job and cancel only its scope |
| Calling an untested route rejected | Keep `not_tested` and the reason |
| A frozen evaluator for a choice that is cheap to reverse | Decide by inspection and build |
