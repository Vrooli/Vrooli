import { useEffect, useMemo, useState } from "react";
import { FilePlus2, Link2, Loader2, Search } from "lucide-react";
import { Link } from "react-router-dom";
import { backlogDetailPath, initiativeDetailPath } from "../../app/routes/route-paths";
import {
  planService,
  type CanonicalPlanSummary,
  type PlanImportResult,
} from "../../services/plan-service";
import { cn } from "../../lib/utils";
import { BottomSheet } from "../ui/bottom-sheet";
import { Button } from "../ui/button";
import { Input } from "../ui/input";

export interface CreateWorkFromPlanDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onImported?: (result: PlanImportResult) => void;
}

type SourceMode = "existing" | "markdown";
type ContainerType = "items" | "initiative";

const MODE_OPTIONS = [
  { value: "item-level", label: "Item-level", hint: "Creates phase items without a mode-wide drain." },
  { value: "phased-plan-drain", label: "Phased drain", hint: "Binds the plan as the initiative operating-mode plan." },
  { value: "holistic-loop", label: "Holistic loop", hint: "Creates a broad initiative around the plan." },
] as const;

export function CreateWorkFromPlanDialog({ isOpen, onClose, onImported }: CreateWorkFromPlanDialogProps) {
  const [sourceMode, setSourceMode] = useState<SourceMode>("existing");
  const [plans, setPlans] = useState<CanonicalPlanSummary[]>([]);
  const [query, setQuery] = useState("");
  const [selectedPlanId, setSelectedPlanId] = useState("");
  const [sourcePath, setSourcePath] = useState("");
  const [markdown, setMarkdown] = useState("");
  const [title, setTitle] = useState("");
  const [slug, setSlug] = useState("");
  const [containerType, setContainerType] = useState<ContainerType>("items");
  const [initiativeName, setInitiativeName] = useState("");
  const [initiativeTitle, setInitiativeTitle] = useState("");
  const [initiativeDescription, setInitiativeDescription] = useState("");
  const [mode, setMode] = useState("phased-plan-drain");
  const [loadingPlans, setLoadingPlans] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<PlanImportResult | null>(null);

  useEffect(() => {
    if (!isOpen) return;
    setError(null);
    setResult(null);
    setLoadingPlans(true);
    const controller = new AbortController();
    planService.listCanonicalPlans({ signal: controller.signal })
      .then((items) => setPlans(items))
      .catch((err) => {
        if (!controller.signal.aborted) {
          setError(err instanceof Error ? err.message : "Failed to load plans.");
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoadingPlans(false);
      });
    return () => controller.abort();
  }, [isOpen]);

  const filteredPlans = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    const base = plans
      .filter((plan) => plan.id && plan.slug)
      .sort((a, b) => (b.updatedAt ?? b.createdAt ?? "").localeCompare(a.updatedAt ?? a.createdAt ?? ""));
    if (!normalized) return base.slice(0, 80);
    return base
      .filter((plan) =>
        plan.slug.toLowerCase().includes(normalized) ||
        plan.title.toLowerCase().includes(normalized) ||
        plan.id.toLowerCase().includes(normalized)
      )
      .slice(0, 80);
  }, [plans, query]);

  const selectedPlan = plans.find((plan) => plan.id === selectedPlanId || plan.slug === selectedPlanId);
  const canSubmit = sourceMode === "existing"
    ? selectedPlanId.trim().length > 0
    : sourcePath.trim().length > 0 || markdown.trim().length > 0;

  const resetAndClose = () => {
    if (submitting) return;
    onClose();
  };

  const submit = () => {
    if (!canSubmit || submitting) return;
    setSubmitting(true);
    setError(null);
    setResult(null);
    planService.importPlan({
      planId: sourceMode === "existing" ? selectedPlanId : undefined,
      sourcePath: sourceMode === "markdown" ? sourcePath.trim() || undefined : undefined,
      markdown: sourceMode === "markdown" ? markdown.trim() || undefined : undefined,
      title: sourceMode === "markdown" ? title.trim() || undefined : undefined,
      slug: sourceMode === "markdown" ? slug.trim() || undefined : undefined,
      container: {
        type: containerType,
        name: containerType === "initiative" ? initiativeName.trim() || undefined : undefined,
        title: containerType === "initiative" ? initiativeTitle.trim() || undefined : undefined,
        description: containerType === "initiative" ? initiativeDescription.trim() || undefined : undefined,
        mode: containerType === "initiative" ? mode : undefined,
      },
    })
      .then((next) => {
        setResult(next);
        onImported?.(next);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to create work from plan."))
      .finally(() => setSubmitting(false));
  };

  return (
    <BottomSheet
      isOpen={isOpen}
      onClose={resetAndClose}
      title="Create work from plan"
      description="Bind a canonical plan-manager plan to backlog work or an initiative."
      className="!max-w-3xl border-slate-700/80 bg-slate-900"
      contentClassName="px-0 py-0"
      data-testid="create-work-from-plan-dialog"
      footer={
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <p className="min-h-5 text-xs text-slate-400">
            {error ? <span className="text-red-300" data-testid="create-work-from-plan-error">{error}</span> : result ? (
              <span className="text-emerald-300" data-testid="create-work-from-plan-success">
                {result.created} created, {result.updated} updated, {result.linked} linked.
              </span>
            ) : selectedPlan ? (
              <span>{selectedPlan.phaseCount} {selectedPlan.phaseCount === 1 ? "phase" : "phases"} selected.</span>
            ) : null}
          </p>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="ghost" size="sm" onClick={resetAndClose} disabled={submitting}>
              Close
            </Button>
            <Button
              type="button"
              size="sm"
              onClick={submit}
              disabled={!canSubmit || submitting}
              data-testid="create-work-from-plan-submit"
            >
              {submitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden />}
              Create work
            </Button>
          </div>
        </div>
      }
    >
      <div className="grid min-h-0 gap-0 md:grid-cols-[1.2fr_0.8fr]">
        <section className="space-y-3 border-b border-white/10 p-4 md:border-b-0 md:border-r">
          <SegmentedSource value={sourceMode} onChange={setSourceMode} />
          {sourceMode === "existing" ? (
            <div className="space-y-3">
              <Input
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="Search title, slug, or id"
                leftIcon={<Search className="h-4 w-4" aria-hidden />}
                size="sm"
                data-testid="create-work-plan-search"
              />
              <div className="max-h-80 space-y-1 overflow-y-auto pr-1" data-testid="create-work-plan-list">
                {loadingPlans ? (
                  <div className="flex items-center gap-2 rounded border border-slate-800 px-3 py-2 text-sm text-slate-400">
                    <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
                    Loading plans
                  </div>
                ) : filteredPlans.length === 0 ? (
                  <div className="rounded border border-dashed border-slate-800 px-3 py-6 text-center text-sm text-slate-500">
                    No plans match this search.
                  </div>
                ) : filteredPlans.map((plan) => (
                  <button
                    key={plan.id}
                    type="button"
                    onClick={() => setSelectedPlanId(plan.id)}
                    className={cn(
                      "w-full rounded-lg border px-3 py-2 text-left transition-colors",
                      selectedPlanId === plan.id
                        ? "border-cyan-500/70 bg-cyan-500/10"
                        : "border-slate-800 bg-slate-950/60 hover:border-slate-700 hover:bg-slate-900",
                    )}
                    data-testid={`create-work-plan-option-${plan.slug}`}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="min-w-0 truncate text-sm font-medium text-slate-100">
                        {plan.title || plan.slug}
                      </span>
                      <span className="shrink-0 text-xs text-slate-500">{plan.phaseCount} phases</span>
                    </div>
                    <div className="mt-0.5 flex items-center gap-1.5 text-xs text-slate-500">
                      <Link2 className="h-3.5 w-3.5" aria-hidden />
                      <span className="truncate">{plan.slug}</span>
                    </div>
                  </button>
                ))}
              </div>
            </div>
          ) : (
            <div className="space-y-3">
              <Input
                value={sourcePath}
                onChange={(event) => setSourcePath(event.target.value)}
                placeholder="/absolute/path/to/external-plan.markdown"
                size="sm"
                data-testid="create-work-source-path"
              />
              <textarea
                value={markdown}
                onChange={(event) => setMarkdown(event.target.value)}
                placeholder="# Plan markdown"
                className="min-h-36 w-full resize-y rounded-lg border border-white/10 bg-slate-800/50 px-3 py-2 text-base text-slate-50 placeholder:text-slate-400 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500 md:text-sm"
                data-testid="create-work-markdown"
              />
              <div className="grid gap-2 sm:grid-cols-2">
                <Input value={title} onChange={(event) => setTitle(event.target.value)} placeholder="Plan title" size="sm" />
                <Input value={slug} onChange={(event) => setSlug(event.target.value)} placeholder="stable-slug" size="sm" />
              </div>
            </div>
          )}
        </section>
        <section className="space-y-4 p-4">
          <div className="space-y-2">
            <span className="text-xs font-medium uppercase text-slate-500">Container</span>
            <div className="grid gap-2">
              <ContainerOption
                selected={containerType === "items"}
                title="Backlog items"
                hint="Create or link one execute item per plan phase."
                onClick={() => setContainerType("items")}
              />
              <ContainerOption
                selected={containerType === "initiative"}
                title="Initiative"
                hint="Create or update an initiative and attach the phase items."
                onClick={() => setContainerType("initiative")}
              />
            </div>
          </div>
          {containerType === "initiative" && (
            <div className="space-y-3">
              <div className="grid gap-2 sm:grid-cols-2">
                <Input value={initiativeName} onChange={(event) => setInitiativeName(event.target.value)} placeholder="initiative-name" size="sm" />
                <Input value={initiativeTitle} onChange={(event) => setInitiativeTitle(event.target.value)} placeholder="Initiative title" size="sm" />
              </div>
              <textarea
                value={initiativeDescription}
                onChange={(event) => setInitiativeDescription(event.target.value)}
                placeholder="Initiative description"
                className="min-h-20 w-full resize-y rounded-lg border border-white/10 bg-slate-800/50 px-3 py-2 text-base text-slate-50 placeholder:text-slate-400 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500 md:text-sm"
              />
              <div className="space-y-2">
                <span className="text-xs font-medium uppercase text-slate-500">Operating mode</span>
                <div className="grid gap-2">
                  {MODE_OPTIONS.map((option) => (
                    <ContainerOption
                      key={option.value}
                      selected={mode === option.value}
                      title={option.label}
                      hint={option.hint}
                      onClick={() => setMode(option.value)}
                    />
                  ))}
                </div>
              </div>
            </div>
          )}
          {result && <ImportResultLinks result={result} />}
        </section>
      </div>
    </BottomSheet>
  );
}

function SegmentedSource({ value, onChange }: { value: SourceMode; onChange: (value: SourceMode) => void }) {
  return (
    <div className="grid grid-cols-2 rounded-lg border border-slate-800 bg-slate-950 p-1">
      {([
        ["existing", "Existing plan", Link2],
        ["markdown", "Adopt markdown", FilePlus2],
      ] as const).map(([mode, label, Icon]) => (
        <button
          key={mode}
          type="button"
          onClick={() => onChange(mode)}
          className={cn(
            "inline-flex h-8 items-center justify-center gap-1.5 rounded-md text-sm transition-colors",
            value === mode ? "bg-slate-800 text-slate-100" : "text-slate-400 hover:text-slate-200",
          )}
        >
          <Icon className="h-4 w-4" aria-hidden />
          {label}
        </button>
      ))}
    </div>
  );
}

function ContainerOption({
  selected,
  title,
  hint,
  onClick,
}: {
  selected: boolean;
  title: string;
  hint: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "rounded-lg border px-3 py-2 text-left transition-colors",
        selected ? "border-cyan-500/70 bg-cyan-500/10" : "border-slate-800 bg-slate-950/60 hover:border-slate-700",
      )}
    >
      <div className="text-sm font-medium text-slate-100">{title}</div>
      <div className="mt-0.5 text-xs text-slate-500">{hint}</div>
    </button>
  );
}

function ImportResultLinks({ result }: { result: PlanImportResult }) {
  return (
    <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-3 text-sm" data-testid="create-work-result-links">
      <div className="font-medium text-emerald-200">Linked to {result.slug}</div>
      <div className="mt-2 flex flex-wrap gap-2">
        {result.initiative && (
          <Link className="text-cyan-300 hover:text-cyan-200" to={initiativeDetailPath(result.initiative.name)}>
            Initiative: {result.initiative.title || result.initiative.name}
          </Link>
        )}
        {result.items.slice(0, 5).map((item) => (
          <Link key={`${item.kind}/${item.name}`} className="text-cyan-300 hover:text-cyan-200" to={backlogDetailPath(item.kind, item.name)}>
            {item.kind}/{item.name}
          </Link>
        ))}
        {result.items.length > 5 && <span className="text-slate-400">+{result.items.length - 5} more</span>}
      </div>
    </div>
  );
}
