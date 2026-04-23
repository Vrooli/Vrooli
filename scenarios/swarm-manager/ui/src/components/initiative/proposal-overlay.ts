/**
 * proposal-overlay
 *
 * Translates a `Proposal` (or a filtered subset of mutation IDs) into a
 * `InitiativeGraphOverlay` the existing `InitiativeDependencyGraph` can
 * render as a diff against the current items.
 *
 * Kept as a pure function (no React, no service access) so it's easy to
 * unit-test and reuse from elsewhere — a future "revision diff" or CLI
 * preview can consume the same shape.
 */

import type { BacklogStatus } from "../../types";
import type { InitiativeGraphOverlay } from "./InitiativeDependencyGraph";
import type { Proposal, ProposalMutation } from "../../types";

export interface BuildOverlayOptions {
  /** Filter mutations to just these IDs. When undefined, all mutations apply. */
  acceptedIds?: string[];
}

/**
 * Build an overlay from a mutation-list proposal.
 *
 * Rules:
 *   - Unknown ops are ignored (the overlay is best-effort visualization; the
 *     server is the authority for what ultimately gets applied).
 *   - `full_graph` form isn't handled here — the server normalizes to a
 *     mutation list before the UI ever sees it for this surface, and the
 *     proposal review refuses to apply raw full_graph proposals anyway.
 */
export function buildOverlay(proposal: Proposal, options: BuildOverlayOptions = {}): InitiativeGraphOverlay {
  if (proposal.form !== "mutation_list" || !proposal.mutations) {
    return {};
  }
  const accepted = options.acceptedIds ? new Set(options.acceptedIds) : null;
  const keep = (m: ProposalMutation) => (accepted ? accepted.has(m.id) : true);

  const overlay: Required<InitiativeGraphOverlay> = {
    addedNodeIds: [],
    archivedNodeIds: [],
    movedOutNodeIds: [],
    statusChanges: {},
    addedEdges: [],
    removedEdges: [],
    addedNodes: [],
  };

  for (const m of proposal.mutations) {
    if (!keep(m)) continue;
    switch (m.op) {
      case "add_item": {
        if (m.item) {
          const id = `${m.item.kind}/${m.item.name}`;
          overlay.addedNodeIds.push(id);
          overlay.addedNodes.push({
            id,
            kind: m.item.kind,
            name: m.item.name,
            title: m.item.title || m.item.name,
            status: "backlog",
          });
          for (const dep of m.item.depends_on ?? []) {
            overlay.addedEdges.push({ from: dep, to: id });
          }
        }
        break;
      }
      case "archive_item": {
        if (m.target) overlay.archivedNodeIds.push(m.target);
        break;
      }
      case "move_initiative": {
        if (m.target) overlay.movedOutNodeIds.push(m.target);
        break;
      }
      case "change_status": {
        if (m.target && m.status) overlay.statusChanges[m.target] = m.status as BacklogStatus;
        break;
      }
      case "add_edge": {
        if (m.from && m.to) overlay.addedEdges.push({ from: m.from, to: m.to });
        else if (m.target && m.to) overlay.addedEdges.push({ from: m.target, to: m.to });
        break;
      }
      case "remove_edge": {
        if (m.from && m.to) overlay.removedEdges.push({ from: m.from, to: m.to });
        else if (m.target && m.to) overlay.removedEdges.push({ from: m.target, to: m.to });
        break;
      }
      case "split_item": {
        if (m.target) overlay.archivedNodeIds.push(m.target);
        for (const spec of m.into ?? []) {
          const id = `${spec.kind}/${spec.name}`;
          overlay.addedNodeIds.push(id);
          overlay.addedNodes.push({
            id,
            kind: spec.kind,
            name: spec.name,
            title: spec.title || spec.name,
            status: "backlog",
          });
        }
        break;
      }
      default:
        // update_item / change_priority / interrupt_in_progress don't
        // change graph topology visibly — they're skipped from the overlay.
        break;
    }
  }
  return overlay;
}
