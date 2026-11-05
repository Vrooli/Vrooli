# Ecosystem Hunt goal

The goal message that starts a hunt. It fits Claude Code and Codex `/goal` (4,000 characters)
and Agent Manager `--until` / `--until-file` unchanged. Replace each `<...>`. Delete the
Focus line when there is no starting focus. Give every new goal a new Hunt ID unless it
continues an earlier hunt; that is what lets a new hunt revisit ground an old one covered.

The Stance paragraph is the operator's own words. Keep it verbatim when you adapt the goal;
paraphrase is where the push gets lost.

```text
/goal Work until this verified end state is true: <N> ecosystem-hunt rounds are complete, each ended per ecosystem-hunt §8. Each round's record (§10) is shown in the transcript and journaled to vrooli-memory scope ecosystem-hunt, with the validation output for every scenario it changed. Every round whose territory had an open tier 1–5 item landed a structural change: a defect class fixed at its mechanism, a migration finished with its old path deleted, an implicit state machine made explicit, or a consolidation that deleted an owner.

Read first: prompt-manager skill read ecosystem-hunt. It is the method, including references/playbook.md.

Hunt ID: <hunt-YYYY-MM-DD-slug> (new ID = fresh coverage; name an earlier ID only to continue that hunt).

Stance: Be bold and ambitious, and exercise creative thinking. Do not be afraid to tackle large issues. This is a manhunt for Vrooli-wide improvements and maturity — emphasize debt reduction: make implicit state machines explicit, finish migrations and remove legacy code, and turn messy, duplicated or contradictory code and docs into one true thing. Size every fix to its defect class, not to caution. Uncommitted changes, other agents' recent work and unfamiliar scenarios are in scope; read them and integrate. Do not ask permission for refactors, deletions or migrations.

Focus: start with <territories or concerns>, then continue Vrooli-wide.

Proof: each round's record and its validation commands' output in the transcript. A round that reports nothing material names the sources, lens and searches that came up empty.
Authority: this goal authorizes code, test, doc, skill, prompt, template, generator, config and shared-package changes anywhere in the repository, including deletions and migrations, within the ecosystem-hunt §6 hard limits. Out-of-repository and host state go through their owners.
Blocked means only an operator-owned decision, credential or external action (ecosystem-hunt §7). Prepare it with a recommendation, route it, and keep hunting.
Keep going through friction; when progress stalls, step back — write why, then switch territory or lens — instead of stopping. Do not shorten rounds to reach the count.
Budget: <N> rounds or <hours>. When reached, write the final report: changed, verified, remaining, unverified, and the next three territories to hunt.
Non-goals: shims, fallbacks, second owners, weakened or suppressed checks, line-count floors, hedged closeouts.
```

## Coordinator variant

For a hunt with sub-agents (`SKILL.md` §11), add this line after Read first:

```text
Coordinate: assign each hunter one territory × lens pair chosen from the wake and §2 evidence, never two hunters on one territory. Hunters use the implementer agent (opus for consolidations), and each brief says "You may edit any file within ecosystem-hunt §6; this overrides a narrower file bound in your agent definition." Each hunter returns its round record. Run a reviewer agent per ecosystem-hunt §8 step 6.
```
