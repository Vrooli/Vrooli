import { useState } from "react";
import type { CommandResult } from "@vrooli/proto-types/unit-health/v1/validation/validation_pb";

import { Button } from "../../../components/ui/button";
import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { Panel, Pill } from "./shared";
import { statusToneClass } from "./tone";

/**
 * ExecutionResults renders each executed command with its status, exit code,
 * duration, and failure class. Captured stdout/stderr excerpts are collapsed
 * behind a per-row toggle so the table stays scannable.
 */
export function ExecutionResults({ results }: { results: CommandResult[] }) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});

  return (
    <Panel title={t(strings.validation.executionTitle)} testId={selectors.validationWorkbench.execution}>
      {results.length === 0 ? (
        <p
          data-testid={selectors.validationWorkbench.executionEmpty}
          className="text-sm text-app-muted-foreground"
        >
          {t(strings.validation.executionEmpty)}
        </p>
      ) : (
        <div className="flex flex-col gap-2">
          {results.map((result) => {
            const isOpen = expanded[result.name] ?? false;
            const hasOutput =
              result.stdoutExcerpt !== "" ||
              result.stderrExcerpt !== "" ||
              result.failureReason !== "";
            return (
              <article
                key={result.name}
                data-testid={selectors.validationWorkbench.commandRow({ name: result.name })}
                className="rounded-control border border-app-border bg-app-surface p-3"
              >
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-medium">{result.name}</span>
                  <Pill tone={statusToneClass(result.status)}>{result.status}</Pill>
                  <span className="text-xs text-app-muted-foreground">
                    {t(strings.validation.colExitCode)}: {result.exitCode}
                  </span>
                  <span className="text-xs text-app-muted-foreground">
                    {t(strings.validation.durationMs, { ms: Number(result.durationMs) })}
                  </span>
                  {result.failureClass && (
                    <Pill tone="border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-300">
                      {result.failureClass}
                    </Pill>
                  )}
                  {hasOutput && (
                    <Button
                      data-testid={selectors.validationWorkbench.commandOutputToggle({
                        name: result.name,
                      })}
                      type="button"
                      variant="outline"
                      className="ml-auto h-7 px-2 text-xs"
                      onClick={() =>
                        setExpanded((prev) => ({ ...prev, [result.name]: !isOpen }))
                      }
                    >
                      {t(strings.validation.toggleOutput)}
                    </Button>
                  )}
                </div>
                <p className="mt-1 font-mono text-xs text-app-muted-foreground">{result.command}</p>
                {isOpen && (
                  <div
                    data-testid={selectors.validationWorkbench.commandOutput({ name: result.name })}
                    className="mt-2 flex flex-col gap-2 text-xs"
                  >
                    {result.failureReason && (
                      <p>
                        <span className="font-semibold">{t(strings.validation.failureReasonLabel)}:</span>{" "}
                        {result.failureReason}
                      </p>
                    )}
                    {result.stdoutExcerpt ? (
                      <div>
                        <p className="font-semibold">{t(strings.validation.stdoutLabel)}</p>
                        <pre className="mt-1 whitespace-pre-wrap rounded-control bg-app-surface-muted p-2">
                          {result.stdoutExcerpt}
                        </pre>
                      </div>
                    ) : null}
                    {result.stderrExcerpt ? (
                      <div>
                        <p className="font-semibold">{t(strings.validation.stderrLabel)}</p>
                        <pre className="mt-1 whitespace-pre-wrap rounded-control bg-app-surface-muted p-2">
                          {result.stderrExcerpt}
                        </pre>
                      </div>
                    ) : null}
                    {!result.stdoutExcerpt && !result.stderrExcerpt && !result.failureReason && (
                      <p className="text-app-muted-foreground">{t(strings.validation.noOutput)}</p>
                    )}
                  </div>
                )}
              </article>
            );
          })}
        </div>
      )}
    </Panel>
  );
}
