import type { AreaShare, FrameAuditMap, StateAudit } from "./checks";
import type { Coverage } from "./rcl";
import type { Verdict } from "../verdicts";

export interface HillClimbStep {
  id: string;
  title: string;
  detail: string;
  priority: number;
  action?: "replace" | "overflow-frame" | "states";
  frameId?: string;
}

/**
 * Turns observable findings into next steps. Ordering reflects direct usability
 * impact first; it deliberately does not collapse evidence into a design score.
 */
export function hillClimbSteps(input: {
  coverage: Coverage;
  area: AreaShare | null;
  audits: FrameAuditMap;
  frameNames: readonly { id: string; name: string }[];
  stateGap: StateAudit;
  selectedIsCustom: boolean;
  claims: readonly { id: string; label?: string; verdict: Verdict; detail: string }[];
}): HillClimbStep[] {
  const steps: HillClimbStep[] = [];
  const overflowFrame = input.frameNames.find((frame) => {
    const audit = input.audits[frame.id];
    return audit?.measured && audit.overflow > 0;
  });
  if (overflowFrame)
    steps.push({
      id: "overflow",
      title: `Fix horizontal overflow in ${overflowFrame.name}`,
      detail: `${input.audits[overflowFrame.id]?.overflow ?? 0}px measured beyond the frame.`,
      priority: 100,
      action: "overflow-frame",
      frameId: overflowFrame.id,
    });
  const under = input.frameNames.reduce(
    (n, frame) => n + (input.audits[frame.id]?.buttons?.under ?? 0),
    0,
  );
  if (under > 0)
    steps.push({
      id: "targets",
      title: "Increase undersized touch targets",
      detail: `${under} measured button${under === 1 ? " is" : "s are"} below 44px.`,
      priority: 90,
    });
  if (input.stateGap.missing.length)
    steps.push({
      id: "states",
      title: "Declare component states",
      detail: `${input.stateGap.missing.length} data or input layer${input.stateGap.missing.length === 1 ? " has" : "s have"} no declared state.`,
      priority: 80,
      action: "states",
    });
  if (input.coverage.total > 0 && input.coverage.custom > 0)
    steps.push({
      id: "rcl",
      title: "Use an RCL building block for a custom layer",
      detail: `${input.coverage.rcl}/${input.coverage.total} layers map to RCL components.${input.selectedIsCustom ? " Best matches are available for the selected layer." : " Select a custom layer to see compatible replacements."}`,
      priority: 60,
      ...(input.selectedIsCustom ? { action: "replace" as const } : {}),
    });
  if (input.area && input.coverage.ratio > input.area.ratio + 0.15)
    steps.push({
      id: "painted-share",
      title: "Review large custom-painted regions",
      detail: `RCL is ${Math.round(input.coverage.ratio * 100)}% of layers and ${Math.round(input.area.ratio * 100)}% of measured painted area. This is descriptive coverage, not an art-quality judgment.`,
      priority: 50,
    });
  for (const claim of input.claims)
    if (claim.verdict !== "pass")
      steps.push({
        id: `evidence-${claim.id}`,
        title:
          claim.verdict === "unmeasured"
            ? `Capture evidence for ${claim.label ?? claim.id}`
            : `Review ${claim.label ?? claim.id}`,
        detail: claim.detail,
        priority: claim.verdict === "unmeasured" ? 30 : 45,
      });
  return steps.sort((a, b) => b.priority - a.priority || a.id.localeCompare(b.id));
}
