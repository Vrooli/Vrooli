import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { modelsClient, type BackendStatus } from "../../api/models";
import { errorMessage } from "../../lib/errorMessage";

const BACKENDS_QUERY_KEY = ["backends-doctor"] as const;

/** Per-ensure feedback surfaced inline on the backend that launched it. */
interface EnsureNotice {
  tool: string;
  jobId: string;
  eta: number;
  alreadyInstalled: boolean;
  manual: boolean;
  detail: string;
}

/**
 * Readiness state of a host-tool backend, in domain language. `installed` is
 * ready; `installable` can be fetched on demand; `manual` needs an out-of-band
 * install (pip/from-source); `unsupported` cannot run on this host. Drives the
 * affordance shown (Install button vs. guidance) — never silent.
 */
type Readiness = "installed" | "installable" | "manual" | "unsupported";

function readinessOf(b: BackendStatus): Readiness {
  if (b.hostToolReady) return "installed";
  // A manual tool has a remediation that is not a fetch; the doctor still sets
  // remediation to the host-install command for fetchable tools. We treat any
  // not-ready backend with a host tool as installable and let EnsureBackend tell
  // the truth (manual / not-applicable) on click — but pre-classify obvious
  // cloud/in-process rows (no host tool) as unsupported-for-install.
  if (!b.hostTool) return "unsupported";
  return "installable";
}

/**
 * BackendsCard lists the optional host-tool AI backends and lets the operator
 * install a missing one on demand (EnsureBackend → durable job), with live
 * job/ETA feedback and a clear ready/manual/failed outcome. Default setup never
 * downloads these (headless tenet); this is the in-product opt-in path.
 */
export function BackendsCard() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [notice, setNotice] = useState<EnsureNotice | null>(null);

  const doctorQuery = useQuery({
    queryKey: BACKENDS_QUERY_KEY,
    queryFn: () => modelsClient.doctorBackends({}),
  });

  const ensureMutation = useMutation({
    mutationFn: (tool: string) => modelsClient.ensureBackend({ tool }),
    onSuccess: (res) => {
      setNotice({
        tool: res.tool,
        jobId: res.jobId,
        eta: res.etaSeconds,
        alreadyInstalled: res.alreadyInstalled,
        manual: res.manual,
        detail: res.detail,
      });
      void queryClient.invalidateQueries({ queryKey: BACKENDS_QUERY_KEY });
    },
  });

  // Only host-tool-backed backends are installable here; cloud / in-process
  // providers have no host tool to fetch.
  const backends: BackendStatus[] = (doctorQuery.data?.backends ?? []).filter(
    (b) => b.hostTool !== "",
  );

  return (
    <section
      data-testid={selectors.backends.card}
      aria-label={t(strings.backends.title)}
      className="mt-4 rounded-xl border border-app-border bg-app-surface p-4"
    >
      <h2 className="text-sm font-medium text-app-muted-foreground">{t(strings.backends.title)}</h2>
      <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.backends.description)}</p>

      {doctorQuery.isLoading && (
        <p data-testid={selectors.backends.loading} className="mt-2 text-app-foreground">
          {t(strings.backends.loading)}
        </p>
      )}
      {doctorQuery.error && (
        <p data-testid={selectors.backends.error} className="mt-2 text-app-danger">
          {errorMessage(doctorQuery.error, t)}
        </p>
      )}
      {doctorQuery.data && backends.length === 0 && (
        <p data-testid={selectors.backends.empty} className="mt-2 text-app-foreground">
          {t(strings.backends.empty)}
        </p>
      )}

      {backends.length > 0 && (
        <ul data-testid={selectors.backends.list} className="mt-2 space-y-2 text-sm text-app-foreground">
          {backends.map((b) => {
            const readiness = readinessOf(b);
            return (
              <li key={b.name} className="rounded-lg border border-app-border p-3">
                <div className="flex flex-wrap items-center gap-2">
                  <span data-testid={selectors.backends.name} className="font-medium">
                    {b.name}
                  </span>
                  <span
                    data-testid={selectors.backends.state}
                    className={
                      readiness === "installed"
                        ? "rounded bg-app-success/15 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-app-success"
                        : "rounded bg-app-warning/15 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-app-warning"
                    }
                  >
                    {readiness === "installed" ? t(strings.backends.ready) : t(strings.backends.notInstalled)}
                  </span>
                </div>

                {b.operations.length > 0 && (
                  <p data-testid={selectors.backends.operations} className="mt-1 text-xs text-app-muted-foreground">
                    {t(strings.backends.operationsLabel)}: {b.operations.join(", ")}
                  </p>
                )}

                {readiness === "installable" && (
                  <div className="mt-2">
                    <Button
                      data-testid={selectors.backends.installButton}
                      onClick={() => ensureMutation.mutate(b.hostTool)}
                      disabled={ensureMutation.isPending && ensureMutation.variables === b.hostTool}
                    >
                      {ensureMutation.isPending && ensureMutation.variables === b.hostTool
                        ? t(strings.backends.installing)
                        : t(strings.backends.install)}
                    </Button>
                  </div>
                )}

                {b.remediation !== "" && readiness !== "installed" && (
                  <p data-testid={selectors.backends.remediation} className="mt-2 text-xs text-app-muted-foreground">
                    {t(strings.backends.remediationLabel, { cmd: b.remediation })}
                  </p>
                )}

                {notice?.tool === b.hostTool && notice.manual && (
                  <p data-testid={selectors.backends.manualHint} className="mt-2 text-xs text-app-warning">
                    {t(strings.backends.manualHint, { detail: notice.detail })}
                  </p>
                )}
                {notice?.tool === b.hostTool && !notice.manual && (
                  <p data-testid={selectors.backends.installNotice} className="mt-2 text-xs text-app-success">
                    {notice.alreadyInstalled
                      ? t(strings.backends.alreadyInstalled)
                      : t(strings.backends.installStarted, { jobId: notice.jobId, eta: notice.eta })}
                  </p>
                )}
              </li>
            );
          })}
        </ul>
      )}

      {ensureMutation.error && (
        <p data-testid={selectors.backends.error} className="mt-2 text-app-danger">
          {errorMessage(ensureMutation.error, t)}
        </p>
      )}
    </section>
  );
}
