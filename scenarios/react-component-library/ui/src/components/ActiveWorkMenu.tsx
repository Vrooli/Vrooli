/** @vrooliComponentSource overlays.popover */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Activity, RotateCcw, Square } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useLocation } from "react-router-dom";

import { workflowsClient } from "../api/workflows";
import { selectors } from "../consts/selectors";
import { useTranslation } from "../i18n";
import { assetInfoTab, assetPath } from "../routes";
import { Button } from "./Button";
import { Pressable } from "./Pressable";

export function ActiveWorkMenu() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const location = useLocation();
  const currentTab = location.pathname.startsWith("/assets/")
    ? assetInfoTab(new URLSearchParams(location.search))
    : undefined;
  const query = useQuery({
    queryKey: ["workflows", "recent"],
    queryFn: () => workflowsClient.listWorkflows({ activeOnly: false, limit: 20 }),
    // React Query owns this bounded lifecycle: terminal history is shown, but
    // polling stops once no run can make progress.
    refetchInterval: (state) =>
      state.state.data?.workflows?.some((workflow) => workflow.canStop) ? 15_000 : false,
  });
  const stop = useMutation({
    mutationFn: (id: string) => workflowsClient.stopWorkflow({ id }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["workflows"] }),
  });
  const retry = useMutation({
    mutationFn: (id: string) =>
      workflowsClient.retryWorkflow({ id, idempotencyKey: `ui-retry:${id}:${Date.now()}` }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["workflows"] }),
  });
  const workflows = query.data?.workflows ?? [];
  const activeCount = workflows.filter((workflow) => workflow.canStop).length;
  const closeAndRestoreFocus = useCallback(() => {
    setOpen(false);
    triggerRef.current?.focus();
  }, []);
  // A popover that traps no focus still owes the keyboard and the pointer a way
  // out: Escape returns focus to the trigger, and a press outside dismisses it.
  useEffect(() => {
    if (!open) return undefined;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.stopPropagation();
        closeAndRestoreFocus();
      }
    };
    const onPointerDown = (event: MouseEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) setOpen(false);
    };
    document.addEventListener("keydown", onKeyDown);
    document.addEventListener("mousedown", onPointerDown);
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      document.removeEventListener("mousedown", onPointerDown);
    };
  }, [closeAndRestoreFocus, open]);
  return (
    <div ref={containerRef} className="relative">
      {/* Pressable, not a raw <button>: the trigger needs the shared control
          treatment (tap target, hover/active/:focus-visible, disabled opacity,
          token-backed motion) but keeps its own content layout, which is why it
          composes Pressable rather than Button. */}
      <Pressable
        ref={triggerRef}
        tone="ghost"
        size="sm"
        density="compact"
        data-testid={selectors.workflows.activeMenu}
        onClick={() => setOpen((value) => !value)}
        aria-expanded={open}
        aria-haspopup="dialog"
      >
        <Activity aria-hidden className="h-icon-sm w-icon-sm" />
        {activeCount > 0 && <span>{activeCount}</span>}
        <span className="sr-only">{t("workflows.active", { defaultValue: "Active work" })}</span>
      </Pressable>
      {open && (
        <section
          role="dialog"
          aria-label={t("workflows.active", { defaultValue: "Active work" })}
          className="absolute end-0 z-50 mt-space-2xs w-sidebar rounded-panel border border-app-border bg-app-surface p-space-xs shadow-lg"
        >
          <h2 className="text-sm font-semibold">
            {t("workflows.active", { defaultValue: "Active work" })}
          </h2>
          {workflows.length === 0 ? (
            <p className="mt-space-2xs text-xs text-app-muted-foreground">
              {t("workflows.none", { defaultValue: "No active assisted work." })}
            </p>
          ) : (
            <ul className="mt-space-2xs space-y-space-2xs">
              {workflows.map((workflow) => (
                <li
                  key={workflow.id}
                  className="rounded-control bg-app-surface-muted p-space-2xs text-xs"
                >
                  <div className="flex items-center justify-between gap-space-2xs">
                    <Link
                      to={
                        workflow.assetId
                          ? assetPath(workflow.assetId, currentTab ? { tab: currentTab } : {})
                          : "/"
                      }
                      onClick={() => setOpen(false)}
                      className="truncate font-medium"
                    >
                      {workflow.assetId || workflow.sourceScenario || workflow.targetScenario}
                    </Link>
                    <span>{workflow.status}</span>
                  </div>
                  <p className="mt-space-3xs truncate text-app-muted-foreground">
                    {workflow.error ||
                      workflow.summary ||
                      workflow.targetScenario ||
                      workflow.sourcePath}
                  </p>
                  <div className="mt-space-2xs flex gap-space-3xs">
                    {/* Button carries the in-flight state these controls previously
                        dropped on the floor: a mutation could be fired twice with no
                        visible acknowledgement. */}
                    {workflow.canStop && (
                      <Button
                        variant="ghost"
                        size="xs"
                        density="compact"
                        data-testid={`workflow-stop-${workflow.id}`}
                        icon={<Square aria-hidden className="h-icon-xs w-icon-xs" />}
                        pending={stop.isPending && stop.variables === workflow.id}
                        pendingLabel={t("workflows.stopping", { defaultValue: "Stopping…" })}
                        onClick={() => stop.mutate(workflow.id)}
                      >
                        {t("workflows.stop", { defaultValue: "Stop" })}
                      </Button>
                    )}
                    {workflow.canRetry && (
                      <Button
                        variant="ghost"
                        size="xs"
                        density="compact"
                        data-testid={`workflow-retry-${workflow.id}`}
                        icon={<RotateCcw aria-hidden className="h-icon-xs w-icon-xs" />}
                        pending={retry.isPending && retry.variables === workflow.id}
                        pendingLabel={t("workflows.retrying", { defaultValue: "Retrying…" })}
                        onClick={() => retry.mutate(workflow.id)}
                      >
                        {t("workflows.retry", { defaultValue: "Retry" })}
                      </Button>
                    )}
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>
      )}
    </div>
  );
}
