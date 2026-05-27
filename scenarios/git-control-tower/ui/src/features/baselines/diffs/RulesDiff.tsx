// RulesDiff (Plan B §4.3) — scenario-auditor rules violation-set delta for one
// baseline. Items are rule violations added (regressions/new) or cleared.

import { DiffSection, SurfaceDiffBody } from "../parts";
import type { SurfaceDiff } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

export function RulesDiff({ diff }: { diff: SurfaceDiff }) {
  return (
    <DiffSection surfaceId="rules" diff={diff}>
      <SurfaceDiffBody diff={diff} cleanLabel="Rules match the baseline." />
    </DiffSection>
  );
}
