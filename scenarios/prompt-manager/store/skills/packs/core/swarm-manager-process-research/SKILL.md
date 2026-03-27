# Process Research Item

## Purpose

Execute the actions defined in a research item's `conclusion.md`. Read the conclusion, carry out each action (create backlog items, update documents), and write a completion summary.

**Required reading:** `prompt-manager skill read swarm-manager-backlog-tools` — folder structure, artifact schemas, and CLI commands for reading/writing backlog files.

**Required reading:** `prompt-manager skill read swarm-manager-processing-guidance` — shared processing workflow, decision hierarchy, and completion evidence patterns.

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

   Also read `spec.json` to understand the item's initiative, tags, and other metadata that should be preserved in follow-up items.

2. **Find the Actions section**

   Locate the `## Actions` section in `conclusion.md`. This contains the explicit instructions for what to do.

3. **Execute each action**

   Process actions in order. For each action, identify its type and execute accordingly:

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
   [
     {"kind": "...", "name": "...", "title": "...", "description": "...", ...},
     {"kind": "...", "name": "...", "title": "...", "description": "...", ...}
   ]
   EOF
   ```

   **Important:** Preserve initiative references from the research item. If the research item belongs to an initiative, follow-up items should reference the same initiative unless the conclusion explicitly states otherwise.

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
- **Don't** create backlog items with vague descriptions — copy the detail from the conclusion
- **Don't** forget to preserve initiative references on follow-up items
- **Don't** write files directly to disk — always use the backlog CLI for item folder files
- **Don't** silently fail — always document what happened in notes.md
