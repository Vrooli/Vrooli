import { useCallback, useMemo, useState } from "react";

import { useTranslation } from "../../i18n";
import type { ComponentExperience } from "../../api/components";
import type { WorkbenchStory } from "../data";
import { summarizeExperience } from "../experienceSummary";
import { buildFixtureChecks } from "../component/fixtureChecks";
import { summarizeCheckRows } from "../component/contextChecks";
import type { ComponentProbe } from "../component/componentProbe";
import type { InContextReport } from "../component/InContextStage";
import type { FixturePaint } from "../component/fixtureBench";

type Mode = "fixtures" | "context";

/**
 * One projection from the selected story, its measured frame and the asset's
 * experience evidence to the rows the header and inspector show. Keeping it
 * here stops ComponentPage from owning four parallel derived values and two
 * measurement maps inline.
 */
export function useComponentPageEvidence({
  mode,
  selectedStory,
  contextReport,
  experience,
}: {
  mode: Mode;
  selectedStory?: WorkbenchStory;
  contextReport: InContextReport | null;
  experience?: ComponentExperience;
}) {
  const { t } = useTranslation();
  const [fixturePaints, setFixturePaints] = useState<Record<string, FixturePaint>>({});
  const [fixtureProbes, setFixtureProbes] = useState<Record<string, ComponentProbe>>({});

  const evidence = useMemo(() => summarizeExperience(experience), [experience]);
  const contextRows = contextReport?.rows ?? [];
  const activeStoryId = selectedStory?.id ?? "";

  const fixtureRows = useMemo(
    () =>
      buildFixtureChecks(
        fixturePaints[activeStoryId] ?? null,
        fixtureProbes[activeStoryId] ?? null,
        (key, options) => t(key as never, options as never) as unknown as string,
      ),
    [activeStoryId, fixturePaints, fixtureProbes, t],
  );
  const activeRows = mode === "fixtures" ? fixtureRows : contextRows;
  const { fitSummary, pageIssues } = useMemo(() => {
    const fit = summarizeCheckRows(activeRows, "fit");
    const page = summarizeCheckRows(activeRows, "page");
    return { fitSummary: fit, pageIssues: page.pageIssues };
  }, [activeRows]);

  const evidenceNote = useMemo(() => {
    if (!experience) return "No experience contract";
    const coverage = evidence.stateCoverage;
    if (!coverage) return "No declared states";
    if (coverage.covered < coverage.total) {
      const last = coverage.latest
        ? ` · ${t("workbench.component.ctxEvidenceLast", {
            defaultValue: "last {{date}}",
            date: coverage.latest.slice(0, 10),
          })}`
        : "";
      return t("workbench.component.ctxEvidenceRecords", {
        defaultValue: "{{count}} capture record{{plural}}{{last}}",
        count: coverage.captures,
        plural: coverage.captures === 1 ? "" : "s",
        last,
      });
    }
    return t("workbench.component.ctxEvidenceCovered", {
      defaultValue: "{{covered}}/{{total}} states captured",
      covered: coverage.covered,
      total: coverage.total,
    });
  }, [evidence.stateCoverage, experience, t]);

  const recordFixturePaint = useCallback((storyId: string, paint: FixturePaint) => {
    setFixturePaints((current) => ({ ...current, [storyId]: paint }));
  }, []);
  const recordFixtureProbe = useCallback((storyId: string, probe: ComponentProbe) => {
    setFixtureProbes((current) => ({ ...current, [storyId]: probe }));
  }, []);

  return {
    contextRows,
    contextStates: evidence.stateCoverage,
    evidenceStatus: evidence.status,
    evidenceNote,
    fixtureRows,
    fitSummary,
    pageIssues,
    recordFixturePaint,
    recordFixtureProbe,
  };
}
