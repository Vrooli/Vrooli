import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { getInvestigationPolicy, updateInvestigationPolicy } from "../../api/investigation";
import { Card, SectionPanel } from "../../components/Surfaces";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

type PolicyDocument = { mode?: "shadow" | "automatic"; version?: string; rules?: unknown[]; limits?: Record<string, unknown> };

function parsePolicy(raw: string): PolicyDocument {
  try {
    const value = JSON.parse(raw) as unknown;
    if (!value || typeof value !== "object" || Array.isArray(value)) return {};
    const entries = Object.entries(value as Record<string, unknown>);
    const mode = entries.find(([key]) => key === "mode")?.[1];
    const version = entries.find(([key]) => key === "version")?.[1];
    const limits = entries.find(([key]) => key === "limits")?.[1];
    return {
      mode: mode === "shadow" || mode === "automatic" ? mode : undefined,
      version: typeof version === "string" ? version : undefined,
      limits: limits && typeof limits === "object" && !Array.isArray(limits) ? limits as Record<string, unknown> : undefined,
    };
  } catch {
    return {};
  }
}

/** Read-only by default, with an explicit reviewed mode change for operators. */
export function InvestigationPolicyPanel() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const policy = useQuery({ queryKey: ["investigation-policy"], queryFn: getInvestigationPolicy });
  const [mode, setMode] = useState<"shadow" | "automatic">("shadow");
  const parsedPolicy = policy.data?.policyJson ? parsePolicy(policy.data.policyJson) : undefined;
  useEffect(() => {
    if (policy.data?.policyJson) {
      const parsed = parsePolicy(policy.data.policyJson);
      if (parsed.mode) setMode(parsed.mode);
      else if (policy.data.mode === "automatic" || policy.data.mode === "shadow") setMode(policy.data.mode);
    }
  }, [policy.data]);
  const save = useMutation({
    mutationFn: async () => {
      if (!policy.data?.policyJson) throw new Error("policy document is unavailable");
      const document = parsePolicy(policy.data.policyJson);
      return updateInvestigationPolicy(JSON.stringify({ ...document, mode }), policy.data.version);
    },
    onSuccess: () => { void queryClient.invalidateQueries({ queryKey: ["investigation-policy"] }); },
  });

  return (
    <SectionPanel
      title={t(strings.pages.settings.investigationTitle)}
      headingId="settings-investigation-heading"
      description={t(strings.pages.settings.investigationDescription)}
    >
      {policy.isPending ? <p className="text-sm text-app-muted-foreground">{t(strings.pages.settings.investigationLoading)}</p> : null}
      {policy.isError ? <Card className="text-sm text-app-danger">{t(strings.pages.settings.investigationUnavailable)}</Card> : null}
      {policy.data ? (
        <div className="flex flex-col gap-3 text-sm">
          <div className="grid gap-3 sm:grid-cols-3">
            <div><span className="text-xs uppercase text-app-muted-foreground">{t(strings.pages.settings.investigationMode)}</span><p className="font-medium">{policy.data.mode}</p></div>
            <div><span className="text-xs uppercase text-app-muted-foreground">{t(strings.pages.settings.investigationVersion)}</span><p className="font-medium break-all">{policy.data.version}</p></div>
            <div><span className="text-xs uppercase text-app-muted-foreground">{t(strings.pages.settings.investigationActive)}</span><p className="font-medium">{policy.data.active ? t(strings.pages.settings.investigationYes) : t(strings.pages.settings.investigationNo)}</p></div>
          </div>
          <div className="grid gap-3 text-xs text-app-muted-foreground sm:grid-cols-2">
            <div><span className="uppercase">{t(strings.pages.settings.investigationDigest)}</span><p className="break-all font-mono">{policy.data.digest || t(strings.pages.settings.investigationUnknown)}</p></div>
            <div><span className="uppercase">{t(strings.pages.settings.investigationSourceScope)}</span><p className="break-all font-mono">{policy.data.sourceScope || t(strings.pages.settings.investigationUnknown)}</p></div>
          </div>
          <div className="rounded-control border border-app-border bg-app-surface-muted px-3 py-2 text-xs">
            <p className="font-medium text-app-foreground">{t(strings.pages.settings.investigationLimits)}</p>
            {parsedPolicy?.limits && Object.keys(parsedPolicy.limits).length > 0 ? (
              <ul className="mt-1 grid gap-1 sm:grid-cols-2">
                {Object.entries(parsedPolicy.limits).map(([key, value]) => <li key={key}><span className="font-mono">{key}</span>: {String(value)}</li>)}
              </ul>
            ) : <p>{t(strings.pages.settings.investigationUnknown)}</p>}
          </div>
          <fieldset className="flex flex-wrap gap-3" aria-label={t(strings.pages.settings.investigationModeLabel)}>
            <label className="flex items-center gap-2"><input type="radio" name="investigation-mode" checked={mode === "shadow"} onChange={() => setMode("shadow")} /> {t(strings.pages.settings.investigationShadow)}</label>
            <label className="flex items-center gap-2"><input type="radio" name="investigation-mode" checked={mode === "automatic"} onChange={() => setMode("automatic")} /> {t(strings.pages.settings.investigationAutomatic)}</label>
          </fieldset>
          <p className="text-xs text-app-muted-foreground">{t(strings.pages.settings.investigationNote)}</p>
          <p className="text-xs text-app-muted-foreground">{t(strings.pages.settings.investigationPendingState)}</p>
          <div className="flex items-center gap-3">
            <button type="button" className="rounded-control bg-app-primary px-3 py-1.5 text-sm font-medium text-app-primary-foreground disabled:opacity-50" disabled={save.isPending || mode === policy.data.mode} onClick={() => void save.mutateAsync()}>{save.isPending ? t(strings.pages.settings.investigationSaving) : t(strings.pages.settings.investigationSave)}</button>
            {save.isError ? <span className="text-sm text-app-danger">{save.error instanceof Error ? save.error.message : t(strings.pages.settings.investigationSaveFailed)}</span> : null}
            {save.isSuccess ? <span className="text-sm text-app-success">{t(strings.pages.settings.investigationSaved)}</span> : null}
          </div>
        </div>
      ) : null}
    </SectionPanel>
  );
}
