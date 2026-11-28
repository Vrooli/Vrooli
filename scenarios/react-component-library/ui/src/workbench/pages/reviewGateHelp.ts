/**
 * Plain-language explanations for Review gates.
 *
 * Review used to show only a gate identifier such as `surface-discipline` and a
 * sentence that said "gate failed". That told a reviewer neither what was
 * measured nor what to do next. These strings are intentionally short: they
 * explain the check, and the finding's remediation and evidence carry the
 * specific failure.
 */
const GATE_HELP: Record<string, string> = {
  accessibility:
    "Checks keyboard use, accessible names, focus visibility, contrast and other barriers that stop people using the component.",
  interaction:
    "Runs the interaction story: the component opens, responds and settles in the way its contract says it should.",
  unit: "Runs the component's unit tests. A failure here means an expected behaviour changed or broke.",
  responsive:
    "Checks that the component keeps working across its declared viewport range instead of clipping or overflowing.",
  visual:
    "Compares the rendered component with its recorded appearance. A failure means the pixels moved without an accepted baseline.",
  rtl: "Checks right-to-left layout. A failure means spacing, direction or icon placement breaks when the page direction flips.",
  "reduced-motion":
    "Checks that motion can be removed without hiding content or breaking the interaction for people who ask for less motion.",
  "console-clean":
    "Checks that using the component does not log errors, warnings or unhandled rejections in the browser console.",
  performance: "Checks that the component stays within its declared render and payload budget.",
  documentation:
    "Checks that the component's description, API and usage notes describe what the component actually does.",
  i18n: "Checks that visible text comes from translatable strings and that no user-facing copy is hard-coded.",
  "selector-coverage":
    "Checks that automation can address the component's parts without depending on brittle CSS or DOM shape.",
  "story-grammar":
    "Checks that the component story is a real, runnable example rather than an empty or tautological shell.",
  composition:
    "Checks that composing this component with its dependencies produces a usable result, not just isolated parts.",
  "composition-contract":
    "Checks that the component's declared composition contract matches what the rendered component accepts.",
  "surface-discipline":
    "Checks that the component uses shared surfaces and elevation tokens instead of inventing its own panel styling.",
  "restyle-contract":
    "Checks that consumers can restyle the component through its declared contract without editing its source.",
  "stylesheet-key":
    "Checks that the component's stylesheet is keyed and versioned so two versions cannot silently overwrite each other.",
  "style-injection":
    "Checks that component styles are injected once and cleaned up, instead of leaking between previews or pages.",
  "version-shape":
    "Checks that the version's files and manifest match the library's version layout.",
  "version-liveness":
    "Checks that the version is still reachable and consistent with the version ledger.",
  "released-version-immutable":
    "Checks that a released version was not edited in place. Releases are immutable; changes belong in a new version.",
  "dist-resolution":
    "Checks that the published package resolves to the built files consumers actually import.",
  "evidence-freshness":
    "Checks that the verdict was measured against the current source and dependency revisions, not an older snapshot.",
  "graph-reconciled":
    "Checks that the component graph agrees with the source tree: no missing, duplicated or orphaned assets.",
  "utility-class":
    "Checks that the component uses library tokens rather than ad-hoc utility classes.",
  lifecycle:
    "Checks that the component's release and retirement state is coherent with the rest of the library.",
  integration:
    "Checks that the component works against the integration it declares, including its provider or data source.",
  api: "Checks that the component's declared API and the rendered implementation agree.",
  types: "Checks that the component's exported types resolve and match its implementation.",
  conformance: "Checks the component against the library's shared conformance rules.",
};

export function gateLabel(gate: string): string {
  return gate.replace(/-/g, " ");
}

export function gateHelp(gate: string): string {
  return (
    GATE_HELP[gate.toLowerCase()] ??
    "A library check for this asset failed. The evidence below records what was measured."
  );
}

const SEVERITY_HELP: Record<string, string> = {
  blocker: "This should block a release until it is fixed or explicitly accepted.",
  high: "This affects a release or a core journey and needs a decision soon.",
  medium: "This is a quality gap. It should be planned, not treated as an outage.",
  low: "This is a minor quality issue that can be batched with other work.",
  info: "This is recorded for awareness; it does not imply a defect.",
};

export function severityHelp(severity: string): string {
  return (
    SEVERITY_HELP[severity.toLowerCase()] ?? "The library recorded this severity for the check."
  );
}
