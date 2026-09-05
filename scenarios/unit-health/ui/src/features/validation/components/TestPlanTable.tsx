import type {
  ExecutionPlan,
  TestWorkspace,
} from "@vrooli/proto-types/unit-health/v1/validation/validation_pb";

import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { Panel, Pill } from "./shared";
import { statusToneClass } from "./tone";

/**
 * TestPlanTable joins discovered workspaces with the planned commands. It flags
 * workspaces whose framework diverges from the canonical framework and shows
 * the planned timeout per workspace (falling back to the workspace metadata
 * when the plan omits it).
 */
export function TestPlanTable({
  workspaces,
  plan,
}: {
  workspaces: TestWorkspace[];
  plan?: ExecutionPlan;
}) {
  const { t } = useTranslation();
  const commandByWorkspace = new Map(
    (plan?.commands ?? []).map((command) => [command.workspaceId, command]),
  );

  return (
    <Panel title={t(strings.validation.testPlanTitle)} testId={selectors.validationWorkbench.testPlan}>
      {workspaces.length === 0 ? (
        <p
          data-testid={selectors.validationWorkbench.testPlanEmpty}
          className="text-sm text-app-muted-foreground"
        >
          {t(strings.validation.testPlanEmpty)}
        </p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="text-xs uppercase text-app-muted-foreground">
              <tr>
                <th className="py-1 pr-3">{t(strings.validation.colWorkspace)}</th>
                <th className="py-1 pr-3">{t(strings.validation.colLanguage)}</th>
                <th className="py-1 pr-3">{t(strings.validation.colFramework)}</th>
                <th className="py-1 pr-3">{t(strings.validation.colTestCommand)}</th>
                <th className="py-1 pr-3">{t(strings.validation.colCoverageCommand)}</th>
                <th className="py-1 pr-3">{t(strings.validation.colTimeout)}</th>
              </tr>
            </thead>
            <tbody>
              {workspaces.map((workspace) => {
                const command = commandByWorkspace.get(workspace.id);
                const noncanonical =
                  workspace.canonicalFramework !== "" &&
                  workspace.framework !== workspace.canonicalFramework;
                const timeoutSeconds = command?.timeoutSeconds ?? 0;
                return (
                  <tr
                    key={workspace.id}
                    data-testid={selectors.validationWorkbench.workspaceRow({ id: workspace.id })}
                    className="border-t border-app-border align-top"
                  >
                    <td className="py-2 pr-3 font-medium">
                      <div className="flex items-center gap-2">
                        {workspace.id}
                        <Pill tone={statusToneClass(workspace.status)}>{workspace.status}</Pill>
                      </div>
                    </td>
                    <td className="py-2 pr-3">{workspace.language}</td>
                    <td className="py-2 pr-3">
                      {workspace.framework}
                      {noncanonical && (
                        <span className="ml-1 text-xs text-amber-600 dark:text-amber-400">
                          {t(strings.validation.noncanonicalFramework, {
                            canonical: workspace.canonicalFramework,
                          })}
                        </span>
                      )}
                    </td>
                    <td className="py-2 pr-3 font-mono text-xs">{command?.command || workspace.testCommand}</td>
                    <td className="py-2 pr-3 font-mono text-xs">{workspace.coverageCommand || "—"}</td>
                    <td className="py-2 pr-3">
                      {timeoutSeconds > 0
                        ? t(strings.validation.timeoutSeconds, { seconds: timeoutSeconds })
                        : "—"}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </Panel>
  );
}
