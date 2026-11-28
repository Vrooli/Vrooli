import { summarizeChecks, type StateAudit } from "./checks";
import type { FrameAuditMap } from "./checks";
import type { DesignFrame } from "../design-types";
import type { TemplateNode } from "../templates/spec";
import type { Coverage } from "./rcl";
import type { StudioClaim } from "./StudioInspector";
import type { Verdict } from "../verdicts";
import { isRclLayer } from "./rcl";
import { paintedShare } from "./checks";
import { hillClimbSteps } from "./hillClimb";
import { rankReplacementCandidates } from "./nodeReplacement";

export interface StudioInspectorCheckSummary {
  overflowFrames: DesignFrame[];
  measuredFrameCount: number;
  overflowVerdict: Verdict | "unmeasured";
  targetTotal: number;
  targetUnder: number;
  targetVerdict: Verdict | "unmeasured";
  stateVerdict: Verdict;
  checks: ReturnType<typeof summarizeChecks>;
  steps: ReturnType<typeof hillClimbSteps>;
  bestReplacement?: ReturnType<typeof rankReplacementCandidates>[number];
}

export function buildStudioInspectorCheckSummary({
  coverage,
  audits,
  frames,
  stateGap,
  selected,
  claims,
}: {
  coverage: Coverage;
  audits: FrameAuditMap;
  frames: DesignFrame[];
  stateGap: StateAudit;
  selected: TemplateNode | null;
  claims: StudioClaim[];
}): StudioInspectorCheckSummary {
  const overflowFrames = frames.filter((entry) => {
    const audit = audits[entry.id];
    return Boolean(audit?.measured && audit.overflow > 0);
  });
  const measuredFrameCount = frames.filter((entry) => audits[entry.id]?.measured).length;
  const targetAudits = frames.flatMap((entry) =>
    audits[entry.id]?.buttons ? [audits[entry.id]] : [],
  );
  const targetTotal = targetAudits.reduce((sum, audit) => sum + (audit?.buttons?.total ?? 0), 0);
  const targetUnder = targetAudits.reduce((sum, audit) => sum + (audit?.buttons?.under ?? 0), 0);
  const overflowVerdict =
    measuredFrameCount === 0 ? "unmeasured" : overflowFrames.length > 0 ? "fail" : "pass";
  const targetVerdict = targetTotal === 0 ? "unmeasured" : targetUnder > 0 ? "warn" : "pass";
  const stateVerdict = stateGap.missing.length > 0 ? "warn" : "pass";
  const checks = summarizeChecks([
    ...claims.map((claim) => claim.verdict),
    overflowVerdict,
    targetVerdict,
    stateVerdict,
  ]);
  const steps = hillClimbSteps({
    coverage,
    area: paintedShare(audits),
    audits,
    frameNames: frames.map(({ id, name }) => ({ id, name })),
    stateGap,
    selectedIsCustom: Boolean(selected && !isRclLayer(selected)),
    claims,
  });

  return {
    overflowFrames,
    measuredFrameCount,
    overflowVerdict,
    targetTotal,
    targetUnder,
    targetVerdict,
    stateVerdict,
    checks,
    steps,
    bestReplacement: selected ? rankReplacementCandidates(selected)[0] : undefined,
  };
}
