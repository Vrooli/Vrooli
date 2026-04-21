# Process Research Item

## Purpose

Execute the actions defined in a research item's `conclusion.md`. Read the conclusion, carry out each action (create backlog items, update documents), and write a completion summary.

**Required reading:** `prompt-manager skill read swarm-manager-backlog-tools` — folder structure, artifact schemas, and CLI commands for reading/writing backlog files.

**Required reading:** `prompt-manager skill read swarm-manager-processing-guidance` — shared processing workflow, decision hierarchy, and completion evidence patterns.

**Required reading:** `prompt-manager skill read swarm-manager-initiative-context` — load the initiative's members and related initiatives before executing actions; the reuse-before-create heuristic applies at execution time too if the conclusion left the choice open.

## Scope

**In scope:**
- Reading the research item's `conclusion.md`
- Executing each action in the Actions section
- Writing `notes.md` with a completion summary

**Out of scope:**
- Modifying the `conclusion.md` itself (it is the authoritative research output)
- Implementing code changes (create backlog items for that)
- Re-running or extending the research (create a new research item if needed)
- Modifying `archive/` — user-provided materials must not be altered

## Instructions

You are processing a completed research backlog item. The research workshop has produced a `conclusion.md` with findings and actions. Your job is to execute those actions.

**Item context:**
- Kind: research
- Title: {{ITEM_TITLE}}
- Name: {{ITEM_NAME}}

### Processing Steps

1. **Read the conclusion**

   ```bash
   swarm-manager backlog get --kind research --name {{ITEM_NAME}}
   swarm-manager backlog files --kind research --name {{ITEM_NAME}}
   ```

   Then read `conclusion.md`:
   ```bash
   swarm-manager backlog file-get --kind research --name {{ITEM_NAME}} --path conclusion.md
   ```

   Also read `spec.json` to understand the item's initiative, tags, and other metadata that should be preserved in follow-up items. If the item belongs to an initiative, check for initiative-level context files (`swarm-manager initiatives files --name <initiative>`) that may inform your processing.

2. **Find the Actions section**

   Locate the `## Actions` section in `conclusion.md`. This contains the explicit instructions for what to do.

3. **Execute each action**

   Process actions in order. For each action, identify its type and execute accordingly. The action vocabulary is not additive-only: if the conclusion says delete, delete; if it says update, update.

   #### `Create backlog item`

   Use the swarm-manager CLI to create the item:

   ```bash
   swarm-manager backlog create \
     --kind {kind} \
     --name {name} \
     --title "{title}" \
     --description "{description}" \
     --priority {priority} \
     --effort {effort} \
     --initiative {initiative} \
     --depends-on {depends_on}
   ```

   For multiple items, use batch create:
   ```bash
   swarm-manager backlog batch-create --stdin <<'EOF'
   {"items": [
     {"kind": "...", "name": "...", "title": "...", "description": "...", "initiative": "..."},
     {"kind": "...", "name": "...", "title": "...", "description": "...", "initiative": "..."}
   ]}
   EOF
   ```

   Preserve initiative references from the research item. If the research item belongs to an initiative, follow-up items should reference the same initiative unless the conclusion explicitly states otherwise. When you set `initiative`, the server attaches the item to that initiative's `items[]` automatically — no follow-up `initiatives add-items` call is needed.

   #### `Update backlog item`

   Patch metadata on an existing item. Supply only the fields the conclusion changed:

   ```bash
   swarm-manager backlog update --kind {kind} --name {name} --data '{"priority": 2, "depends_on": ["fix/new-blocker"]}'
   ```

   Valid patchable fields: `title`, `description`, `priority`, `depends_on`, `initiative`, `tags`, `effort`. Changing `initiative` automatically syncs membership on both the old and new initiatives — do not emit follow-up `initiatives` calls.

   #### `Delete backlog item`

   Delete the item entirely:

   ```bash
   swarm-manager backlog delete --kind {kind} --name {name}
   ```

   The server cascades referential integrity automatically: the ref is removed from every other item's `depends_on` and from the enclosing initiative's `items[]` in one atomic operation. Do not run manual cleanup.

   #### `Update initiative`

   Patch initiative metadata:

   ```bash
   swarm-manager initiatives update --name {name} --data '{"priority": 3, "depends_on": ["other-initiative"]}'
   ```

   Valid patchable fields: `title`, `description`, `priority`, `depends_on`, `status` (`active` | `completed`).

   #### `Update document`

   Make the specified changes to the specified file:
   - Read the file first to understand current content
   - Apply the changes described in the action
   - Verify the result

   #### `No further action required`

   No actions to take. Write `notes.md` confirming the research is complete and the findings are the deliverable.

4. **Write completion summary**

   Write `notes.md` in the item folder with a summary of what was done:

   ```bash
   swarm-manager backlog file-upload --kind research --name {{ITEM_NAME}} --path notes.md --stdin <<'EOF'
   # Completion Summary

   ## Actions Taken
   - [List each action executed and its outcome]
   - [Include backlog item names/IDs created]
   - [Include files modified]

   ## Deviations
   - [Any differences from what the conclusion specified, and why]
   - [Or: "None — all actions executed as specified"]

   ## Verification
   - [x] All actions in conclusion.md executed
   - [x] Follow-up backlog items created (if applicable)
   - [x] Documents updated (if applicable)
   - [x] notes.md written

   ## Follow-up
   - [Any remaining considerations or monitoring needed]
   EOF
   ```

5. **Verify outputs**

   ```bash
   swarm-manager backlog files --kind research --name {{ITEM_NAME}}
   ```

   Confirm `notes.md` was created.

## Error Handling

If an action cannot be completed:

1. **Document what blocked you** — be specific about which action failed and why
2. **Continue with remaining actions** — don't stop at the first failure
3. **Record partial progress in notes.md** — list what succeeded and what failed
4. **Do not mark the item as complete** if critical actions failed

## Anti-Patterns

- **Don't** modify `conclusion.md` — it is the authoritative research output
- **Don't** implement code changes directly — create backlog items for implementation work
- **Don't** skip actions — execute every action in the Actions section
- **Don't** treat the action list as additive-only. If the conclusion says delete or update, execute that action; do not silently downgrade it to a creation.
- **Don't** create backlog items with vague descriptions — copy the detail from the conclusion
- **Don't** forget to preserve initiative references on follow-up items
- **Don't** write files directly to disk — always use the backlog CLI for item folder files
- **Don't** silently fail — always document what happened in notes.md
- **Don't** chase cascade bookkeeping. Deletes and initiative moves keep both sides in sync on the server. Agent-side "also update the initiative" or "also clean up dependents" calls are redundant.
