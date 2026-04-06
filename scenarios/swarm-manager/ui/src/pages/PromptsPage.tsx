import { useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import { useResizablePanel } from "../hooks/useResizablePanel";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { BottomSheet } from "../components/ui/bottom-sheet";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { SkillsPanel } from "../components/prompts/SkillsPanel";
import { PromptCatalog } from "../components/prompts/PromptCatalog";
import { PromptEditor } from "../components/prompts/PromptEditor";
import { ExperimentResults } from "../components/prompts/ExperimentResults";
import { SimulationDialog, defaultSimulationPayload, type SimulationPayload } from "../components/prompts/SimulationDialog";
import { selectors } from "../consts/selectors";
import { defaultQueryOptions } from "../lib";
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

type PromptGroup = (typeof GROUP_ORDER)[number];
type PromptTab = "catalog" | "viewer" | "experiments";

const MIN_SKILLS_PANEL_WIDTH = 260;
const MAX_SKILLS_PANEL_WIDTH = 460;
const MIN_EDITOR_WIDTH = 480;
const RESIZE_HANDLE_WIDTH = 8;

const splitLines = (value: string) => value.replace(/\r\n/g, "\n").split("\n");

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
  const { size: skillsPanelWidth, isResizing, resizeHandleProps: skillsResizeHandleProps } = useResizablePanel({
    containerRef: workspaceRef,
    minSize: MIN_SKILLS_PANEL_WIDTH,
    maxSize: MAX_SKILLS_PANEL_WIDTH,
    defaultSize: 320,
    adjacentMinSize: MIN_EDITOR_WIDTH,
    handleWidth: RESIZE_HANDLE_WIDTH,
  });
  const [selectedExperimentId, setSelectedExperimentId] = useState<string>("");
  const [showSimulationModal, setShowSimulationModal] = useState(false);
  const [showMobileSkills, setShowMobileSkills] = useState(false);
  const [simulationPayload, setSimulationPayload] = useState<SimulationPayload>(defaultSimulationPayload());
  const [lastSimulationPayload, setLastSimulationPayload] = useState<SimulationPayload | null>(null);

  // --- Queries ---
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

  // --- Mutations ---
  const updateMutation = useMutation({
    mutationFn: ({ draft, nextContent }: { draft: boolean; nextContent: string }) =>
      promptService.updateSkill(selectedSkillId, { content: nextContent, draft }),
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

  // --- Derived data ---
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

  const experimentIds = useMemo(() => {
    const ids = new Set<string>();
    for (const entry of catalogQuery.data ?? []) {
      if (entry.experiment_id) ids.add(entry.experiment_id);
    }
    return Array.from(ids);
  }, [catalogQuery.data]);

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

  // --- Effects ---
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
      if (nextMode === prev.mode && nextKind === prev.kind) return prev;
      return { ...prev, mode: nextMode, kind: nextKind };
    });
  }, [canSimulateSelectedSkill, selectedSkillCatalogEntries]);

  // --- Handlers ---
  const openInViewer = (skillID?: string) => {
    if (!skillID) return;
    setSelectedSkillId(skillID);
    setActiveTab("viewer");
  };

  const runSimulation = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const payload: SimulationPayload = { ...simulationPayload };
    setLastSimulationPayload(payload);
    simulateMutation.mutate(payload);
  };

  // --- Loading / error states ---
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

  // --- Render ---
  return (
    <div className="space-y-6" data-testid={selectors.prompts.page}>
      <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as PromptTab)}>
        <TabsList data-testid={selectors.prompts.tabs}>
          <TabsTrigger value="catalog" data-testid={selectors.prompts.tabMap}>Prompt Catalog</TabsTrigger>
          <TabsTrigger value="viewer" data-testid={selectors.prompts.tabViewer}>Skills Viewer</TabsTrigger>
          {experimentIds.length > 0 && (
            <TabsTrigger value="experiments">Experiments</TabsTrigger>
          )}
        </TabsList>

        <TabsContent value="catalog" data-testid={selectors.prompts.mapPanel}>
          <PromptCatalog
            catalogData={catalogQuery.data ?? []}
            groupEntries={groupEntries}
            onOpenInViewer={openInViewer}
          />
        </TabsContent>

        <TabsContent value="viewer" data-testid={selectors.prompts.viewerPanel}>
          <PromptEditor
            workspaceRef={workspaceRef}
            skillsPanelWidth={skillsPanelWidth}
            isResizing={isResizing}
            skillsResizeHandleProps={skillsResizeHandleProps}
            skillsSidebar={
              <SkillsPanel
                skills={skillsQuery.data ?? []}
                selectedSkillId={selectedSkillId}
                onSelectSkill={setSelectedSkillId}
              />
            }
            selectedSkill={selectedSkill}
            skillLoading={skillQuery.isLoading}
            content={content}
            onContentChange={setContent}
            markdownView={markdownView}
            onToggleMarkdownView={() => setMarkdownView((prev) => (prev === "rendered" ? "raw" : "rendered"))}
            markdownPreviewSource={markdownPreviewSource}
            lastSimulationPayload={lastSimulationPayload}
            onSaveDraft={() => updateMutation.mutate({ draft: true, nextContent: content })}
            onPublish={() => updateMutation.mutate({ draft: false, nextContent: content })}
            updatePending={updateMutation.isPending}
            canSimulate={canSimulateSelectedSkill}
            onOpenSimulation={() => setShowSimulationModal(true)}
            hasSimulationResult={Boolean(simulateMutation.data)}
            onClearSimulation={() => {
              simulateMutation.reset();
              setLastSimulationPayload(null);
            }}
            onShowMobileSkills={() => setShowMobileSkills(true)}
            versions={versionsQuery.data?.versions ?? []}
            comparisonVersion={comparisonVersion}
            onCompare={setComparisonVersion}
            onRollback={(version) => revertMutation.mutate(version)}
            revertPending={revertMutation.isPending}
            diffLines={diffLines}
          />
        </TabsContent>
        {experimentIds.length > 0 && (
          <TabsContent value="experiments">
            <div className="space-y-4">
              {experimentIds.length > 1 && (
                <div className="flex gap-2 flex-wrap">
                  {experimentIds.map((eid) => (
                    <button
                      key={eid}
                      type="button"
                      onClick={() => setSelectedExperimentId(eid)}
                      className={`rounded-md px-3 py-1.5 text-sm border transition-colors ${
                        selectedExperimentId === eid
                          ? "bg-primary text-primary-foreground border-primary"
                          : "bg-muted border-border hover:bg-accent"
                      }`}
                    >
                      {eid}
                    </button>
                  ))}
                </div>
              )}
              <ExperimentResults
                experimentId={selectedExperimentId || experimentIds[0] || ""}
              />
            </div>
          </TabsContent>
        )}
      </Tabs>

      <SimulationDialog
        isOpen={showSimulationModal}
        onClose={() => setShowSimulationModal(false)}
        isPending={simulateMutation.isPending}
        selectedSkillId={selectedSkillId}
        payload={simulationPayload}
        onPayloadChange={setSimulationPayload}
        kindOptions={simulationKindOptions}
        modeOptions={simulationModeOptions}
        onSubmit={runSimulation}
      />

      <BottomSheet
        isOpen={showMobileSkills}
        onClose={() => setShowMobileSkills(false)}
        title="Prompt Skills"
        className="lg:hidden"
      >
        <SkillsPanel
          skills={skillsQuery.data ?? []}
          selectedSkillId={selectedSkillId}
          onSelectSkill={setSelectedSkillId}
          onClose={() => setShowMobileSkills(false)}
        />
      </BottomSheet>
    </div>
  );
}
