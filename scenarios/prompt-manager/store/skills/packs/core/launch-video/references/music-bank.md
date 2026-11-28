# Exact-hash launch-music bank

The Music Library owns approvals, tags, purposes and project-use lifecycle. The
launch-video manifest is a portable, versioned editorial overlay: it binds exact
source bytes to generation receipts, technical measurements, a private-use scope
and a candidate classification. It must not replace or silently update Music
Library state.

## Eligibility and query

Export the current rows through the supported Music Library CLI. Run
`music_bank.py audit` before querying or selecting. It checks source SHA-256,
generation-receipt SHA-256/output hash, Music Library paths and approval, and
reports the receipt's generation operation/request ids, the Library take id,
ffprobe format/duration, existing loudness facts, prompt tempo intent, rights
terms and every recorded use. A mismatch blocks selection. Prompt BPM is never a
measured tempo claim. Local measurements and prompt wording do not establish
operator taste or licensing rights.

`music_bank.py query --tag launch-video --tag hype --approval liked --unused`
uses both Music Library tags and the manifest's exact-hash editorial tag.
`--unused` matches Music Library's contract: accepted use makes a hash used, while
a draft association remains unused. `--available` adds a stricter filter matching
the default no-use reservation candidate set and excludes draft associations too.
The catalog export does not carry active reservation leases; use the owner CLI to
recheck availability immediately before reserve. Refresh the export after reserve,
draft, accept, rollback or tag changes; an old JSON export is a historical
snapshot, not a live lease or source of truth.

`music_bank.py tag --track-sha256 <sha256> --classification hype|non-hype|unclassified
--basis <evidence> --out <new-manifest.json>` writes a new manifest revision and
rebases its relative source and receipt paths. It never overwrites the current
manifest and never writes to Music Library. `hype` here means an editorial
candidate tag for this brief; it is not a new operator rating. Preserve existing
operator feedback as recorded, and do not promote a candidate to operator-liked
without the Library owner workflow.

## Render lint and use lifecycle

Every music-mode production must pass the bank lint through `workflow.py freeze`
before render. Freeze requires `audio.source_sha256` and a WAV in `render_inputs` to match the linted
exact hash, then includes the bank, current catalog export, selection and
lint receipt in the immutable snapshot. `workflow.py verify --snapshot ...`
checks those hashes after render. A rejected selection produces no snapshot.
This preflight binds the intended source; it does not prove which audio a renderer
actually used. The managed render receipt must still bind to the complete frozen
snapshot, and the delivered export needs its own audio inspection.

Use Music Library's supported lifecycle: reserve an unused liked hash for a
project/revision/consumer, bind a draft when the edit is real, release an
abandoned reservation, roll back a draft only through the owner command, and mark
accepted only after operator acceptance. Never infer acceptance from a private
render or a passing lint. Exact-hash use attaches to that track; a re-cut must
carry its own revision and preserve earlier use receipts.

`music_bank_lifecycle.py reserve-plan` verifies the frozen lint and candidate
against a current Music Library export (including the exact row's
`purposes: [launch-video]` and live `bank_state`), then prints a hash-pinned
`music-library catalog reserve` command. It refuses stale exports that omit
those owner-returned fields or show an active reservation. Default `--mode new`
requires a liked track with no uses and `bank_state: available`;
`--mode re-edit` permits the same liked exact hash with prior use records, but
rejects a duplicate project/revision and keeps all earlier uses intact. Use
re-edit only when the new cut deliberately reuses the same source SHA; a trim,
loop or remaster has a different derived hash and needs its own provenance and
review. Run the command only when the Music Library CLI's owner admission is
current and a fresh owner-CLI catalog confirms availability. Save the command's
JSON response unchanged as the reservation receipt. After render,
`music_bank_lifecycle.py prepare` verifies the managed bundle against the whole
snapshot, exact-hash reservation response and the saved reserve plan's
project/revision/consumer binding, video-bound decoder/playback receipt
(`decoder_status` and `playback_status` are `pass`), and clean footprint receipt
(`status: clean`, no `new_owner_items`); then it emits the exact
`commit --status draft` command and cleanup `release`/`rollback` commands.
Execute those through the owner CLI only. Refresh catalog before/after exports
and use `verify-state` to prove that one draft was added for the exact selected
hash, its `bank_state` became `draft`, prior use history was retained, and other
catalog rows did not change. Keep the render bundle,
reservation and mutation receipts, before/after exports, decoder/playback
evidence, and resource-footprint cleanup receipt together under the film
revision. Generate the cleanup receipt with `music_bank_lifecycle.py
cleanup-receipt` from the governed `footprint.py diff --baseline B --keep-file K
--disk --json` output; any new item or unchecked section makes it unclean. The
helper never calls Music Library or changes catalog state itself.

For an abandoned render before draft commit, export catalog immediately before
and after owner-CLI `release` and use `verify-release`; it requires uses to be
identical and only the reservation-derived bank state to return to `available`,
`draft` or `used`. For an abandoned committed cut, run owner-CLI `rollback`, then
`verify-rollback`; it requires exactly the target draft revision to disappear,
preserves all older draft/accepted uses and other metadata, and checks the
derived bank state. A refill request created by commit is separately governed
by Music Tools cancellation as described below.

Music Library draft commit also inserts an idempotent candidate-refill outbox
event keyed by the reservation. Its worker forwards the event to Music Tools;
the Music Tools contract records it in an unreviewed candidate queue held until
explicit dispatch after research and capacity gates. A commit therefore does
not itself start generation. Draft rollback cancels an undelivered request or
asks the queue to cancel a delivered one. Refill is for replenishing candidate
inventory, not automatic reuse of the old source or an inherited taste/rights
judgment. New generated output needs its own exact hash, take/receipt, and fresh
review. Do not dispatch or generate until the owner path is admitted.

## Re-edit and regeneration policy

- Re-editing an existing source preserves the original WAV, source hash,
  generation receipt, operator feedback and prior use records. Keep the new
  project/revision association separate. A render does not change the source's
  identity or inherit a rating for its new picture edit.
- A trim, loop, stem or remaster is a derived asset with a new output hash and a
  parent-source hash plus a processing receipt. Keep the original source. It is
  not a new generation and does not inherit a fresh operator judgment.
- Regeneration creates a new take record with a new exact hash and full request,
  model/version/terms, job, seed and output receipt. Never replace a failed or
  disliked track in place; preserve it and its feedback. A new take starts
  unreviewed, and earlier tags or approvals do not transfer automatically.
- Prefer the existing exact-hash alternatives for a re-cut before requesting
  more generation. Generate only through the admitted Music Tools owner path;
  check RAM, GPU, disk floor and teardown before each job. If any required
  admission or bounded install peak is denied, preserve the candidates and defer
  generation. Never evict another owner, use paid services or add credentials.
- The current Aquila A–F receipts record model terms that are not qualified for
  public or commercial distribution. This bank and its linter allow private
  review only until an authorized rights decision is recorded separately from
  taste.

## Current Aquila candidate set

The retained 2026-10-29 six-track batch has exact-hash operator feedback: A–D
were liked, with A preferred and C second; the follow-up E/F were also liked with
no ranking. A has a draft Aquila association; B–F have no recorded uses in the
snapshot. These are existing judgments about the tracks, not a film-bound
judgment that a new composition reaches the Aquila-A level. A new film still
needs a private exact-hash taste decision. See the topic LV8 run for current
catalog snapshot, query/lint receipts and its capacity decision.
