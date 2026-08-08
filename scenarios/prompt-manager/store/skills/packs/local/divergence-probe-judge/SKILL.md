## Meta focus: Divergence Probe — Judge

Three runners independently executed the skill `{{.skill_id}}` against the same
target scenario. Each produced a structured decision list. Your job is to diff
the three lists and report where they **materially diverged** — and, for each
divergence, anchor it to the exact sentence in the skill that permitted both
readings. Material divergence on a decision the skill exists to pin down is a
proven instance of forced guessing, which is the defect this probe exists to
find.

### Input

The three decision lists are provided below, one per runner, each labeled with
its runner node id:

{{.decision_lists}}

### Procedure

1. Read all three decision lists. Note each runner's `outcome`. If runners split
   between `executed` and `abstained`, that is itself a material divergence about
   whether the skill even applies.
2. Compare the lists dimension by dimension: `files_touched`, `commands_run`,
   `acceptance_checks`, and `key_decisions`.
3. A difference is **material** when it changes what the skill produced, not when
   it is cosmetic. Different target file for the same content, a different
   command that yields a different result, a different definition of "done", or
   opposite interpretations of the same term are material. Whitespace, ordering
   that does not affect outcome, and equivalent phrasings are not.
4. For each material divergence, find the sentence in the skill that permitted
   both readings. Use the `skill_sentence` anchors the runners recorded; if they
   point at different sentences, read the skill to locate the one sentence that
   actually left the decision open. Quote it verbatim in `skill_sentence`.
5. If the runners agree on every material dimension, the verdict is `converged`
   and `findings` is empty. Do not invent findings to fill the report.

### Severity

- `critical`: runners produced incompatible results the skill was explicitly
  meant to make identical.
- `major`: runners diverged on a decision the skill should have determined
  (the default for a real divergence — forced guessing).
- `gap`: the skill never addressed the decision at all; it needs a new rule.
- `minor`: divergence that barely affects the outcome.

### Output

Emit a single structured report:

- `verdict`: `converged` or `diverged`.
- `summary`: two to four sentences on how aligned the runners were and where the
  sharpest divergence sits.
- `findings`: one entry per material divergence, each with `severity`,
  `dimension` (which work-list field diverged), `skill_sentence` (the exact
  quoted sentence that permitted both readings), `divergent_readings` (two or
  more short descriptions of what each runner did), and `explanation` (why this
  is material and what the skill would need to say to converge).

Anchor every finding to a sentence. A divergence you cannot tie to a specific
sentence is either not material or not yet understood — resolve which before
reporting it.
