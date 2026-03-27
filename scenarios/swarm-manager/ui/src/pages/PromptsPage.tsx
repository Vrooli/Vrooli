import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent, type KeyboardEvent, type PointerEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Editor from "@monaco-editor/react";
import { Code, Eye, List } from "lucide-react";
import { renderMarkdown } from "../lib/render-markdown";
import { Card } from "../components/ui/card";
import { BottomSheet } from "../components/ui/bottom-sheet";
import { Button } from "../components/ui/button";
import { Dialog } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Select } from "../components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { ErrorState } from "../components/ui/error-state";
import { InlineLoadingIndicator, PageLoadingState } from "../components/ui/loading-states";
import { selectors } from "../consts/selectors";
import { defaultQueryOptions, formatRelativeTime } from "../lib";
import { promptService } from "../services";
import type { BacklogKind, PromptCatalogEntry, PromptSkillVersion } from "../types";

const KINDS: BacklogKind[] = ["idea", "research", "fix", "execute", "chore"];
const MODES: string[] = ["workshop", "initialize", "finalize"];
const GROUP_ORDER = ["capture", "backlog", "execution", "archive", "support"] as const;
const GROUP_LABELS: Record<(typeof GROUP_ORDER)[number], string> = {
  capture: "Capture",
  backlog: "Backlog",
  execution: "Execution",
  archive: "Archive",
  support: "Support",
};

const MIN_SKILLS_PANEL_WIDTH = 260;
const MAX_SKILLS_PANEL_WIDTH = 460;
const MIN_EDITOR_WIDTH = 480;
const RESIZE_HANDLE_WIDTH = 8;

const EDITOR_OPTIONS = {
  minimap: { enabled: false },
  wordWrap: "on",
  lineNumbers: "on",
  fontSize: 13,
  fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
  tabSize: 2,
  scrollBeyondLastLine: false,
  padding: { top: 12, bottom: 12 },
  renderLineHighlight: "line",
  cursorBlinking: "smooth",
  smoothScrolling: true,
  scrollbar: {
    vertical: "auto",
    horizontal: "auto",
    verticalScrollbarSize: 8,
    horizontalScrollbarSize: 8,
  },
  overviewRulerBorder: false,
  hideCursorInOverviewRuler: true,
  folding: true,
  foldingStrategy: "indentation",
  automaticLayout: true,
} as const;

type PromptGroup = (typeof GROUP_ORDER)[number];
type PromptTab = "catalog" | "viewer";

type SimulationPayload = {
  kind: BacklogKind;
  mode: string;
  item_name: string;
  item_title: string;
  item_description: string;
  item_status: string;
  item_priority: string;
  item_tags: string;
  item_folder: string;
};

const defaultSimulationPayload = (): SimulationPayload => ({
  kind: "idea",
  mode: "workshop",
  item_name: "sample-item",
  item_title: "Sample Item",
  item_description: "Sample description for simulation preview.",
  item_status: "backlog",
  item_priority: "3",
  item_tags: "sample, prompt-center",
  item_folder: "scenarios/swarm-manager/ideas/sample-item",
});

const splitLines = (value: string) => value.replace(/\r\n/g, "\n").split("\n");
const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));
const joinParts = (parts?: string[]) => (parts && parts.length > 0 ? parts.join(", ") : "-");
const formatUsageLabel = (value: string) => value.replace(/_/g, " ");

const buildSimpleDiff = (next: string, previous: string): string[] => {
  const nextLines = splitLines(next);
  const previousLines = splitLines(previous);
  const max = Math.max(nextLines.length, previousLines.length);
  const output: string[] = [];
  for (let i = 0; i < max; i++) {
    const before = previousLines[i] ?? "";
    const after = nextLines[i] ?? "";
    if (before === after) {
      output.push(`  ${after}`);
      continue;
    }
    if (before !== "") output.push(`- ${before}`);
    if (after !== "") output.push(`+ ${after}`);
  }
  return output;
};

export function PromptsPage() {
  const queryClient = useQueryClient();
  const workspaceRef = useRef<HTMLDivElement | null>(null);

  const [activeTab, setActiveTab] = useState<PromptTab>("catalog");
  const [selectedSkillId, setSelectedSkillId] = useState<string>("");
  const [content, setContent] = useState("");
  const [comparisonVersion, setComparisonVersion] = useState<PromptSkillVersion | null>(null);
  const [markdownView, setMarkdownView] = useState<"raw" | "rendered">("raw");
  const [skillsPanelWidth, setSkillsPanelWidth] = useState(320);
  const [isResizing, setIsResizing] = useState(false);
  const [showSimulationModal, setShowSimulationModal] = useState(false);
  const [showMobileSkills, setShowMobileSkills] = useState(false);
  const [simulationPayload, setSimulationPayload] = useState<SimulationPayload>(defaultSimulationPayload());
  const [lastSimulationPayload, setLastSimulationPayload] = useState<SimulationPayload | null>(null);

  const catalogQuery = useQuery({
    queryKey: ["prompts", "catalog"],
    queryFn: () => promptService.listCatalog(),
    ...defaultQueryOptions,
  });

  const skillsQuery = useQuery({
    queryKey: ["prompts", "skills"],
    queryFn: () => promptService.listSkills(),
    ...defaultQueryOptions,
  });

  useEffect(() => {
    const firstSkill = skillsQuery.data?.[0];
    if (!selectedSkillId && firstSkill) {
      setSelectedSkillId(firstSkill.id);
    }
  }, [selectedSkillId, skillsQuery.data]);

  const skillQuery = useQuery({
    queryKey: ["prompts", "skill", selectedSkillId],
    queryFn: () => promptService.getSkill(selectedSkillId),
    enabled: selectedSkillId.length > 0,
    ...defaultQueryOptions,
  });

  const versionsQuery = useQuery({
    queryKey: ["prompts", "versions", selectedSkillId],
    queryFn: () => promptService.getSkillVersions(selectedSkillId),
    enabled: selectedSkillId.length > 0,
    ...defaultQueryOptions,
  });

  useEffect(() => {
    if (skillQuery.data?.current_content !== undefined) {
      setContent(skillQuery.data.current_content);
    }
  }, [skillQuery.data?.current_content]);

  const updateMutation = useMutation({
    mutationFn: ({
      draft,
      nextContent,
    }: {
      draft: boolean;
      nextContent: string;
    }) =>
      promptService.updateSkill(selectedSkillId, {
        content: nextContent,
        draft,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["prompts", "skills"] });
      queryClient.invalidateQueries({ queryKey: ["prompts", "skill", selectedSkillId] });
      queryClient.invalidateQueries({ queryKey: ["prompts", "versions", selectedSkillId] });
    },
  });

  const revertMutation = useMutation({
    mutationFn: (version: number) => promptService.revertSkillVersion(selectedSkillId, version),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["prompts", "skills"] });
      queryClient.invalidateQueries({ queryKey: ["prompts", "skill", selectedSkillId] });
      queryClient.invalidateQueries({ queryKey: ["prompts", "versions", selectedSkillId] });
      setComparisonVersion(null);
    },
  });

  const simulateMutation = useMutation({
    mutationFn: (payload: SimulationPayload) => promptService.simulate(payload),
    onSuccess: () => {
      setShowSimulationModal(false);
      setMarkdownView("rendered");
    },
  });

  const selectedSkill = skillQuery.data;

  const diffLines = useMemo(() => {
    if (!comparisonVersion) return [];
    return buildSimpleDiff(content, comparisonVersion.content);
  }, [comparisonVersion, content]);
  const markdownPreviewSource = simulateMutation.data?.prompt ?? content;

  const groupEntries = useMemo(() => {
    const grouped = new Map<PromptGroup, PromptCatalogEntry[]>(
      GROUP_ORDER.map((group) => [group, [] as PromptCatalogEntry[]])
    );
    for (const entry of catalogQuery.data ?? []) {
      grouped.get(entry.group)?.push(entry);
    }
    return GROUP_ORDER.map((group) => ({
      group,
      label: GROUP_LABELS[group],
      items: grouped.get(group) ?? [],
    }));
  }, [catalogQuery.data]);

  const selectedSkillCatalogEntries = useMemo(
    () =>
      (catalogQuery.data ?? []).filter(
        (entry) =>
          entry.skill_id === selectedSkillId &&
          entry.group === "backlog" &&
          entry.usage_type === "direct_runtime"
      ),
    [catalogQuery.data, selectedSkillId]
  );

  const canSimulateSelectedSkill = selectedSkillCatalogEntries.length > 0;
  const simulationKindOptions = useMemo(() => {
    if (!canSimulateSelectedSkill) return KINDS;
    const allowed = new Set<BacklogKind>();
    for (const entry of selectedSkillCatalogEntries) {
      for (const kind of entry.backlog_kinds ?? []) {
        allowed.add(kind as BacklogKind);
      }
    }
    return KINDS.filter((kind) => allowed.has(kind));
  }, [canSimulateSelectedSkill, selectedSkillCatalogEntries]);
  const simulationModeOptions = useMemo(() => {
    if (!canSimulateSelectedSkill) return MODES;
    const allowed = new Set<string>();
    for (const entry of selectedSkillCatalogEntries) {
      for (const mode of entry.modes ?? []) {
        allowed.add(mode);
      }
    }
    return MODES.filter((mode) => allowed.has(mode));
  }, [canSimulateSelectedSkill, selectedSkillCatalogEntries]);

  const openInViewer = (skillID?: string) => {
    if (!skillID) return;
    setSelectedSkillId(skillID);
    setActiveTab("viewer");
  };

  const openInViewerOnKey = (event: KeyboardEvent<HTMLElement>, skillID?: string) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      openInViewer(skillID);
    }
  };

  const handleResizeStart = useCallback((event: PointerEvent<HTMLDivElement>) => {
    if (event.button !== 0) return;
    event.preventDefault();
    setIsResizing(true);
  }, []);

  useEffect(() => {
    if (!isResizing) return;

    const handlePointerMove = (event: globalThis.PointerEvent) => {
      if (!workspaceRef.current) return;
      const bounds = workspaceRef.current.getBoundingClientRect();
      const maxWidth = Math.max(
        MIN_SKILLS_PANEL_WIDTH,
        Math.min(
          MAX_SKILLS_PANEL_WIDTH,
          bounds.width - MIN_EDITOR_WIDTH - RESIZE_HANDLE_WIDTH
        )
      );
      const nextWidth = clamp(event.clientX - bounds.left, MIN_SKILLS_PANEL_WIDTH, maxWidth);
      setSkillsPanelWidth(nextWidth);
    };

    const handlePointerUp = () => {
      setIsResizing(false);
    };

    const previousCursor = document.body.style.cursor;
    const previousUserSelect = document.body.style.userSelect;
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
    window.addEventListener("pointermove", handlePointerMove);
    window.addEventListener("pointerup", handlePointerUp);

    return () => {
      document.body.style.cursor = previousCursor;
      document.body.style.userSelect = previousUserSelect;
      window.removeEventListener("pointermove", handlePointerMove);
      window.removeEventListener("pointerup", handlePointerUp);
    };
  }, [isResizing]);

  useEffect(() => {
    if (!canSimulateSelectedSkill) return;
    const [entry] = selectedSkillCatalogEntries;
    if (!entry) return;
    setSimulationPayload((prev) => {
      const nextMode = entry.modes?.[0] ?? prev.mode;
      const allowedKinds = entry.backlog_kinds ?? [];
      const nextKind = allowedKinds.includes(prev.kind)
        ? prev.kind
        : ((allowedKinds[0] as BacklogKind | undefined) ?? prev.kind);
      if (nextMode === prev.mode && nextKind === prev.kind) {
        return prev;
      }
      return {
        ...prev,
        mode: nextMode,
        kind: nextKind,
      };
    });
  }, [canSimulateSelectedSkill, selectedSkillCatalogEntries]);

  const runSimulation = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const payload: SimulationPayload = {
      ...simulationPayload,
    };
    setLastSimulationPayload(payload);
    simulateMutation.mutate(payload);
  };

  if (catalogQuery.isLoading || skillsQuery.isLoading) {
    return <PageLoadingState variant="settings" label="Loading prompt center..." />;
  }

  if (catalogQuery.error || skillsQuery.error) {
    const err = catalogQuery.error ?? skillsQuery.error;
    return (
      <ErrorState
        title="Unable to load prompts"
        message={err instanceof Error ? err.message : "Prompt center failed to load."}
        onRetry={() => {
          void catalogQuery.refetch();
          void skillsQuery.refetch();
        }}
      />
    );
  }

  const skillsList = (
    <div className="h-full space-y-3 p-4" data-testid={selectors.prompts.skillsList}>
      <h3 className="text-base font-semibold text-slate-100">Swarm Prompt Skills</h3>
      <div className="max-h-full space-y-2 overflow-auto pr-1">
        {(skillsQuery.data ?? []).map((skill) => (
          <button
            key={skill.id}
            className={`w-full rounded-md border px-3 py-2 text-left transition ${
              selectedSkillId === skill.id
                ? "border-cyan-500/60 bg-cyan-500/10"
                : "border-slate-700/60 bg-slate-900/30 hover:border-slate-500/60"
            }`}
            onClick={() => setSelectedSkillId(skill.id)}
          >
            <p className="font-mono text-xs text-cyan-300">{skill.id}</p>
            <p className="mt-1 text-sm text-slate-100">{skill.name}</p>
            <p className="mt-1 text-[11px] uppercase tracking-wide text-slate-500">
              {formatUsageLabel(skill.usage_type)} • {joinParts(skill.groups)}
            </p>
            <p className="mt-1 text-xs text-slate-400">{skill.impact_summary}</p>
          </button>
        ))}
      </div>
    </div>
  );

  return (
    <div className="space-y-6" data-testid={selectors.prompts.page}>
      <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as PromptTab)}>
        <TabsList data-testid={selectors.prompts.tabs}>
          <TabsTrigger value="catalog" data-testid={selectors.prompts.tabMap}>Prompt Catalog</TabsTrigger>
          <TabsTrigger value="viewer" data-testid={selectors.prompts.tabViewer}>Skills Viewer</TabsTrigger>
        </TabsList>

        <TabsContent value="catalog" data-testid={selectors.prompts.mapPanel}>
          <div className="space-y-6">
            <Card className="space-y-3 p-4" data-testid={selectors.prompts.usageMatrix}>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <h2 className="text-lg font-semibold text-slate-100">Prompt Inventory</h2>
                <p className="text-xs text-slate-400">Single source of truth for runtime prompts, generated prompts, and support skills.</p>
              </div>
              <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                {groupEntries.map(({ group, label, items }) => (
                  <div key={group} className="rounded-md border border-slate-700/60 bg-slate-900/30 p-3">
                    <div className="mb-2 flex items-center justify-between">
                      <h3 className="text-sm font-semibold text-slate-100">{label}</h3>
                      <span className="rounded-full border border-slate-600/60 px-2 py-0.5 text-[10px] text-slate-300">
                        {items.length}
                      </span>
                    </div>
                    <div className="space-y-2">
                      {items.length > 0 ? (
                        items.map((entry) => {
                          if (entry.skill_id) {
                            return (
                              <button
                                key={entry.id}
                                type="button"
                                className="w-full rounded border border-slate-700/50 px-2 py-1.5 text-left transition hover:border-cyan-500/50 hover:bg-cyan-500/5"
                                onClick={() => openInViewer(entry.skill_id)}
                                onKeyDown={(event) => openInViewerOnKey(event, entry.skill_id)}
                              >
                                <p className="text-[11px] text-slate-300">{entry.title}</p>
                                <div className="mt-1 flex flex-wrap items-center gap-2 text-[10px] text-slate-400">
                                  <span>{formatUsageLabel(entry.usage_type)}</span>
                                  <span>{joinParts(entry.modes ?? entry.operations)}</span>
                                  <span className="font-mono text-cyan-300">{entry.skill_id}</span>
                                </div>
                              </button>
                            );
                          }
                          return (
                            <div key={entry.id} className="rounded border border-slate-700/50 px-2 py-1.5">
                              <p className="text-[11px] text-slate-300">{entry.title}</p>
                              <div className="mt-1 flex flex-wrap items-center gap-2 text-[10px] text-slate-400">
                                <span>{formatUsageLabel(entry.usage_type)}</span>
                                <span>{joinParts(entry.operations)}</span>
                                <span className="font-mono text-cyan-300">{entry.builder ?? "generated"}</span>
                              </div>
                            </div>
                          );
                        })
                      ) : (
                        <p className="text-xs text-slate-500">No catalog entries</p>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </Card>

            <Card className="space-y-3 p-4" data-testid={selectors.prompts.bindingMap}>
              <h2 className="text-lg font-semibold text-slate-100">Catalog Details</h2>
              <p className="text-sm text-slate-400">
                Each entry records how swarm-manager resolves the prompt, when it runs, and which artifacts it affects.
              </p>
              <div className="overflow-x-auto">
                <table className="w-full min-w-[1080px] text-left text-sm">
                  <thead className="text-slate-400">
                    <tr>
                      <th className="px-2 py-2">Group</th>
                      <th className="px-2 py-2">Usage</th>
                      <th className="px-2 py-2">Trigger</th>
                      <th className="px-2 py-2">Kinds / Modes</th>
                      <th className="px-2 py-2">Runtime Prompt</th>
                      <th className="px-2 py-2">Purpose</th>
                      <th className="px-2 py-2">Outputs</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(catalogQuery.data ?? []).map((entry) => (
                      entry.skill_id ? (
                        <tr
                          key={entry.id}
                          className="cursor-pointer border-t border-slate-700/60 text-slate-200 transition hover:bg-cyan-500/5"
                          role="button"
                          tabIndex={0}
                          onClick={() => openInViewer(entry.skill_id)}
                          onKeyDown={(event) => openInViewerOnKey(event, entry.skill_id)}
                        >
                          <td className="px-2 py-2 uppercase text-xs text-slate-400">{entry.group}</td>
                          <td className="px-2 py-2 text-slate-300">{formatUsageLabel(entry.usage_type)}</td>
                          <td className="px-2 py-2">
                            <p className="text-slate-100">{entry.title}</p>
                            <p className="text-xs text-slate-400">{entry.trigger}</p>
                          </td>
                          <td className="px-2 py-2 text-slate-300">
                            {joinParts(entry.backlog_kinds)} / {joinParts(entry.modes ?? entry.operations)}
                          </td>
                          <td className="px-2 py-2">
                            <span className="font-mono text-cyan-300">{entry.skill_id}</span>
                          </td>
                          <td className="px-2 py-2 text-slate-300">{entry.purpose}</td>
                          <td className="px-2 py-2 text-xs text-slate-400">{joinParts(entry.output_paths)}</td>
                        </tr>
                      ) : (
                        <tr key={entry.id} className="border-t border-slate-700/60 text-slate-200">
                          <td className="px-2 py-2 uppercase text-xs text-slate-400">{entry.group}</td>
                          <td className="px-2 py-2 text-slate-300">{formatUsageLabel(entry.usage_type)}</td>
                          <td className="px-2 py-2">
                            <p className="text-slate-100">{entry.title}</p>
                            <p className="text-xs text-slate-400">{entry.trigger}</p>
                          </td>
                          <td className="px-2 py-2 text-slate-300">
                            {joinParts(entry.backlog_kinds)} / {joinParts(entry.modes ?? entry.operations)}
                          </td>
                          <td className="px-2 py-2">
                            <span className="font-mono text-cyan-300">{entry.builder ?? "generated"}</span>
                          </td>
                          <td className="px-2 py-2 text-slate-300">{entry.purpose}</td>
                          <td className="px-2 py-2 text-xs text-slate-400">{joinParts(entry.output_paths)}</td>
                        </tr>
                      )
                    ))}
                  </tbody>
                </table>
              </div>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="viewer" data-testid={selectors.prompts.viewerPanel}>
          <div className="h-full overflow-hidden rounded-xl border border-white/10 bg-slate-900/30">
            <div
              ref={workspaceRef}
              className={`flex h-[calc(100dvh-12rem)] flex-col lg:h-[calc(100dvh-15rem)] lg:flex-row ${isResizing ? "select-none" : ""}`}
            >
              <div className="hidden lg:flex lg:flex-col" style={{ width: skillsPanelWidth }}>
                {skillsList}
              </div>
              <div
                className="hidden lg:flex w-2 items-center justify-center border-x border-white/10 bg-slate-900/40 cursor-col-resize"
                onPointerDown={handleResizeStart}
                role="separator"
                aria-orientation="vertical"
                aria-valuenow={Math.round(skillsPanelWidth)}
                aria-valuemin={MIN_SKILLS_PANEL_WIDTH}
                aria-valuemax={MAX_SKILLS_PANEL_WIDTH}
              >
                <div className="h-10 w-1 rounded-full bg-slate-700/80" />
              </div>

              <div className="flex min-h-0 min-w-0 flex-1 flex-col" data-testid={selectors.prompts.editor}>
                {skillQuery.isLoading ? (
                  <InlineLoadingIndicator label="Loading prompt skill..." />
                ) : selectedSkill ? (
                  <>
                    <div className="flex flex-wrap items-center justify-between gap-2 border-b border-white/10 bg-slate-800/50 px-3 py-2">
                      <div>
                        <p className="font-mono text-sm text-cyan-300">{selectedSkill.id}</p>
                        <p className="text-xs text-slate-400">
                          {formatUsageLabel(selectedSkill.usage_type)} • {joinParts(selectedSkill.groups)} • {" "}
                          Updated {selectedSkill.updated_at ? formatRelativeTime(selectedSkill.updated_at) : "unknown"} • {" "}
                          {selectedSkill.impact_summary}
                        </p>
                      </div>
                      <div className="flex flex-wrap gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          className="lg:hidden"
                          onClick={() => setShowMobileSkills(true)}
                        >
                          <List className="mr-1.5 h-4 w-4" />
                          Skills
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          data-testid={selectors.prompts.saveDraft}
                          disabled={updateMutation.isPending}
                          onClick={() => updateMutation.mutate({ draft: true, nextContent: content })}
                        >
                          Save Draft
                        </Button>
                        <Button
                          size="sm"
                          data-testid={selectors.prompts.publish}
                          disabled={updateMutation.isPending}
                          onClick={() => updateMutation.mutate({ draft: false, nextContent: content })}
                        >
                          Publish
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setShowSimulationModal(true)}
                          disabled={!canSimulateSelectedSkill}
                        >
                          Simulation Preview
                        </Button>
                        <button
                          type="button"
                          className="inline-flex h-9 w-9 items-center justify-center rounded-full border border-slate-300/40 text-slate-200 transition-colors hover:bg-slate-900/20 hover:text-white"
                          onClick={() => setMarkdownView((prev) => (prev === "rendered" ? "raw" : "rendered"))}
                          aria-label={markdownView === "rendered" ? "Show raw markdown" : "Show rendered markdown"}
                          title={markdownView === "rendered" ? "Show raw markdown" : "Show rendered markdown"}
                        >
                          {markdownView === "rendered" ? <Code className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                        </button>
                        {simulateMutation.data ? (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => {
                              simulateMutation.reset();
                              setLastSimulationPayload(null);
                            }}
                          >
                            Clear Simulation
                          </Button>
                        ) : null}
                      </div>
                    </div>

                    {selectedSkill.required_missing && selectedSkill.required_missing.length > 0 ? (
                      <div className="mx-3 mt-3 rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-200">
                        Missing required variables: {selectedSkill.required_missing.join(", ")}
                      </div>
                    ) : null}

                    <div className="mt-3 flex min-h-0 flex-1 flex-col px-3 pb-3">
                      <div className="min-h-0 flex-1 overflow-hidden rounded-lg border border-slate-700/70 bg-slate-950">
                        {markdownView === "raw" ? (
                          <Editor
                            language="markdown"
                            theme="vs-dark"
                            value={content}
                            onChange={(value) => setContent(value ?? "")}
                            options={EDITOR_OPTIONS}
                            height="100%"
                            data-testid={selectors.prompts.contentInput}
                          />
                        ) : (
                          <div className="h-full overflow-auto p-4" data-testid={selectors.prompts.preview}>
                            {lastSimulationPayload ? (
                              <p className="mb-3 text-xs text-cyan-300">
                                Simulation: {lastSimulationPayload.kind}/{lastSimulationPayload.mode}
                              </p>
                            ) : null}
                            <div
                              className="prose prose-invert max-w-none prose-headings:mb-2 prose-headings:mt-4 prose-p:my-2 prose-pre:bg-slate-900 prose-code:text-cyan-300"
                              dangerouslySetInnerHTML={{ __html: renderMarkdown(markdownPreviewSource) }}
                            />
                          </div>
                        )}
                      </div>

                      <div className="mt-3 grid gap-3 xl:grid-cols-2">
                        <div className="space-y-2" data-testid={selectors.prompts.versions}>
                          <h4 className="text-sm font-semibold text-slate-100">Version History</h4>
                          <div className="max-h-52 space-y-2 overflow-auto pr-1">
                            {(versionsQuery.data?.versions ?? []).map((version) => (
                              <div key={version.version} className="rounded-md border border-slate-700/60 bg-slate-900/40 p-2">
                                <p className="text-xs text-slate-200">
                                  v{version.version} • {formatRelativeTime(version.updatedAt)}
                                </p>
                                <div className="mt-2 flex gap-2">
                                  <Button size="sm" variant="outline" onClick={() => setComparisonVersion(version)}>
                                    Compare
                                  </Button>
                                  <Button
                                    size="sm"
                                    variant="outline"
                                    disabled={revertMutation.isPending}
                                    onClick={() => revertMutation.mutate(version.version)}
                                  >
                                    Rollback
                                  </Button>
                                </div>
                              </div>
                            ))}
                          </div>
                        </div>

                        {comparisonVersion ? (
                          <div className="space-y-2">
                            <h4 className="text-sm font-semibold text-slate-100">Diff vs v{comparisonVersion.version}</h4>
                            <div className="max-h-52 overflow-auto rounded-md border border-slate-700/60 bg-slate-950 p-2">
                              <pre className="whitespace-pre-wrap break-words font-mono text-xs text-slate-200">
                                {diffLines.join("\n")}
                              </pre>
                            </div>
                          </div>
                        ) : (
                          <div className="rounded-md border border-slate-700/60 bg-slate-900/30 p-3 text-xs text-slate-400">
                            Select a version to compare against your current draft.
                          </div>
                        )}
                      </div>
                    </div>
                  </>
                ) : (
                  <p className="p-4 text-sm text-slate-400">Select a prompt skill to inspect and edit.</p>
                )}
              </div>
            </div>
          </div>
        </TabsContent>
      </Tabs>

      <Dialog
        isOpen={showSimulationModal}
        onClose={() => setShowSimulationModal(false)}
        title="Simulation Preview"
        maxWidth="max-w-3xl"
        isLoading={simulateMutation.isPending}
      >
        <p className="text-sm text-slate-400 -mt-4 mb-4">Enter context values and generate an exact prompt preview.</p>

        <form className="space-y-3" onSubmit={runSimulation}>
          <div className="grid gap-2 md:grid-cols-3">
            <Select
              value={simulationPayload.kind}
              onChange={(event) =>
                setSimulationPayload((prev) => ({ ...prev, kind: event.target.value as BacklogKind }))
              }
            >
              {simulationKindOptions.map((kind) => (
                <option key={kind} value={kind}>{kind}</option>
              ))}
            </Select>
            <Select
              value={simulationPayload.mode}
              onChange={(event) =>
                setSimulationPayload((prev) => ({ ...prev, mode: event.target.value }))
              }
            >
              {simulationModeOptions.map((mode) => (
                <option key={mode} value={mode}>{mode}</option>
              ))}
            </Select>
            <div className="rounded-md border border-slate-700/70 bg-slate-950 px-3 py-2 text-sm text-slate-400">
              {selectedSkillId ? `Skill: ${selectedSkillId}` : "Backlog prompt simulation"}
            </div>
          </div>

          <div className="grid gap-2 md:grid-cols-2">
            <Input
              value={simulationPayload.item_name}
              onChange={(event) => setSimulationPayload((prev) => ({ ...prev, item_name: event.target.value }))}
              placeholder="item_name"
            />
            <Input
              value={simulationPayload.item_title}
              onChange={(event) => setSimulationPayload((prev) => ({ ...prev, item_title: event.target.value }))}
              placeholder="item_title"
            />
            <Input
              value={simulationPayload.item_status}
              onChange={(event) => setSimulationPayload((prev) => ({ ...prev, item_status: event.target.value }))}
              placeholder="item_status"
            />
            <Input
              value={simulationPayload.item_priority}
              onChange={(event) => setSimulationPayload((prev) => ({ ...prev, item_priority: event.target.value }))}
              placeholder="item_priority"
            />
            <Input
              value={simulationPayload.item_tags}
              onChange={(event) => setSimulationPayload((prev) => ({ ...prev, item_tags: event.target.value }))}
              placeholder="item_tags"
            />
            <Input
              value={simulationPayload.item_folder}
              onChange={(event) => setSimulationPayload((prev) => ({ ...prev, item_folder: event.target.value }))}
              placeholder="item_folder"
            />
          </div>

          <textarea
            value={simulationPayload.item_description}
            onChange={(event) => setSimulationPayload((prev) => ({ ...prev, item_description: event.target.value }))}
            className="min-h-[110px] w-full rounded-md border border-slate-700/70 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none"
            placeholder="item_description"
          />

          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setShowSimulationModal(false)} disabled={simulateMutation.isPending}>
              Cancel
            </Button>
            <Button type="submit" disabled={simulateMutation.isPending}>
              {simulateMutation.isPending ? "Generating..." : "Generate Preview"}
            </Button>
          </div>
        </form>
      </Dialog>

      <BottomSheet
        isOpen={showMobileSkills}
        onClose={() => setShowMobileSkills(false)}
        title="Prompt Skills"
        className="lg:hidden"
      >
        <div className="space-y-2">
          {(skillsQuery.data ?? []).map((skill) => (
            <button
              key={skill.id}
              className={`w-full rounded-md border px-3 py-2 text-left transition ${
                selectedSkillId === skill.id
                  ? "border-cyan-500/60 bg-cyan-500/10"
                  : "border-slate-700/60 bg-slate-900/30 hover:border-slate-500/60"
              }`}
              onClick={() => {
                setSelectedSkillId(skill.id);
                setShowMobileSkills(false);
              }}
            >
              <p className="font-mono text-xs text-cyan-300">{skill.id}</p>
              <p className="mt-1 text-sm text-slate-100">{skill.name}</p>
              <p className="mt-1 text-[11px] uppercase tracking-wide text-slate-500">
                {formatUsageLabel(skill.usage_type)} • {joinParts(skill.groups)}
              </p>
              <p className="mt-1 text-xs text-slate-400">{skill.impact_summary}</p>
            </button>
          ))}
        </div>
      </BottomSheet>
    </div>
  );
}
