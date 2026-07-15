import { Fragment, type ComponentProps, type ReactNode, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Editor, { type Monaco } from "@monaco-editor/react";
import type { editor } from "monaco-editor";
import {
  closestCenter,
  DndContext,
  DragOverlay,
  KeyboardSensor,
  PointerSensor,
  type DragEndEvent,
  type DragOverEvent,
  type DragStartEvent,
  type Modifier,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import {
  arrayMove,
  horizontalListSortingStrategy,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
} from "@dnd-kit/sortable";
import { ArrowLeft, Code2, Eye, FileCode2, GripVertical, Info, Minus, Plus, RotateCcw, Save, X } from "lucide-react";
import { Group, Panel, Separator } from "react-resizable-panels";

import { Button } from "../../components/ui/button";
import { StatusBadge } from "../../components/ui/status-badge";
import { useTheme } from "../../components/theme/useTheme";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { API_BASE } from "../../api/client";
import { componentsClient, listComponentExamples, type ComponentExample } from "../../api/components";
import { useComponentInspector } from "../../hooks/useComponentInspector";
import { useDeviceEmulation } from "../../hooks/useDeviceEmulation";
import { useDeviceFilters } from "../../hooks/useDeviceFilters";
import { useMediaQuery } from "../../hooks/useMediaQuery";
import { errorMessage } from "../../lib/errorMessage";
import { EmulatorToolbar, EmulatorViewport } from "./EmulatorChrome";
import { InspectorPanel } from "./InspectorPanel";
import { ThemeSwitcher } from "./ThemeSwitcher";
import { PropsExperimentPanel } from "./PropsExperimentPanel";
import { AdoptionFileTree } from "./AdoptionFileTree";
import { ADOPTION_TEMPLATES, DEFAULT_ADOPTION_TEMPLATE } from "./adoptionTemplates";
import { VersionDiffViewer } from "../versions/VersionDiffViewer";
import { AssetWorkspace } from "../assets/AssetWorkspace";
import type { DiffRow } from "../../api/versions";

const PREVIEW_LOAD_TIMEOUT_MS = 8_000;
const PANEL_LAYOUT_STORAGE_KEY = "rcl.component-editor.workspace.v2";
const DEFAULT_DESKTOP_PANEL_LAYOUT = { preview: 42, files: 38, details: 20 };
const DEFAULT_PANE_ORDER = ["files", "preview", "details"] as const;

type WorkspacePane = (typeof DEFAULT_PANE_ORDER)[number];
type WorkspaceDropEdge = "before" | "after";
type FilesView = "tree" | "source" | "diff";
type SpecimenIdentity = `${string}:${string}`;

const restrictWorkspaceDragToHorizontalAxis: Modifier = ({ transform }) => ({
  ...transform,
  y: 0,
});

function isWorkspacePane(value: unknown): value is WorkspacePane {
  return typeof value === "string" && DEFAULT_PANE_ORDER.includes(value as WorkspacePane);
}

function reorderWorkspacePanes(
  order: readonly WorkspacePane[],
  active: WorkspacePane,
  over: WorkspacePane,
): WorkspacePane[] {
  const activeIndex = order.indexOf(active);
  const overIndex = order.indexOf(over);
  if (activeIndex < 0 || overIndex < 0 || activeIndex === overIndex) return [...order];
  return arrayMove([...order], activeIndex, overIndex);
}

function getWorkspaceDropEdge(
  order: readonly WorkspacePane[],
  active: WorkspacePane | null,
  over: WorkspacePane | null,
): WorkspaceDropEdge | null {
  if (!active || !over || active === over) return null;
  const activeIndex = order.indexOf(active);
  const overIndex = order.indexOf(over);
  if (activeIndex < 0 || overIndex < 0) return null;
  return activeIndex < overIndex ? "after" : "before";
}

type SortableHandle = Pick<
  ReturnType<typeof useSortable>,
  "attributes" | "listeners" | "setActivatorNodeRef"
>;

interface SortableWorkspacePanelProps {
  pane: WorkspacePane;
  disabled: boolean;
  dropEdge: WorkspaceDropEdge | null;
  minSize: ComponentProps<typeof Panel>["minSize"];
  defaultSize: ComponentProps<typeof Panel>["defaultSize"];
  className?: string;
  children: (handle: SortableHandle) => ReactNode;
}

function SortableWorkspacePanel({
  pane,
  disabled,
  dropEdge,
  minSize,
  defaultSize,
  className,
  children,
}: SortableWorkspacePanelProps) {
  const sortable = useSortable({ id: pane, disabled });

  return (
    <Panel
      id={pane}
      elementRef={sortable.setNodeRef}
      minSize={minSize}
      defaultSize={defaultSize}
      className={className}
    >
      <div className="relative h-full min-h-0">
        {dropEdge && (
          <div
            data-testid={selectors.components.editor.workspacePaneDropIndicator}
            data-pane={pane}
            data-edge={dropEdge}
            aria-hidden="true"
            className={`pointer-events-none absolute inset-y-1 z-40 w-1 rounded-full bg-app-primary shadow-[0_0_0_1px_var(--color-background)] ${dropEdge === "before" ? "start-0" : "end-0"}`}
          />
        )}
        {children(sortable)}
      </div>
    </Panel>
  );
}

function specimenIdentity(example?: Pick<ComponentExample, "version" | "name">): SpecimenIdentity {
  return `${example?.version || "__current__"}:${example?.name || "__default__"}`;
}

function withoutRecordKey<T>(record: Record<string, T>, key: string): Record<string, T> {
  return Object.fromEntries(Object.entries(record).filter(([candidate]) => candidate !== key));
}

export interface ComparisonSession {
  fromLabel: string;
  toLabel: string;
  rows: DiffRow[];
}

interface WorkspaceState {
  order: WorkspacePane[];
  visible: Record<WorkspacePane, boolean>;
  layout: Record<string, number>;
}

function loadWorkspaceState(renderable: boolean): WorkspaceState {
  const fallback: WorkspaceState = {
    order: [...DEFAULT_PANE_ORDER],
    visible: { files: true, preview: renderable, details: true },
    layout: DEFAULT_DESKTOP_PANEL_LAYOUT,
  };
  try {
    const raw = window.localStorage.getItem(PANEL_LAYOUT_STORAGE_KEY);
    if (!raw) return fallback;
    const saved = JSON.parse(raw) as Partial<WorkspaceState>;
    const order = Array.isArray(saved.order)
      ? saved.order.filter((pane): pane is WorkspacePane => DEFAULT_PANE_ORDER.includes(pane))
      : fallback.order;
    const uniqueOrder = [...new Set(order)];
    for (const pane of DEFAULT_PANE_ORDER) if (!uniqueOrder.includes(pane)) uniqueOrder.push(pane);
    const visible = DEFAULT_PANE_ORDER.reduce<Record<WorkspacePane, boolean>>((next, pane) => {
      next[pane] = saved.visible?.[pane] ?? true;
      return next;
    }, {} as Record<WorkspacePane, boolean>);
    if (!renderable) visible.preview = false;
    if (!Object.values(visible).some(Boolean)) visible.files = true;
    const layout = { ...fallback.layout, ...(saved.layout ?? {}) };
    return { order: uniqueOrder, visible, layout };
  } catch {
    return fallback;
  }
}

// JSDOM does not provide ResizeObserver, while the browser-only panel library
// requires one at mount time. The fallback is intentionally inert: production
// browsers retain the native observer and panel sizing; unit tests only need a
// stable layout tree.
if (typeof ResizeObserver === "undefined") {
  class NoopResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  globalThis.ResizeObserver = NoopResizeObserver;
}

/**
 * Build the harness URL the iframe loads. The query param `v` is the
 * latest content sha256 — when the sha changes (i.e. after a save),
 * React's `src` diff forces the iframe to reload. The harness route is
 * served by the API at the same origin as the Connect transport.
 */
function harnessUrl(
  id: string,
  contentVersion: string,
  reloadKey = 0,
  example?: ComponentExample,
  selectedVersion?: string,
): string {
  const base = API_BASE.replace(/\/$/, "");
  const v = encodeURIComponent(contentVersion || "initial");
  const url = new URL(`${base}/preview/${encodeURIComponent(id)}/harness.html`);
  url.searchParams.set("v", v);
  url.searchParams.set("r", String(reloadKey));
  if (selectedVersion) url.searchParams.set("version", selectedVersion);
  if (example) {
    url.searchParams.set("example", example.name);
    url.searchParams.set("version", selectedVersion || example.version);
  }
  return url.toString();
}

interface ComponentEditorProps {
  id: string;
  libraryId: string;
  onClose: () => void;
  metadataSlot?: ReactNode;
  selectedVersion?: string;
  comparison?: ComparisonSession | null;
  onCloseComparison?: () => void;
  /** Hooks share the source/details workspace but intentionally omit preview. */
  renderable?: boolean;
}

/**
 * ComponentEditor opens a Monaco TSX editor over a component's source
 * file. Mounts as a full-card surface; the user returns to the list
 * via Back-to-list. Save POSTs the buffer through
 * `componentsClient.updateComponentContent` with the optimistic-
 * concurrency `expectedSha256` taken from the most recent server
 * fetch — the server rejects with FailedPrecondition when the on-disk
 * file moved underneath us, so the user sees a typed error instead of
 * silently overwriting drift.
 */
export function ComponentEditor({
  id,
  libraryId,
  onClose,
  metadataSlot,
  selectedVersion,
  comparison,
  onCloseComparison,
  renderable = true,
}: ComponentEditorProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const emulator = useDeviceEmulation();
  const filters = useDeviceFilters();
  const desktopLayout = useMediaQuery("(min-width: 1024px)");
  const workspaceDragSensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );
  const { resolved: appResolvedTheme } = useTheme();
  const previewFrameRef = useRef<HTMLIFrameElement | null>(null);
  const previewFramesRef = useRef(new Set<HTMLIFrameElement>());
  const specimenFramesRef = useRef(new Map<SpecimenIdentity, HTMLIFrameElement>());
  const inspector = useComponentInspector(previewFrameRef);
  const [selectedFile, setSelectedFile] = useState("");
  const [selectedTemplate, setSelectedTemplate] = useState(DEFAULT_ADOPTION_TEMPLATE);
  const versionsQuery = useQuery({
    queryKey: ["components", "versions", id],
    queryFn: () => componentsClient.listComponentVersions({ componentId: id, limit: 100 }),
  });
  const activeVersion = selectedVersion || versionsQuery.data?.versions[0]?.version || "";
  const activeVersionFiles = ((versionsQuery.data?.versions ?? []).find((version) => version.version === activeVersion)?.files ?? []) as Array<{ path: string; isEntry: boolean }>;

  const contentQuery = useQuery({
    queryKey: ["components", "content", id, selectedVersion ?? "current", selectedFile],
    queryFn: async (): Promise<{ content: string; sha256: string }> => {
      if (!selectedVersion) {
        const current = await componentsClient.getComponentContent({ id, ...(selectedFile ? { path: selectedFile } : {}) });
        return { content: current.content, sha256: current.sha256 };
      }
      const historical = await componentsClient.getComponentVersionContent({
        componentId: id,
        version: selectedVersion,
        ...(selectedFile ? { path: selectedFile } : {}),
      });
      return {
        content: historical.content,
        sha256: historical.version?.contentSha256 ?? "",
      };
    },
  });

  const examplesQuery = useQuery({
    queryKey: ["components", "examples", id, selectedVersion ?? "current"],
    queryFn: () => listComponentExamples({ componentId: id, version: selectedVersion, limit: 200 }),
  });

  const [buffer, setBuffer] = useState<string>("");
  const [baselineSha, setBaselineSha] = useState<string>("");
  const [showSaved, setShowSaved] = useState(false);
  const [previewState, setPreviewState] = useState<"waiting" | "ready" | "error">("waiting");
  const [previewMessage, setPreviewMessage] = useState("");
  const [readyExamples, setReadyExamples] = useState<ReadonlySet<string>>(() => new Set());
  const [specimenErrors, setSpecimenErrors] = useState<Record<string, string>>({});
  const [specimenRetries, setSpecimenRetries] = useState<Record<string, number>>({});
  const [comparedSpecimens, setComparedSpecimens] = useState<ReadonlySet<SpecimenIdentity>>(() => new Set());
  const [activeSpecimen, setActiveSpecimen] = useState<SpecimenIdentity | null>(null);
  const [specimenOverrides, setSpecimenOverrides] = useState<Record<string, Record<string, unknown>>>({});
  const [overrideStatus, setOverrideStatus] = useState<Record<string, "idle" | "applying" | "applied" | "error">>({});
  const [overrideMessages, setOverrideMessages] = useState<Record<string, string>>({});
  const previewReloadKey = 0;
  const [mode, setMode] = useState<WorkspacePane>("files");
  const [workspace, setWorkspace] = useState<WorkspaceState>(() => loadWorkspaceState(renderable));
  const [activeDraggedPane, setActiveDraggedPane] = useState<WorkspacePane | null>(null);
  const [draggedOverPane, setDraggedOverPane] = useState<WorkspacePane | null>(null);
  const [filesView, setFilesView] = useState<FilesView>("source");
  const [wordWrap, setWordWrap] = useState<"on" | "off">("on");
  const [fontSize, setFontSize] = useState(13);
  const savedToastTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const previewTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const previewReady = previewState === "ready";
  const resolvedPreviewTheme = filters.colorScheme === "system"
    ? appResolvedTheme
    : filters.colorScheme;

  const postToPreviewFrames = useCallback((message: unknown) => {
    for (const frame of previewFramesRef.current) {
      frame.contentWindow?.postMessage(message, "*");
    }
  }, []);

  const postToSpecimen = useCallback((identity: SpecimenIdentity, message: unknown) => {
    specimenFramesRef.current.get(identity)?.contentWindow?.postMessage(message, "*");
  }, []);

  const registerPreviewFrame = useCallback((identity: SpecimenIdentity, frame: HTMLIFrameElement | null) => {
    const previous = specimenFramesRef.current.get(identity);
    if (previous) previewFramesRef.current.delete(previous);
    if (!frame) {
      specimenFramesRef.current.delete(identity);
      if (previewFrameRef.current === previous) previewFrameRef.current = null;
      return;
    }
    specimenFramesRef.current.set(identity, frame);
    previewFramesRef.current.add(frame);
    if (!previewFrameRef.current) previewFrameRef.current = frame;
  }, []);

  const activateSpecimen = useCallback((identity: SpecimenIdentity) => {
    setActiveSpecimen(identity);
    previewFrameRef.current = specimenFramesRef.current.get(identity) ?? null;
  }, []);

  const retrySpecimen = useCallback((identity: SpecimenIdentity) => {
    setSpecimenErrors((current) => {
      return withoutRecordKey(current, identity);
    });
    setSpecimenRetries((current) => ({ ...current, [identity]: (current[identity] ?? 0) + 1 }));
  }, []);

  const toggleComparison = useCallback((identity: SpecimenIdentity) => {
    setComparedSpecimens((current) => {
      const next = new Set(current);
      if (next.has(identity)) next.delete(identity);
      else if (next.size < 2) next.add(identity);
      return next;
    });
    activateSpecimen(identity);
  }, [activateSpecimen]);

  useEffect(() => {
    const handler = (ev: MessageEvent) => {
      const data = ev.data as { type?: string; id?: string; message?: string; example?: string; version?: string } | null;
      const identity: SpecimenIdentity = `${data?.version || "__current__"}:${data?.example || "__default__"}`;
      const frame = specimenFramesRef.current.get(identity);
      if (!data || data.id !== id || !frame || ev.source !== frame.contentWindow) return;
      if (data.type === "preview-ready") {
        setReadyExamples((current) => {
          const next = new Set(current);
          next.add(identity);
          return next;
        });
        setSpecimenErrors((current) => {
          if (!current[identity]) return current;
          return withoutRecordKey(current, identity);
        });
      } else if (data.type === "preview-error") {
        setSpecimenErrors((current) => ({
          ...current,
          [identity]: data.message || t(strings.components.editor.previewFailed),
        }));
      } else if (data.type === "rcl-preview-props-applied") {
        setOverrideStatus((current) => ({ ...current, [identity]: "applied" }));
      } else if (data.type === "rcl-preview-props-reset") {
        setOverrideStatus((current) => ({ ...current, [identity]: "idle" }));
        setOverrideMessages((current) => ({ ...current, [identity]: "" }));
      } else if (data.type === "rcl-preview-props-error") {
        setOverrideStatus((current) => ({ ...current, [identity]: "error" }));
        setOverrideMessages((current) => ({ ...current, [identity]: data.message || t(strings.components.editor.propsRejected) }));
      }
    };
    window.addEventListener("message", handler);
    return () => window.removeEventListener("message", handler);
  }, [id, t]);

  // The app owns the resolved theme. Emulator "system" means follow the
  // app decision, never a separate OS media decision inside the iframe.
  useEffect(() => {
    if (!previewReady) return;
    postToPreviewFrames({ type: "rcl-resolved-theme", theme: resolvedPreviewTheme });
  }, [postToPreviewFrames, previewReady, resolvedPreviewTheme]);

  useEffect(() => {
    // A new baselineSha (post-save or first fetch) means the iframe is
    // about to reload with the freshly bundled output — flip the badge
    // back to "waiting" until the harness re-announces preview-ready.
    setPreviewState("waiting");
    setPreviewMessage("");
    setReadyExamples(new Set());
    setSpecimenErrors({});
    setSpecimenRetries({});
    setSpecimenOverrides({});
    setOverrideStatus({});
    setOverrideMessages({});
  }, [baselineSha, previewReloadKey]);

  const examples = useMemo(() => examplesQuery.data?.examples ?? [], [examplesQuery.data?.examples]);
  const expectedReadyCount = Math.max(1, examples.length);

  useEffect(() => {
    if (activeSpecimen || examples.length === 0) return;
    activateSpecimen(specimenIdentity(examples[0]));
  }, [activeSpecimen, activateSpecimen, examples]);

  useEffect(() => {
    if (previewState !== "waiting") return;
    if (readyExamples.size + Object.keys(specimenErrors).length >= expectedReadyCount) {
      setPreviewState("ready");
    }
  }, [expectedReadyCount, previewState, readyExamples, specimenErrors]);

  useEffect(() => {
    if (!contentQuery.data || previewState !== "waiting") return undefined;
    if (previewTimeoutRef.current) clearTimeout(previewTimeoutRef.current);
    previewTimeoutRef.current = setTimeout(() => {
      setPreviewMessage(t(strings.components.editor.previewTimeout));
    }, PREVIEW_LOAD_TIMEOUT_MS);
    return () => {
      if (previewTimeoutRef.current) clearTimeout(previewTimeoutRef.current);
    };
  }, [baselineSha, contentQuery.data, previewReloadKey, previewState, t]);

  useEffect(() => {
    if (contentQuery.data) {
      setBuffer(contentQuery.data.content);
      setBaselineSha(contentQuery.data.sha256);
    }
  }, [contentQuery.data]);

  useEffect(() => {
    return () => {
      if (savedToastTimerRef.current) clearTimeout(savedToastTimerRef.current);
      if (previewTimeoutRef.current) clearTimeout(previewTimeoutRef.current);
    };
  }, []);

  const saveMutation = useMutation({
    mutationFn: () =>
      componentsClient.updateComponentContent({
        id,
        content: buffer,
        expectedSha256: baselineSha,
        ...(selectedFile ? { path: selectedFile } : {}),
      }),
    onSuccess: (resp) => {
      setBaselineSha(resp.sha256);
      setShowSaved(true);
      if (savedToastTimerRef.current) clearTimeout(savedToastTimerRef.current);
      savedToastTimerRef.current = setTimeout(() => setShowSaved(false), 2500);
      void queryClient.invalidateQueries({ queryKey: ["components", "content", id] });
      void queryClient.invalidateQueries({ queryKey: ["components"] });
    },
  });

  const readOnly = Boolean(selectedVersion);
  const dirty = !readOnly && !!contentQuery.data && buffer !== contentQuery.data.content;

  const handleBeforeMount = (monaco: Monaco) => {
    const diagnosticsOptions = {
      noSemanticValidation: true,
      noSyntaxValidation: false,
    };
    monaco.languages.typescript.typescriptDefaults.setDiagnosticsOptions(diagnosticsOptions);
    monaco.languages.typescript.javascriptDefaults.setDiagnosticsOptions(diagnosticsOptions);
  };

  const handleMount = (monacoEditor: editor.IStandaloneCodeEditor, monaco: Monaco) => {
    monacoEditor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
      if (!saveMutation.isPending) saveMutation.mutate();
    });
  };

  const visiblePanes = workspace.order.filter((pane) => pane !== "preview" || renderable).filter((pane) => workspace.visible[pane]);

  useEffect(() => {
    try {
      window.localStorage.setItem(PANEL_LAYOUT_STORAGE_KEY, JSON.stringify(workspace));
    } catch {
      // Workspace persistence is a convenience; editing never depends on storage access.
    }
  }, [workspace]);

  useEffect(() => {
    if (!comparison) return;
    setFilesView("diff");
    setWorkspace((current) => ({
      ...current,
      visible: { ...current.visible, files: true },
    }));
    setMode("files");
  }, [comparison]);

  const showPane = (pane: WorkspacePane) => {
    if (pane === "preview" && !renderable) return;
    setWorkspace((current) => ({
      ...current,
      visible: { ...current.visible, [pane]: true },
    }));
  };

  const hidePane = (pane: WorkspacePane) => {
    if (visiblePanes.length <= 1) return;
    setWorkspace((current) => ({
      ...current,
      visible: { ...current.visible, [pane]: false },
    }));
    if (mode === pane) {
      const nextPane = visiblePanes.find((candidate) => candidate !== pane);
      if (nextPane) setMode(nextPane);
    }
  };

  const resetPaneDrag = () => {
    setActiveDraggedPane(null);
    setDraggedOverPane(null);
  };

  const startPaneDrag = ({ active }: DragStartEvent) => {
    if (!desktopLayout || !isWorkspacePane(active.id)) return;
    setActiveDraggedPane(active.id);
    setDraggedOverPane(active.id);
  };

  const updatePaneDragTarget = ({ over }: DragOverEvent) => {
    setDraggedOverPane(isWorkspacePane(over?.id) ? over.id : null);
  };

  const finishPaneDrag = ({ active, over }: DragEndEvent) => {
    resetPaneDrag();
    if (!desktopLayout || !isWorkspacePane(active.id) || !isWorkspacePane(over?.id)) return;
    const activePane = active.id;
    const overPane = over.id;
    setWorkspace((current) => ({
      ...current,
      order: reorderWorkspacePanes(current.order, activePane, overPane),
    }));
  };

  const selectMode = (pane: WorkspacePane) => {
    showPane(pane);
    setMode(pane);
  };

  const saveDesktopPanelLayout = (layout: Record<string, number>) => {
    if (!desktopLayout) return;
    setWorkspace((current) => ({ ...current, layout: { ...current.layout, ...layout } }));
  };

  const selectFile = (path: string) => {
    setSelectedFile(path);
    setFilesView("source");
  };

  const paneHeader = (pane: WorkspacePane, label: string, icon: ReactNode, dragHandle: SortableHandle) => {
    return (
      <header className="flex h-10 shrink-0 items-center justify-between gap-2 border-b border-app-border bg-app-surface px-2">
        <div className="flex min-w-0 items-center gap-1.5 text-xs font-semibold text-app-foreground">
          {icon}
          <span className="truncate">{label}</span>
        </div>
        <div className="flex shrink-0 items-center gap-0.5">
          <button
            ref={dragHandle.setActivatorNodeRef}
            type="button"
            data-testid={selectors.components.editor.workspacePaneDragHandle}
            data-pane={pane}
            className="hidden h-7 w-7 cursor-grab touch-none items-center justify-center rounded-control border border-app-border bg-app-surface text-app-foreground transition hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50 active:cursor-grabbing lg:inline-flex"
            {...dragHandle.attributes}
            {...dragHandle.listeners}
            aria-label={t(strings.components.editor.reorderPane, { pane: label })}
          >
            <GripVertical aria-hidden className="h-3.5 w-3.5" />
          </button>
          <Button
            type="button"
            variant="secondary"
            data-testid={selectors.components.editor.workspacePaneClose}
            data-pane={pane}
            aria-label={t("components.editor.closePane", { defaultValue: "Close pane" })}
            disabled={visiblePanes.length <= 1}
            onClick={() => hidePane(pane)}
            className="h-7 w-7 p-0"
          >
            <X aria-hidden className="h-3.5 w-3.5" />
          </Button>
        </div>
      </header>
    );
  };

  const specimens: Array<ComponentExample | undefined> = examples.length > 0 ? examples : [undefined];
  const comparisonActive = comparedSpecimens.size > 0;
  const visibleSpecimens = comparisonActive
    ? specimens.filter((example) => comparedSpecimens.has(specimenIdentity(example)))
    : specimens;
  const activeExample = specimens.find((example) => specimenIdentity(example) === activeSpecimen);
  const activeSpecimenLabel = activeExample?.displayName || activeExample?.name;
  const paneLabels: Record<WorkspacePane, string> = {
    files: t(strings.components.editor.files),
    preview: t(strings.components.editor.previewMode),
    details: t(strings.components.editor.info),
  };
  const panePosition = (pane: WorkspacePane) => visiblePanes.indexOf(pane) + 1;
  const applyPropsOverride = (props: Record<string, unknown>) => {
    if (!activeSpecimen) return;
    const example = specimens.find((candidate) => specimenIdentity(candidate) === activeSpecimen);
    setSpecimenOverrides((current) => ({ ...current, [activeSpecimen]: props }));
    setOverrideStatus((current) => ({ ...current, [activeSpecimen]: "applying" }));
    setOverrideMessages((current) => ({ ...current, [activeSpecimen]: "" }));
    postToSpecimen(activeSpecimen, {
      type: "rcl-preview-props-override",
      componentId: id,
      example: example?.name || "",
      version: example?.version || "",
      props,
    });
  };
  const resetPropsOverride = () => {
    if (!activeSpecimen) return;
    const example = specimens.find((candidate) => specimenIdentity(candidate) === activeSpecimen);
    setSpecimenOverrides((current) => {
      return withoutRecordKey(current, activeSpecimen);
    });
    setOverrideStatus((current) => ({ ...current, [activeSpecimen]: "applying" }));
    postToSpecimen(activeSpecimen, {
      type: "rcl-preview-props-reset",
      componentId: id,
      example: example?.name || "",
      version: example?.version || "",
    });
  };

  return (
    <AssetWorkspace testId={selectors.components.editor.panel} label={t(strings.components.editor.title, { libraryId })}>
      <header className="shrink-0 border-b border-app-border bg-app-surface">
        <div className="flex min-w-0 items-center justify-between gap-3 px-4 py-2">
          <div className="min-w-0">
            <h2
              data-testid={selectors.components.editor.title}
              className="truncate text-base font-semibold text-app-foreground"
            >
              {libraryId}
            </h2>
            <div className="mt-0.5 flex min-w-0 flex-wrap items-center gap-2 text-xs text-app-muted-foreground">
              <span className="truncate">{t(strings.components.editor.subtitle)}</span>
              {baselineSha && (
                <span
                  data-testid={selectors.components.editor.shaHash}
                  className="font-mono text-app-muted-foreground"
                >
                  {t(strings.components.editor.sha, { sha: baselineSha.slice(0, 12) })}
                </span>
              )}
              {dirty && (
                <StatusBadge
                  data-testid={selectors.components.editor.dirtyBadge}
                  tone="warning"
                >
                  {t(strings.components.editor.dirty)}
                </StatusBadge>
              )}
              {previewReady && (desktopLayout || mode === "preview") && (
                <StatusBadge
                  data-testid={selectors.components.editor.previewBadge}
                  tone="success"
                >
                  {t(strings.components.editor.previewReady)}
                </StatusBadge>
              )}
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-1.5">
            <details className="relative hidden lg:block">
              <summary
                data-testid={selectors.components.editor.workspaceAddPane}
                className="flex h-8 cursor-pointer list-none items-center gap-1.5 rounded-control border border-app-border bg-app-surface px-2 text-xs font-medium text-app-foreground transition hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
              >
                <Plus aria-hidden className="h-3.5 w-3.5" />
                {t("components.editor.addPane", { defaultValue: "Add pane" })}
              </summary>
              <div className="absolute right-0 z-20 mt-1 w-40 rounded-control border border-app-border bg-app-surface p-1 shadow-lg">
              {DEFAULT_PANE_ORDER.filter((pane) => (pane !== "preview" || renderable) && !workspace.visible[pane]).map((pane) => (
                  <Button
                    key={pane}
                    data-testid={selectors.components.editor.workspacePaneRestore}
                    data-pane={pane}
                    type="button"
                    variant="secondary"
                    className="h-8 w-full justify-start px-2 text-xs"
                    onClick={() => showPane(pane)}
                  >
                    {pane === "files"
                      ? t("components.editor.files", { defaultValue: "Files" })
                      : pane === "preview"
                        ? t(strings.components.editor.previewMode)
                        : t(strings.components.editor.info)}
                  </Button>
                ))}
                {DEFAULT_PANE_ORDER.filter((pane) => pane !== "preview" || renderable).every((pane) => workspace.visible[pane]) && (
                  <p className="px-2 py-1 text-xs text-app-muted-foreground">
                    {t("components.editor.allPanesOpen", { defaultValue: "All panes are open" })}
                  </p>
                )}
              </div>
            </details>
            {renderable && <Button
              data-testid={selectors.components.editor.closeButton}
              onClick={onClose}
              className="h-8 gap-1.5 rounded-md px-2 text-xs"
              variant="secondary"
            >
              <ArrowLeft aria-hidden className="h-3.5 w-3.5" />
              {t(strings.components.editor.close)}
            </Button>}
          </div>
        </div>

        <div className="flex min-w-0 flex-wrap items-center gap-2 border-t border-app-border lg:hidden">
          <div
            data-testid={selectors.components.editor.modeSwitch}
            className="flex shrink-0 overflow-hidden rounded-md border border-app-border"
          >
            <Button
              data-testid={selectors.components.editor.previewModeButton}
              type="button"
              variant={mode === "preview" ? "primary" : "secondary"}
              className="h-7 gap-1.5 rounded-none px-2 text-xs"
              onClick={() => selectMode("preview")}
            >
              <Eye aria-hidden className="h-3.5 w-3.5" />
              {t(strings.components.editor.previewMode)}
            </Button>
            <Button
              data-testid={selectors.components.editor.codeModeButton}
              type="button"
              variant={mode === "files" ? "primary" : "secondary"}
              className="h-7 gap-1.5 rounded-none px-2 text-xs"
              onClick={() => selectMode("files")}
            >
              <Code2 aria-hidden className="h-3.5 w-3.5" />
              {t(strings.components.editor.codeMode)}
            </Button>
            <Button
              type="button"
              variant={mode === "details" ? "primary" : "secondary"}
              className="h-7 gap-1.5 rounded-none px-2 text-xs lg:hidden"
              onClick={() => selectMode("details")}
            >
              <Info aria-hidden className="h-3.5 w-3.5" />
              {t(strings.components.editor.info)}
            </Button>
          </div>
        </div>
      </header>

      {contentQuery.isLoading && (
        <p
          data-testid={selectors.components.editor.loading}
          className="p-4 text-app-foreground"
        >
          {t(strings.components.editor.loading)}
        </p>
      )}

      {contentQuery.error && (
        <p
          data-testid={selectors.components.editor.error}
          className="p-4 text-app-danger"
        >
          {errorMessage(contentQuery.error, t)}
        </p>
      )}

      {saveMutation.error && (
        <p
          data-testid={selectors.components.editor.error}
          className="p-4 text-app-danger"
        >
          {errorMessage(saveMutation.error, t)}
        </p>
      )}

      {contentQuery.data && (
        <div className="min-h-0 flex-1">
          {selectedVersion && (
            <p className="border-b border-app-border bg-app-warning/10 px-4 py-2 text-xs text-app-warning">
              {t(strings.components.editor.viewingVersion, { version: selectedVersion })}
            </p>
          )}
          <DndContext
            sensors={workspaceDragSensors}
            collisionDetection={closestCenter}
            modifiers={[restrictWorkspaceDragToHorizontalAxis]}
            accessibility={{
              screenReaderInstructions: {
                draggable: t(strings.components.editor.paneDragInstructions),
              },
              announcements: {
                onDragStart: ({ active }) => isWorkspacePane(active.id)
                  ? t(strings.components.editor.paneDragStarted, { pane: paneLabels[active.id] })
                  : undefined,
                onDragOver: ({ active, over }) => isWorkspacePane(active.id) && isWorkspacePane(over?.id)
                  ? t(strings.components.editor.paneDragOver, {
                    pane: paneLabels[active.id],
                    position: panePosition(over.id),
                    total: visiblePanes.length,
                  })
                  : undefined,
                onDragEnd: ({ active, over }) => isWorkspacePane(active.id) && isWorkspacePane(over?.id)
                  ? t(strings.components.editor.paneDragEnded, {
                    pane: paneLabels[active.id],
                    position: panePosition(over.id),
                    total: visiblePanes.length,
                  })
                  : t(strings.components.editor.paneDragCancelled),
                onDragCancel: ({ active }) => isWorkspacePane(active.id)
                  ? t(strings.components.editor.paneDragCancelledWithPane, { pane: paneLabels[active.id] })
                  : t(strings.components.editor.paneDragCancelled),
              },
            }}
            onDragStart={startPaneDrag}
            onDragOver={updatePaneDragTarget}
            onDragEnd={finishPaneDrag}
            onDragCancel={resetPaneDrag}
          >
            <SortableContext items={visiblePanes} strategy={horizontalListSortingStrategy}>
              <Group
                id="component-editor-panels"
                orientation={desktopLayout ? "horizontal" : "vertical"}
                defaultLayout={desktopLayout ? workspace.layout : { [mode]: 100 }}
                onLayoutChanged={saveDesktopPanelLayout}
                className="h-full min-h-0"
              >
                {visiblePanes.map((pane, index) => (
                  <Fragment key={pane}>
                    <SortableWorkspacePanel
                      pane={pane}
                      disabled={!desktopLayout}
                      dropEdge={draggedOverPane === pane
                        ? getWorkspaceDropEdge(workspace.order, activeDraggedPane, draggedOverPane)
                        : null}
                      minSize="15%"
                      defaultSize={workspace.layout[pane] ?? (100 / visiblePanes.length)}
                      className={mode === pane ? "" : "max-lg:hidden"}
                    >
                      {(dragHandle) => (
                        <>
                  {pane === "files" && (
                    <div data-testid={renderable ? selectors.components.editor.workspacePane : selectors.assets.hookSource} data-pane="files" className="flex h-full min-h-0 flex-col bg-app-background">
                      {paneHeader("files", paneLabels.files, <FileCode2 aria-hidden className="h-3.5 w-3.5" />, dragHandle)}
                      <div className="flex shrink-0 min-w-0 gap-1 overflow-x-auto border-b border-app-border bg-app-surface px-2 py-1.5">
                        <Button data-testid={selectors.components.editor.filesTreeTab} type="button" variant={filesView === "tree" ? "primary" : "secondary"} className="h-7 shrink-0 px-2 text-xs" onClick={() => setFilesView("tree")}>{t("components.editor.fileTree", { defaultValue: "Files" })}</Button>
                        {activeVersionFiles.map((file) => (
                          <Button key={file.path} data-testid={selectors.components.editor.filesSourceTab} data-file={file.path} type="button" variant={filesView === "source" && (selectedFile === file.path || (!selectedFile && file.isEntry)) ? "primary" : "secondary"} className="h-7 shrink-0 px-2 text-xs" onClick={() => selectFile(file.isEntry ? "" : file.path)}>{file.path}</Button>
                        ))}
                        {comparison && (
                          <div className="flex shrink-0">
                            <Button type="button" variant={filesView === "diff" ? "primary" : "secondary"} className="h-7 rounded-r-none px-2 text-xs" onClick={() => setFilesView("diff")}>
                              {t("components.editor.diffTab", { defaultValue: "Diff" })}: {comparison.fromLabel} → {comparison.toLabel}
                            </Button>
                            <Button
                              data-testid={selectors.components.editor.filesDiffClose}
                              type="button"
                              variant={filesView === "diff" ? "primary" : "secondary"}
                              aria-label={t("components.editor.closeComparison", { defaultValue: "Close comparison" })}
                              className="h-7 w-7 rounded-l-none border-l border-app-border p-0"
                              onClick={() => {
                                onCloseComparison?.();
                                setFilesView("source");
                              }}
                            >
                              <X aria-hidden className="h-3.5 w-3.5" />
                            </Button>
                          </div>
                        )}
                      </div>
                      {filesView === "tree" ? (
                        <div className="min-h-0 flex-1 overflow-auto p-2">
                          <AdoptionFileTree componentId={id} version={selectedVersion} files={activeVersionFiles} selectedFile={selectedFile} onSelectFile={selectFile} template={selectedTemplate} templates={ADOPTION_TEMPLATES} onSelectTemplate={setSelectedTemplate} />
                        </div>
                      ) : filesView === "diff" && comparison ? (
                        <div className="min-h-0 flex-1 overflow-auto p-2"><VersionDiffViewer rows={comparison.rows} /></div>
                      ) : (
                        <>
                          <div className="flex shrink-0 flex-wrap items-center justify-between gap-1.5 border-b border-app-border bg-app-surface px-2 py-1.5">
                            <div className="flex items-center gap-1">
                              <Button data-testid={selectors.components.editor.saveButton} onClick={() => saveMutation.mutate()} disabled={readOnly || !dirty || saveMutation.isPending || contentQuery.isLoading} className="h-7 gap-1 px-2 text-xs"><Save aria-hidden className="h-3.5 w-3.5" />{saveMutation.isPending ? t(strings.components.editor.saving) : t(strings.components.editor.save)}</Button>
                              <Button data-testid={selectors.components.editor.filesRevertButton} type="button" variant="secondary" onClick={() => setBuffer(contentQuery.data.content)} disabled={readOnly || !dirty} className="h-7 gap-1 px-2 text-xs"><RotateCcw aria-hidden className="h-3.5 w-3.5" />{t("components.editor.revert", { defaultValue: "Revert" })}</Button>
                            </div>
                            <div className="flex items-center gap-1">
                              <Button data-testid={selectors.components.editor.filesWrapButton} type="button" variant="secondary" aria-pressed={wordWrap === "on"} onClick={() => setWordWrap((current) => current === "on" ? "off" : "on")} className="h-7 px-2 text-xs">{t("components.editor.wrap", { defaultValue: "Wrap" })}</Button>
                              <Button data-testid={selectors.components.editor.filesFontDecrease} type="button" variant="secondary" aria-label={t("components.editor.decreaseFont", { defaultValue: "Decrease font size" })} onClick={() => setFontSize((current) => Math.max(11, current - 1))} disabled={fontSize <= 11} className="h-7 w-7 p-0"><Minus aria-hidden className="h-3.5 w-3.5" /></Button>
                              <Button data-testid={selectors.components.editor.filesFontIncrease} type="button" variant="secondary" aria-label={t("components.editor.increaseFont", { defaultValue: "Increase font size" })} onClick={() => setFontSize((current) => Math.min(20, current + 1))} disabled={fontSize >= 20} className="h-7 w-7 p-0"><Plus aria-hidden className="h-3.5 w-3.5" /></Button>
                            </div>
                          </div>
                          <div data-testid={selectors.components.editor.surface} className="min-h-0 flex-1 overflow-hidden">
                            <Editor height="100%" language="typescript" path={selectedFile || `${libraryId || id}.tsx`} value={buffer} onChange={(value) => setBuffer(value ?? "")} beforeMount={handleBeforeMount} onMount={handleMount} theme={appResolvedTheme === "dark" ? "vs-dark" : "vs"} options={{ fontSize, lineNumbers: "on", minimap: { enabled: false }, scrollBeyondLastLine: false, tabSize: 2, insertSpaces: true, wordWrap, automaticLayout: true, readOnly }} />
                          </div>
                        </>
                      )}
                    </div>
                  )}
                  {pane === "preview" && (
                    <div data-testid={selectors.components.editor.workspacePane} data-pane="preview" className="flex h-full min-h-0 flex-col overflow-hidden bg-app-background">
                      {paneHeader("preview", paneLabels.preview, <Eye aria-hidden className="h-3.5 w-3.5" />, dragHandle)}
                      <div className="flex shrink-0 flex-wrap items-center gap-2 border-b border-app-border bg-app-surface px-2 py-1.5">
                        <ThemeSwitcher postToFrames={postToPreviewFrames} previewReady={previewReady} colorScheme={filters.colorScheme} setColorScheme={filters.setColorScheme} filters={filters} />
                        <EmulatorToolbar emulator={emulator} />
                      </div>
                      <div className="relative min-h-0 flex-1">
                        <EmulatorViewport emulator={emulator} filters={filters}>
                          <div
                            data-testid={selectors.components.editor.gallery}
                            className="h-full overflow-auto bg-app-background p-3"
                            style={{ width: emulator.displayWidth, height: emulator.displayHeight }}
                          >
                            {comparisonActive && (
                              <div data-testid={selectors.components.editor.comparisonToolbar} className="mb-3 flex flex-wrap items-center justify-between gap-2 rounded-md border border-app-primary/30 bg-app-primary/10 px-3 py-2">
                                <p className="text-xs text-app-foreground">{t(strings.components.editor.comparing)}</p>
                                <Button data-testid={selectors.components.editor.comparisonClear} type="button" variant="secondary" className="h-7 px-2 text-xs" onClick={() => setComparedSpecimens(new Set())}>
                                  {t(strings.components.editor.showAllSpecimens)}
                                </Button>
                              </div>
                            )}
                            <p data-testid={selectors.components.editor.galleryStatus} aria-live="polite" className="mb-2 text-xs text-app-muted-foreground">
                              {previewMessage || t(strings.components.editor.specimenStatus, { ready: readyExamples.size, total: specimens.length })}
                            </p>
                            <div className={comparisonActive ? "grid min-w-0 grid-cols-1 gap-3 md:grid-cols-2" : "grid min-w-0 grid-cols-[repeat(auto-fit,minmax(20rem,1fr))] gap-3"}>
                              {visibleSpecimens.map((example) => {
                                const identity = specimenIdentity(example);
                                const title = example?.displayName || example?.name || t(strings.components.editor.previewHeading);
                                const error = specimenErrors[identity];
                                const isActive = activeSpecimen === identity;
                                const compared = comparedSpecimens.has(identity);
                                return (
                                  <section key={identity} data-testid={selectors.components.editor.exampleCard} data-specimen={identity} className={`min-w-0 overflow-hidden rounded-md border bg-app-surface ${isActive ? "border-app-primary ring-1 ring-app-primary/30" : "border-app-border"}`}>
                                    <header className="flex items-center justify-between gap-2 border-b border-app-border px-3 py-2">
                                      <h3 data-testid={selectors.components.editor.exampleTitle} className="min-w-0 truncate text-sm font-semibold text-app-foreground">{title}</h3>
                                      <div className="flex shrink-0 gap-1">
                                        <Button data-testid={selectors.components.editor.exampleFocus} type="button" variant={isActive ? "primary" : "secondary"} className="h-7 px-2 text-xs" onClick={() => activateSpecimen(identity)}>{t(strings.components.editor.focusSpecimen)}</Button>
                                        {examples.length > 1 && <Button data-testid={selectors.components.editor.exampleCompare} type="button" variant={compared ? "primary" : "secondary"} aria-pressed={compared} disabled={!compared && comparedSpecimens.size >= 2} className="h-7 px-2 text-xs" onClick={() => toggleComparison(identity)}>{t(strings.components.editor.compareSpecimen)}</Button>}
                                      </div>
                                    </header>
                                    {error ? (
                                      <div data-testid={selectors.components.editor.specimenError} className="flex min-h-[18rem] flex-col items-center justify-center gap-3 bg-app-danger/5 p-4 text-center">
                                        <p className="text-xs text-app-danger">{error}</p>
                                        <Button data-testid={selectors.components.editor.specimenRetry} type="button" variant="secondary" className="h-8 px-3 text-xs" onClick={() => retrySpecimen(identity)}>{t(strings.components.editor.previewRetry)}</Button>
                                      </div>
                                    ) : (
                                      <iframe
                                        data-testid={selectors.components.editor.previewFrame}
                                        data-specimen={identity}
                                        title={`${t(strings.components.editor.previewHeading)} - ${title}`}
                                        src={harnessUrl(id, baselineSha, previewReloadKey + (specimenRetries[identity] ?? 0), example, selectedVersion)}
                                        sandbox="allow-scripts allow-same-origin"
                                        ref={(frame) => registerPreviewFrame(identity, frame)}
                                        onLoad={() => {
                                          activateSpecimen(identity);
                                          postToPreviewFrames({ type: "rcl-resolved-theme", theme: resolvedPreviewTheme });
                                        }}
                                        onError={() => setSpecimenErrors((current) => ({ ...current, [identity]: t(strings.components.editor.previewFailed) }))}
                                        className="block h-[20rem] w-full border-0 bg-white"
                                      />
                                    )}
                                  </section>
                                );
                              })}
                            </div>
                          </div>
                        </EmulatorViewport>
                      </div>
                      {activeSpecimen && (
                        <PropsExperimentPanel
                          key={activeSpecimen}
                          example={activeExample}
                          status={overrideStatus[activeSpecimen] ?? (specimenOverrides[activeSpecimen] ? "applied" : "idle")}
                          message={overrideMessages[activeSpecimen]}
                          onApply={applyPropsOverride}
                          onReset={resetPropsOverride}
                        />
                      )}
                      <InspectorPanel inspector={inspector} specimenLabel={activeSpecimenLabel} />
                    </div>
                  )}
                  {pane === "details" && (
                    <aside data-testid={selectors.components.editor.workspacePane} data-pane="details" className="flex h-full min-h-0 flex-col overflow-hidden bg-app-surface">
                      {paneHeader("details", paneLabels.details, <Info aria-hidden className="h-3.5 w-3.5" />, dragHandle)}
                      <div data-testid={selectors.components.editor.infoDialog} className="min-h-0 flex-1 overflow-auto p-4">{metadataSlot ?? <p className="text-sm text-app-muted-foreground">{t(strings.components.editor.noInfo)}</p>}</div>
                    </aside>
                  )}
                        </>
                      )}
                    </SortableWorkspacePanel>
                    {index < visiblePanes.length - 1 && <Separator className="hidden w-1 shrink-0 bg-app-border hover:bg-app-primary lg:block" />}
                  </Fragment>
                ))}
              </Group>
            </SortableContext>
            <DragOverlay dropAnimation={null}>
              {activeDraggedPane && (
                <div className="flex h-10 items-center gap-2 rounded-control border border-app-primary bg-app-surface px-3 text-xs font-semibold text-app-foreground shadow-xl">
                  <GripVertical aria-hidden className="h-3.5 w-3.5" />
                  <span>{paneLabels[activeDraggedPane]}</span>
                </div>
              )}
            </DragOverlay>
          </DndContext>
        </div>
      )}

      {showSaved && (
        <p
          data-testid={selectors.components.editor.savedToast}
          className="absolute bottom-3 right-3 rounded-md bg-app-success/10 px-3 py-2 text-xs text-app-success shadow-lg"
        >
          {t(strings.components.editor.saved)}
        </p>
      )}
    </AssetWorkspace>
  );
}
