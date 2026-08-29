import { useEffect, useMemo, useState } from "react";
import type { TFunction } from "i18next";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { useApprovePairingMutation, usePairingRequestsQuery } from "./queries";

function presetLabel(name: string, t: TFunction): string {
  switch (name) {
    case "read-only":
      return t(strings.fleet.pairing.permissionReadOnly);
    case "operate":
      return t(strings.fleet.pairing.permissionOperate);
    case "full-control":
      return t(strings.fleet.pairing.permissionFull);
    default:
      return name;
  }
}

/**
 * Owner-side approval for request/approve pairing. The request id is not a
 * credential: approval still requires the key-derived words shown by the
 * joining machine. Permission choices come from the server's catalog-derived
 * presets, with the narrow read-only preset selected by default.
 */
export function PendingPairingPanel() {
  const { t } = useTranslation();
  const query = usePairingRequestsQuery();
  const approve = useApprovePairingMutation();
  const [selectedPreset, setSelectedPreset] = useState<Record<string, string>>({});
  const [confirmed, setConfirmed] = useState<Record<string, boolean>>({});

  const presets = query.data?.presets ?? [];
  const defaultPreset = useMemo(
    () => presets.find((preset) => preset.name === "read-only") ?? presets[0],
    [presets],
  );

  useEffect(() => {
    if (!defaultPreset) return;
    setSelectedPreset((current) => {
      const next = { ...current };
      for (const request of query.data?.requests ?? []) {
        if (!next[request.id]) next[request.id] = defaultPreset.name;
      }
      return next;
    });
  }, [defaultPreset, query.data?.requests]);

  if (query.isLoading || (query.data?.requests.length ?? 0) === 0) return null;

  return (
    <section
      data-testid={selectors.fleet.pairingRequests.panel}
      aria-labelledby="fleet-pairing-requests-heading"
      className="mt-3 rounded-panel border border-app-warning/40 bg-app-warning/10 p-4"
    >
      <h3 id="fleet-pairing-requests-heading" className="text-sm font-semibold text-app-foreground">
        {t(strings.fleet.pairing.requestsHeading)}
      </h3>
      <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.fleet.pairing.requestsDescription)}</p>

      <ul className="mt-3 flex flex-col gap-3">
        {(query.data?.requests ?? []).map((request) => {
          const presetName = selectedPreset[request.id] ?? defaultPreset?.name ?? "read-only";
          const preset = presets.find((candidate) => candidate.name === presetName) ?? defaultPreset;
          const isConfirmed = confirmed[request.id] === true;
          const isPending = approve.isPending && approve.variables?.requestId === request.id;
          return (
            <li
              key={request.id}
              data-testid={selectors.fleet.pairingRequests.row({ id: request.id })}
              className="rounded-control border border-app-border bg-app-background p-3"
            >
              <div className="flex flex-wrap items-start justify-between gap-2">
                <div>
                  <p className="text-sm font-medium text-app-foreground">
                    {t(strings.fleet.pairing.requestName)}: {request.name || request.id}
                  </p>
                  <p className="text-xs text-app-muted-foreground">
                    {t(strings.fleet.pairing.requestPlatform)}: {request.os || "unknown"} / {request.arch || "unknown"}
                    {request.endpoint ? ` · ${request.endpoint}` : ""}
                  </p>
                </div>
                <code
                  data-testid={selectors.fleet.pairingRequests.words({ id: request.id })}
                  className="rounded-control bg-app-surface px-2 py-1 font-mono text-sm font-semibold tracking-wide text-app-foreground"
                >
                  {request.confirmationWords.join(" ") || "—"}
                </code>
              </div>

              <p className="mt-2 text-xs text-app-muted-foreground">{t(strings.fleet.pairing.confirmationHelp)}</p>

              <div className="mt-3 grid gap-2 sm:grid-cols-2">
                <label className="flex flex-col gap-1 text-xs text-app-muted-foreground">
                  {t(strings.fleet.pairing.permissionLabel)}
                  <select
                    data-testid={selectors.fleet.pairingRequests.preset({ id: request.id })}
                    className="h-9 rounded-control border border-app-border bg-app-surface px-2 text-sm text-app-foreground"
                    value={presetName}
                    onChange={(event) => setSelectedPreset((current) => ({ ...current, [request.id]: event.target.value }))}
                    disabled={isPending}
                  >
                      {presets.map((candidate) => (
                      <option key={candidate.name} value={candidate.name}>{presetLabel(candidate.name, t)}</option>
                    ))}
                  </select>
                </label>
                <div className="text-xs text-app-muted-foreground">
                  <p>{t(strings.fleet.pairing.permissionScopes)}: {preset?.scopes.join(", ") || "none"}</p>
                  {preset && preset.withholds.length > 0 ? <p className="mt-1">{t(strings.fleet.pairing.permissionWithholds)}: {preset.withholds.join(", ")}</p> : null}
                </div>
              </div>

              <label className="mt-3 flex items-start gap-2 text-xs text-app-foreground">
                <input
                  type="checkbox"
                  data-testid={selectors.fleet.pairingRequests.wordsMatch({ id: request.id })}
                  checked={isConfirmed}
                  onChange={(event) => setConfirmed((current) => ({ ...current, [request.id]: event.target.checked }))}
                  disabled={isPending}
                />
                {t(strings.fleet.pairing.wordsMatch)}
              </label>

              <div className="mt-3 flex flex-wrap gap-2">
                <Button
                  type="button"
                  size="sm"
                  data-testid={selectors.fleet.pairingRequests.approve({ id: request.id })}
                  disabled={!isConfirmed || !preset || isPending}
                  onClick={() => approve.mutate({ requestId: request.id, approve: true, scopes: preset?.scopes, confirmationWords: request.confirmationWords })}
                >
                  {isPending ? t(strings.fleet.pairing.approving) : t(strings.fleet.pairing.approve)}
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  data-testid={selectors.fleet.pairingRequests.reject({ id: request.id })}
                  disabled={isPending}
                  onClick={() => approve.mutate({ requestId: request.id, approve: false })}
                >
                  {isPending ? t(strings.fleet.pairing.rejecting) : t(strings.fleet.pairing.reject)}
                </Button>
              </div>
              {approve.error && approve.variables?.requestId === request.id ? (
                <p data-testid={selectors.fleet.pairingRequests.error} role="alert" className="mt-2 text-xs text-app-danger">
                  {errorMessage(approve.error, t)}
                </p>
              ) : null}
            </li>
          );
        })}
      </ul>
    </section>
  );
}
