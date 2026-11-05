import * as React from "react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { cn } from "../../lib/utils";
import { Button } from "../../components/ui/button";
import { Badge } from "../../components/ui/badge";
import { EmptyState } from "../../components/EmptyState";
import { ErrorState } from "../../components/ErrorState";
import { LoadingState } from "../../components/LoadingState";
import { SeverityBadge } from "../../components/SeverityBadge";
import { useListFixtures, useValidateFixture } from "./controllers/useFixtures";

/**
 * Fixture validator tab. Lists the golden fixtures the server ships and runs a
 * server-side validation (re-extract + byte-compare against
 * expected-graph.json), showing pass/fail plus a diff. The comparison happens
 * on the server — the browser never reads the fixture files.
 */
export function FixturesTab() {
  const { t } = useTranslation();
  const list = useListFixtures();
  const validate = useValidateFixture();
  const [activeName, setActiveName] = React.useState<string | null>(null);

  const runValidate = (name: string) => {
    setActiveName(name);
    validate.mutate(name);
  };

  return (
    <div data-testid={selectors.features.fixtures.root} className="flex flex-col gap-3">
      <div className="flex flex-col gap-1">
        <h3 className="text-lg font-semibold">{t(strings.fixtures.title)}</h3>
        <p className="text-sm text-app-muted-foreground">{t(strings.fixtures.description)}</p>
      </div>

      {list.isPending ? (
        <div data-testid={selectors.features.fixtures.loading}>
          <LoadingState label={t(strings.shared.loading.label)} />
        </div>
      ) : list.isError ? (
        <div data-testid={selectors.features.fixtures.error}>
          <ErrorState
            title={t(strings.shared.error.title)}
            message={list.error instanceof Error ? list.error.message : String(list.error)}
            retryLabel={t(strings.shared.error.retry)}
            onRetry={() => void list.refetch()}
          />
        </div>
      ) : list.data.fixtures.length === 0 ? (
        <div data-testid={selectors.features.fixtures.empty}>
          <EmptyState title={t(strings.fixtures.empty)} />
        </div>
      ) : (
        <ul data-testid={selectors.features.fixtures.list} className="flex flex-col gap-2">
          {list.data.fixtures.map((fixture) => {
            const isActive = activeName === fixture.name;
            return (
              <li
                key={fixture.name}
                data-testid={selectors.features.fixtures.item({ name: fixture.name })}
                className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-3 backdrop-blur-sm"
              >
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-mono text-sm text-app-foreground">{fixture.name}</span>
                  <span className="font-mono text-xs text-app-muted-foreground">{fixture.path}</span>
                  <Badge variant={fixture.hasExpected ? "success" : "warning"}>
                    {fixture.hasExpected
                      ? t(strings.fixtures.hasExpected)
                      : t(strings.fixtures.noExpected)}
                  </Badge>
                  <Button
                    size="sm"
                    className="ms-auto"
                    disabled={!fixture.hasExpected || (isActive && validate.isPending)}
                    onClick={() => runValidate(fixture.name)}
                  >
                    {isActive && validate.isPending
                      ? t(strings.fixtures.validating)
                      : t(strings.fixtures.validate)}
                  </Button>
                </div>

                {isActive && validate.isError ? (
                  <ErrorState
                    title={t(strings.shared.error.title)}
                    message={validate.error.message}
                    retryLabel={t(strings.shared.error.retry)}
                    onRetry={() => runValidate(fixture.name)}
                  />
                ) : null}

                {isActive && validate.data ? (
                  <div
                    data-testid={selectors.features.fixtures.result}
                    className="flex flex-col gap-2"
                  >
                    <div className="flex flex-wrap items-center gap-2 text-xs">
                      <SeverityBadge
                        level={validate.data.passed ? "info" : "high"}
                        label={
                          validate.data.passed
                            ? t(strings.fixtures.passed)
                            : t(strings.fixtures.failed)
                        }
                      />
                      <span className="text-app-muted-foreground">
                        {validate.data.passed
                          ? t(strings.fixtures.passMessage)
                          : t(strings.fixtures.failMessage)}
                      </span>
                    </div>
                    <div className="flex flex-wrap gap-3 text-xs text-app-muted-foreground">
                      <span>
                        {t(strings.fixtures.expectedBytes, { count: Number(validate.data.expectedBytes) })}
                      </span>
                      <span>
                        {t(strings.fixtures.actualBytes, { count: Number(validate.data.actualBytes) })}
                      </span>
                      <span className="font-mono">
                        {t(strings.fixtures.hashLabel)} {validate.data.graphHash.slice(0, 12)}…
                      </span>
                    </div>
                    {!validate.data.passed && validate.data.diff.length > 0 ? (
                      <>
                      <p className="text-xs font-semibold text-app-muted-foreground">
                        {t(strings.fixtures.diffTitle)}
                      </p>
                      <pre
                        data-testid={selectors.features.fixtures.diff}
                        className={cn(
                          "max-h-80 overflow-auto rounded-control border border-app-border bg-app-surface-muted p-3 text-xs",
                        )}
                      >
                        {validate.data.diff}
                      </pre>
                      </>
                    ) : null}
                  </div>
                ) : null}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
