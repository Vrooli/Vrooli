import { useMemo } from "react";
import type { TFunction } from "i18next";

import type { DesignDocument, DesignFrame } from "../design-types";
import { claimsFor } from "../studio/claims";
import { stateAudit } from "../studio/checks";
import { visibleLayerRows } from "../studio/model";
import { designCoverage, frameCoverage, type Coverage } from "../studio/rcl";
import { findNode } from "../templates/spec";
import type { FrameAuditMap } from "../studio/checks";

const EMPTY_COVERAGE: Coverage = { rcl: 0, custom: 0, total: 0, ratio: 0 };

/** Derive the workspace editor's layer, coverage, and quality summaries together. */
export function useWorkspaceStudioPageMetrics({
  design,
  frames,
  frame,
  selectedId,
  collapsed,
  audits,
  translate,
}: {
  design: DesignDocument | null | undefined;
  frames: DesignFrame[];
  frame: DesignFrame | undefined;
  selectedId: string | null;
  collapsed: Set<string>;
  audits: FrameAuditMap;
  translate: TFunction;
}) {
  const rows = useMemo(
    () => (frame ? visibleLayerRows(frame.spec, collapsed) : []),
    [collapsed, frame],
  );
  const coverage = useMemo(
    () => (design ? designCoverage(design.frames) : EMPTY_COVERAGE),
    [design],
  );
  const frameCoverageValue = useMemo(
    () => (frame ? frameCoverage(frame.spec) : EMPTY_COVERAGE),
    [frame],
  );
  const selected = useMemo(
    () => (frame && selectedId ? findNode(frame.spec, selectedId) : undefined),
    [frame, selectedId],
  );
  const stateGap = useMemo(
    () => (frame ? stateAudit(frame.spec) : { total: 0, covered: 0, missing: [] }),
    [frame],
  );
  const claims = useMemo(() => (frame ? claimsFor(frame.spec, translate) : []), [frame, translate]);
  const issueCount = useMemo(() => {
    const claimIssues = claims.filter(
      (claim) => claim.verdict === "warn" || claim.verdict === "fail",
    ).length;
    const overflowFrames = frames.filter((entry) => {
      const audit = audits[entry.id];
      return Boolean(audit?.measured && audit.overflow > 0);
    }).length;
    const targetsUnder = frames.some((entry) => (audits[entry.id]?.buttons?.under ?? 0) > 0);
    return (
      claimIssues +
      (overflowFrames > 0 ? 1 : 0) +
      (targetsUnder ? 1 : 0) +
      (stateGap.missing.length > 0 ? 1 : 0)
    );
  }, [audits, claims, frames, stateGap.missing.length]);

  return { rows, coverage, frameCoverageValue, selected, stateGap, claims, issueCount };
}
