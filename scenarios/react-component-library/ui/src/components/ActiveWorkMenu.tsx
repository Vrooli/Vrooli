import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Activity, RotateCcw, Square } from "lucide-react";
import { useState } from "react";
import { Link, useLocation } from "react-router-dom";

import { workflowsClient } from "../api/workflows";
import { selectors } from "../consts/selectors";
import { useTranslation } from "../i18n";
import { assetInfoTab, assetPath } from "../routes";

export function ActiveWorkMenu() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const location = useLocation();
  const currentTab = location.pathname.startsWith("/assets/") ? assetInfoTab(new URLSearchParams(location.search)) : undefined;
  const query = useQuery({
    queryKey: ["workflows", "recent"],
    queryFn: () => workflowsClient.listWorkflows({ activeOnly: false, limit: 20 }),
    // React Query owns this bounded lifecycle: terminal history is shown, but
    // polling stops once no run can make progress.
    refetchInterval: (state) => state.state.data?.workflows.some((workflow) => workflow.canStop) ? 15_000 : false,
  });
  const stop = useMutation({ mutationFn: (id: string) => workflowsClient.stopWorkflow({ id }), onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["workflows"] }) });
  const retry = useMutation({ mutationFn: (id: string) => workflowsClient.retryWorkflow({ id, idempotencyKey: `ui-retry:${id}:${Date.now()}` }), onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["workflows"] }) });
  const workflows = query.data?.workflows ?? [];
  const activeCount = workflows.filter((workflow) => workflow.canStop).length;
  return <div className="relative"><button type="button" data-testid={selectors.workflows.activeMenu} onClick={() => setOpen((value) => !value)} aria-expanded={open} aria-haspopup="dialog" className="touch-target inline-flex items-center gap-1 rounded-control px-2 text-xs text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"><Activity aria-hidden className="h-4 w-4" />{activeCount > 0 && <span>{activeCount}</span>}<span className="sr-only">{t("workflows.active", { defaultValue: "Active work" })}</span></button>{open && <section role="dialog" aria-label={t("workflows.active", { defaultValue: "Active work" })} className="absolute end-0 z-50 mt-2 w-80 rounded-panel border border-app-border bg-app-surface p-3 shadow-lg"><h2 className="text-sm font-semibold">{t("workflows.active", { defaultValue: "Active work" })}</h2>{workflows.length === 0 ? <p className="mt-2 text-xs text-app-muted-foreground">{t("workflows.none", { defaultValue: "No active assisted work." })}</p> : <ul className="mt-2 space-y-2">{workflows.map((workflow) => <li key={workflow.id} className="rounded-control bg-app-surface-muted p-2 text-xs"><div className="flex items-center justify-between gap-2"><Link to={workflow.assetId ? assetPath(workflow.assetId, currentTab ? { tab: currentTab } : {}) : "/"} onClick={() => setOpen(false)} className="truncate font-medium">{workflow.assetId || workflow.sourceScenario || workflow.targetScenario}</Link><span>{workflow.status}</span></div><p className="mt-1 truncate text-app-muted-foreground">{workflow.error || workflow.summary || workflow.targetScenario || workflow.sourcePath}</p><div className="mt-2 flex gap-1">{workflow.canStop && <button type="button" onClick={() => stop.mutate(workflow.id)} className="inline-flex items-center gap-1 rounded-control px-1.5 py-1 hover:bg-app-background"><Square aria-hidden className="h-3 w-3" />{t("workflows.stop", { defaultValue: "Stop" })}</button>}{workflow.canRetry && <button type="button" onClick={() => retry.mutate(workflow.id)} className="inline-flex items-center gap-1 rounded-control px-1.5 py-1 hover:bg-app-background"><RotateCcw aria-hidden className="h-3 w-3" />{t("workflows.retry", { defaultValue: "Retry" })}</button>}</div></li>)}</ul>}</section>}</div>;
}
