# Case: rebaseline

Taken from Hub asks 5ae2012a (15 Nov 2026) and 85eaea3e (11 Nov 2026). You are the
install agent for the disk-capacity work (`--from "Install agent"`).

## Facts you know

- The disk-capacity update for Vrooli's control plane is built and its tests pass.
- `make install` refused to install it: the update changes how stored data is
  written (the "persistent-state contract") in five areas: disk capacity,
  resources, process records, managed services and operator state.
- Only the operator may approve that switch. Agents must not run the command.
- The command the operator runs, in a terminal on the Vrooli computer:
  `cd ~/Vrooli && make rebaseline`
- What it does, in order: runs the full build check first (takes several minutes
  before it asks anything); lists the stored-data changes and waits for the
  operator to type a confirmation; saves the current installed version to
  `~/.vrooli/state/root-rebaseline/<id>/`; pauses automatic updates; installs the
  new version and checks it; then turns automatic updates back on.
- Undo: there is no automatic rollback, because the old version cannot read the
  new stored data. The saved copy makes a manual rollback possible.
- This is the second time in five days the operator has been asked to run this
  command (11 Nov for the same kind of change). The installer stops every time
  any stored field is added, even when nothing is removed or renamed. A possible
  root-cause fix: propose that the installer stops by itself only when stored data
  is removed or renamed, and lets added fields through. That changes a safety
  rule, so it would come back to the operator as its own decision.
- If the operator holds: the installed version stays, the disk-capacity feature is
  not active, and work continues on source only. Nothing breaks.
- No answer: same as hold.
- After the operator says they ran it, the agent finishes the install and checks it.

## What the sender would recommend

Run it today. The update is built and tested, and only the operator can make this switch.
