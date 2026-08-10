import { useCallback, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useActionMutation } from "../../hooks/useActionMutation";
import { errorMessageOf } from "../../lib/error-utils";
import { Check, Copy, ExternalLink, FileText, List, Loader2 } from "lucide-react";
import { cn } from "../../lib/utils";
import { defaultQueryOptions } from "../../lib";
import { MarkdownRenderer } from "../markdown/MarkdownRenderer";
import { extractHeadings } from "../../lib/heading-utils";
import { backlogService } from "../../services";
import { planWorkshopService } from "../../services/plan-workshop-service";
import type { BacklogKind } from "../../types";
import { useModalBehavior } from "../../hooks/useModalBehavior";
import { Button } from "../ui/button";
import { ErrorState } from "../ui/error-state";
import { selectors } from "../../consts/selectors";
import { isApiError } from "../../lib/api-client";

export interface PlanPanelProps {
  backlogKind: BacklogKind;
  backlogName: string;
  className?: string;
  onAuthorPlan?: () => void;
  authorPlanPending?: boolean;
  authorPlanError?: string | null;
}

const TOC_ITEM_STYLES: Record<number, string> = {
  1: "pl-3 font-medium text-slate-200",
  2: "pl-5 text-slate-400",
  3: "pl-8 text-slate-500 text-xs",
};

export function PlanPanel({
  backlogKind,
  backlogName,
  className,
  onAuthorPlan,
  authorPlanPending = false,
  authorPlanError,
}: PlanPanelProps) {
  const [copySuccess, setCopySuccess] = useState(false);
  const [tocOpen, setTocOpen] = useState(false);
  const [actionMessage, setActionMessage] = useState<string | null>(null);
  const queryClient = useQueryClient();
  const tocRef = useRef<HTMLDivElement>(null);

  useModalBehavior({
    isOpen: tocOpen,
    onClose: () => setTocOpen(false),
    ref: tocRef,
    delayClickOutside: true,
  });

  const {
    data,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ["backlog-plan-render", backlogKind, backlogName],
    queryFn: () => backlogService.getRenderedPlan(backlogKind, backlogName),
    ...defaultQueryOptions,
  });

  const markdown = data?.markdown ?? "";
  const headings = extractHeadings(markdown);
  const planAbsent = isApiError(error) && error.code === "plan_ref_not_found";
  const label = "plan";
  const itemQuery = useQuery({
    queryKey: ["backlog-item", backlogKind, backlogName],
    queryFn: () => backlogService.get(backlogKind, backlogName),
    enabled: Boolean(data?.planRef),
    ...defaultQueryOptions,
  });
  const refreshAcceptance = () => {
    void itemQuery.refetch();
    void queryClient.invalidateQueries({ queryKey: ["backlog-item", backlogKind, backlogName] });
  };
  // These report through the panel's own status line, which stays visible
  // beside the plan; a toast would restate the same event out of context.
  const accept = useActionMutation({
    mutationFn: () => planWorkshopService.acceptPlan(backlogKind, backlogName),
    errorMessage: "Unable to accept this plan.",
    silentError: true,
    source: "PlanPanel.accept",
    onSuccess: () => { setActionMessage("Plan accepted. Queueing will recheck this exact revision and scope."); refreshAcceptance(); },
    onError: (cause) => setActionMessage(errorMessageOf(cause, "Unable to accept this plan.")),
  });
  const unaccept = useActionMutation({
    mutationFn: () => planWorkshopService.unacceptPlan(backlogKind, backlogName),
    errorMessage: "Unable to clear plan acceptance.",
    silentError: true,
    source: "PlanPanel.unaccept",
    onSuccess: () => { setActionMessage("Plan acceptance cleared."); refreshAcceptance(); },
    onError: (cause) => setActionMessage(errorMessageOf(cause, "Unable to clear plan acceptance.")),
  });
  const runReview = useActionMutation({
    mutationFn: async () => {
      const opened = await planWorkshopService.open({ kind: "backlog_item", ref: `${backlogKind}/${backlogName}` });
      return planWorkshopService.startReview(opened.id);
    },
    errorMessage: "Unable to start plan review.",
    silentError: true,
    source: "PlanPanel.runReview",
    onSuccess: () => setActionMessage("Plan review started. Its findings and proposals will appear in Decide."),
    onError: (cause) => setActionMessage(errorMessageOf(cause, "Unable to start plan review.")),
  });
  const accepted = itemQuery.data?.planAcceptance;

  const handleCopy = useCallback(async () => {
    if (!markdown) return;
    await navigator.clipboard.writeText(markdown);
    setCopySuccess(true);
    setTimeout(() => setCopySuccess(false), 2000);
  }, [markdown]);

  const handleTocJump = useCallback((id: string) => {
    setTocOpen(false);
    document.getElementById(id)?.scrollIntoView({ behavior: "smooth", block: "start" });
  }, []);

  if (isLoading) {
    return (
      <div className={cn("flex items-center justify-center py-20", className)}>
        <Loader2 className="h-6 w-6 animate-spin text-slate-400" />
      </div>
    );
  }

  if (planAbsent || (!isLoading && !error && !markdown)) {
    return (
      <div
        className={cn("flex flex-col items-center justify-center gap-3 py-20 text-center", className)}
        data-testid={selectors.backlogDetails.promptPanel}
      >
        <FileText className="h-10 w-10 text-slate-600" />
        <p className="text-sm font-medium text-slate-400">No {label} yet</p>
        <p className="text-xs text-slate-500">
          Author an actionable plan through plan.author, then start plan review and accept the canonical plan before queueing.
        </p>
        {onAuthorPlan && (
          <Button type="button" size="sm" onClick={onAuthorPlan} disabled={authorPlanPending} data-testid="backlog-plan-author-cta">
            {authorPlanPending ? "Starting authoring…" : "Author plan"}
          </Button>
        )}
        {authorPlanError && <p className="text-xs text-red-300" role="alert">{authorPlanError}</p>}
      </div>
    );
  }

  if (error) {
    return (
      <div className={cn("px-4 py-8", className)}>
        <ErrorState error={error} onRetry={() => refetch()} />
      </div>
    );
  }

  return (
    <div className={cn("flex flex-col", className)} data-testid={selectors.backlogDetails.promptPanel}>
      <div className="flex items-center gap-1.5 border-b border-slate-800 px-4 py-2">
        {headings.length > 0 && (
          <div ref={tocRef} className="relative">
            <Button
              variant="outline"
              size="icon"
              className={cn(tocOpen && "bg-slate-700 text-slate-100")}
              onClick={() => setTocOpen((prev) => !prev)}
              aria-label="Table of contents"
              title="Table of contents"
            >
              <List className="h-3.5 w-3.5" />
            </Button>
            {tocOpen && (
              <nav
                className="absolute left-0 top-full z-50 mt-1 w-56 overflow-hidden rounded-lg border border-white/10 bg-slate-900/95 shadow-xl backdrop-blur-sm animate-in fade-in-0 zoom-in-95 duration-100"
                aria-label="Table of contents"
                data-testid="toc-popover"
              >
                <div className="max-h-72 overflow-y-auto py-1.5">
                  {headings.map((heading, index) => (
                    <div key={`${heading.id}-${heading.line}`}>
                      {heading.level === 1 && index > 0 && <div className="mx-3 my-1 border-t border-white/5" />}
                      <button
                        className={cn(
                          "block w-full truncate py-1 pr-3 text-left text-[13px] transition-colors",
                          "hover:bg-white/5 hover:text-slate-100",
                          TOC_ITEM_STYLES[heading.level],
                        )}
                        onClick={() => handleTocJump(heading.id)}
                      >
                        {heading.text}
                      </button>
                    </div>
                  ))}
                </div>
              </nav>
            )}
          </div>
        )}

        <div className="min-w-0 flex-1 truncate text-xs text-slate-500">{data?.path}</div>

        {accepted ? (
          <span className="hidden text-xs text-emerald-300 sm:inline">Accepted {new Date(accepted.acceptedAt).toLocaleDateString()}</span>
        ) : null}

        <Button
          variant="outline"
          size="sm"
          disabled={accept.isPending || unaccept.isPending}
          title={accepted ? "Clear the recorded acceptance before changing queue readiness" : "Accept this exact canonical plan revision"}
          onClick={() => accepted ? unaccept.mutate() : accept.mutate()}
        >
          {accept.isPending || unaccept.isPending ? "Saving…" : accepted ? "Un-accept" : "Accept plan"}
        </Button>

        <Button
          variant="outline"
          size="sm"
          disabled={runReview.isPending}
          title={runReview.isPending ? "A plan review is being started" : "Start a plan review; resulting decisions appear in Decide"}
          onClick={() => runReview.mutate()}
        >
          {runReview.isPending ? "Starting…" : "Run plan review"}
        </Button>

        {data?.planRef?.slug && (
          <Button
            variant="outline"
            size="icon"
            aria-label="Open in plan-manager"
            title="Open in plan-manager"
            onClick={() => window.open(`/plan-manager/plans/${data.planRef?.slug}`, "_blank", "noopener,noreferrer")}
          >
            <ExternalLink className="h-3.5 w-3.5" />
          </Button>
        )}

        <Button
          variant="outline"
          size="icon"
          className={cn(copySuccess && "text-green-400")}
          onClick={handleCopy}
          aria-label={`Copy ${label}`}
          title={copySuccess ? "Copied!" : `Copy ${label}`}
        >
          {copySuccess ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
        </Button>
      </div>

      {data?.qualityStatus && data.qualityStatus !== "clean" && (
        <div className="border-b border-amber-500/20 bg-amber-500/10 px-4 py-2 text-xs text-amber-200">
          <span className="font-medium">Plan quality: {data.qualityStatus}</span>
          {data.qualityFindings && data.qualityFindings.length > 0 && (
            <span className="ml-2 text-amber-100/80">{data.qualityFindings.join("; ")}</span>
          )}
        </div>
      )}

      {actionMessage && <p className="border-b border-slate-800 px-4 py-2 text-xs text-slate-300" role="status">{actionMessage}</p>}

      <div className="flex-1 overflow-y-auto bg-transparent">
        <MarkdownRenderer content={markdown} className="px-4 py-4" />
      </div>
    </div>
  );
}
