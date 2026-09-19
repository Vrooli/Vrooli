import type { CoverageTarget } from "@vrooli/proto-types/unit-health/v1/validation/validation_pb";

import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { Panel, Pill } from "./shared";
import { shortPath, statusToneClass } from "./tone";

/**
 * CoverageDashboard groups coverage targets by surface, showing a per-surface
 * roll-up (summed covered/total lines + aggregate percent) and the per-file
 * rows beneath it with covered/total/percent/threshold/status.
 */
export function CoverageDashboard({ coverage }: { coverage: CoverageTarget[] }) {
  const { t } = useTranslation();

  const bySurface = new Map<string, CoverageTarget[]>();
  for (const target of coverage) {
    const list = bySurface.get(target.surfaceId) ?? [];
    list.push(target);
    bySurface.set(target.surfaceId, list);
  }

  return (
    <Panel title={t(strings.validation.coverageTitle)} testId={selectors.validationWorkbench.coverage}>
      {coverage.length === 0 ? (
        <p
          data-testid={selectors.validationWorkbench.coverageEmpty}
          className="text-sm text-app-muted-foreground"
        >
          {t(strings.validation.coverageEmpty)}
        </p>
      ) : (
        <div className="flex flex-col gap-4">
          {[...bySurface.entries()].map(([surfaceId, targets]) => {
            const covered = targets.reduce((sum, target) => sum + Number(target.coveredLines), 0);
            const total = targets.reduce((sum, target) => sum + Number(target.totalLines), 0);
            const percent = total > 0 ? Math.round((covered / total) * 100) : 0;
            return (
              <div
                key={surfaceId}
                data-testid={selectors.validationWorkbench.coverageSurface({ id: surfaceId })}
                className="rounded-control border border-app-border bg-app-surface p-3"
              >
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-medium">{surfaceId}</span>
                  <span className="text-xs text-app-muted-foreground">
                    {t(strings.validation.coverageSurfaceRollup, { covered, total, percent })}
                  </span>
                </div>
                <div className="mt-2 overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <thead className="text-xs uppercase text-app-muted-foreground">
                      <tr>
                        <th className="py-1 pr-3">{t(strings.validation.colFile)}</th>
                        <th className="py-1 pr-3">{t(strings.validation.colCovered)}</th>
                        <th className="py-1 pr-3">{t(strings.validation.colPercent)}</th>
                        <th className="py-1 pr-3">{t(strings.validation.colThreshold)}</th>
                        <th className="py-1 pr-3">{t(strings.validation.colStatus)}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {targets.map((target) => (
                        <tr
                          key={target.id}
                          data-testid={selectors.validationWorkbench.coverageRow({ id: target.id })}
                          className="border-t border-app-border"
                        >
                          <td className="py-2 pr-3 font-mono text-xs">{shortPath(target.filePath)}</td>
                          <td className="py-2 pr-3">
                            {Number(target.coveredLines)} / {Number(target.totalLines)}
                          </td>
                          <td className="py-2 pr-3">{target.coveragePercent}%</td>
                          <td className="py-2 pr-3">{target.threshold}%</td>
                          <td className="py-2 pr-3">
                            <Pill tone={statusToneClass(target.status)}>{target.status}</Pill>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </Panel>
  );
}
