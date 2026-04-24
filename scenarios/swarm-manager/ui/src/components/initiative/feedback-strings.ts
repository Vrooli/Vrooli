/**
 * User-facing strings for the feedback + proposal-review surface.
 *
 * Kept in one module so copy can be revised without touching layout or
 * behavior, and so tests can assert against the constants rather than
 * snapshots of prose that shift as we iterate on wording.
 */

export const REVISE_PLACEHOLDER =
  "Ask the agent to revise the proposal… (Ctrl+Enter to send)";

export const AGENT_WORKING_LABEL = "Agent is working on a response…";

export const PARSE_ERROR_TITLE = "Agent output did not contain a readable proposal";

export const PARSE_ERROR_BODY =
  "Ask the agent to revise — its last turn was stored in the thread but no " +
  "mutation_list or full_graph block could be parsed. Describe what you want " +
  "differently and send the follow-up below.";

export const PROPOSAL_SELECT_ALL = "Select all";
export const PROPOSAL_CLEAR = "Clear";
export const PROPOSAL_REJECT = "Reject";
export const PROPOSAL_DISMISS = "Dismiss";
export const PROPOSAL_RATIONALE_PLACEHOLDER = "Optional rationale for your decision…";
