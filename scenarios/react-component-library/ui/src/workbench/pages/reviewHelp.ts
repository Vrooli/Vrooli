/**
 * Review help copy.
 *
 * This is domain documentation, not UI chrome: it explains how the queue is
 * produced and what a reviewer is expected to do. It lives next to the page so
 * it changes with the page's contract. Button labels still use i18n.
 */
export const REVIEW_HELP_TITLE = "How Review works";

export const REVIEW_HELP_INTRO =
  "Review is a decision queue. Each row is a check that failed on the newest evidence for a current component version. It is not a list of bugs and it is not the same thing as Coverage: Review asks what someone should do about a failure, while Coverage shows the whole evidence picture.";

export interface ReviewHelpSection {
  title: string;
  body?: string;
  bullets?: string[];
}

export const REVIEW_HELP_SECTIONS: ReviewHelpSection[] = [
  {
    title: "What a row means",
    bullets: [
      "Asset — the catalog asset whose check failed.",
      "Gate — the independent check family, such as documentation, accessibility or restyle-contract. One asset can have many rows because every gate is a separate check.",
      "Severity — the consequence if the failure is left unresolved: blocker blocks a release, high needs a decision soon, medium is a quality gap, low and info are minor records.",
      "Age — when the newest failing evidence was recorded.",
      "Message — the failure summary. Open Evidence for the raw target, version, result, timestamp and measurement.",
    ],
  },
  {
    title: "Why the number is larger than the asset count",
    body: "The queue counts checks, not assets. A single asset can contribute several rows because each gate fails independently. The queue also leaves out passing checks, unmeasured or skipped checks, corpus probes and superseded versions; those remain visible in Coverage.",
  },
  {
    title: "Recording a decision",
    bullets: [
      "Fix — records intent to repair. The row stays open until new evidence passes.",
      "Acknowledge — records that the failure is known and owned. The row leaves the open queue and stays in decision history.",
      "Accept risk — records that the team will live with the failure. The reason becomes the acceptance record.",
      "Retire — archives an item that is no longer relevant. Use it when the asset or version is gone, not to silence a live failure.",
    ],
    body: "Reason and actor are required. The actor is recorded in the decision history; it is not an authenticated identity.",
  },
  {
    title: "How an item actually clears",
    body: "Review is derived from evidence, not edited by hand. When the check is rerun and the newest result passes, the row disappears on the next projection. Acknowledge, Accept risk and Retire remove a row from the open queue even if the evidence still fails. Docs opens the gate documentation; the governed check is usually `react-component-library asset check <asset-id>`.",
  },
  {
    title: "Reading the evidence",
    body: "Evidence JSON contains target, version, result, recorded_at and measurement. The row exists because the newest observation for this asset, gate and version has a failing result. If the evidence itself looks wrong, fix the evidence producer; hiding the row does not fix the check.",
  },
  {
    title: "What this page does not do",
    body: "It does not edit component source, rerun gates, or mark a failing check as fixed. It records intent and ownership, then routes you to the asset and its evidence so the real repair can happen in the owning workflow.",
  },
];
