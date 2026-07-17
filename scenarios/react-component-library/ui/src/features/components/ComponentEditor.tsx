import { Fragment, type ReactNode, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Editor, { type Monaco } from "@monaco-editor/react";
import type { editor } from "monaco-editor";
import { ArrowLeft, Eye, FileCode2, Info, Menu, Minus, PanelsLeftRight, Plus, RotateCcw, Save, X } from "lucide-react";
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
import { Dialog } from "../../components/ui/dialog";
import { ThemeSwitcher } from "./ThemeSwitcher";
import { PropsExperimentPanel } from "./PropsExperimentPanel";
import { AdoptionFileTree } from "./AdoptionFileTree";
import { ADOPTION_TEMPLATES, DEFAULT_ADOPTION_TEMPLATE } from "./adoptionTemplates";
import { VersionDiffViewer } from "../versions/VersionDiffViewer";
import { AssetWorkspace } from "../assets/AssetWorkspace";
import type { DiffRow } from "../../api/versions";
import { ExperienceSurface } from "../../../../library/components/ExperienceSurface/versions/1.0.0/ExperienceSurface";
import { WorkspaceHeader } from "../../../../library/components/WorkspaceHeader/versions/1.0.0/WorkspaceHeader";
import { useShellNavigation } from "../../components/ShellNavigationContext";

const PREVIEW_LOAD_TIMEOUT_MS = 8_000;
const PANEL_LAYOUT_STORAGE_KEY = "rcl.component-editor.split-view.v1";
const DEFAULT_DESKTOP_PANEL_LAYOUT = { primary: 50, secondary: 50 };
const DEFAULT_PANE_ORDER = ["details", "files", "preview"] as const;

type WorkspacePane = (typeof DEFAULT_PANE_ORDER)[number];
type FilesView = "tree" | "source" | "diff";
type SpecimenIdentity = `${string}:${string}`;


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

function loadSplitLayout(): Record<string, number> {
  const fallback = DEFAULT_DESKTOP_PANEL_LAYOUT;
  try {
    const raw = window.localStorage.getItem(PANEL_LAYOUT_STORAGE_KEY);
    if (!raw) return fallback;
    const saved = JSON.parse(raw) as Record<string, number>;
    return { ...fallback, ...saved };
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
  /** Asset-level navigation stays visible while Files or Preview replaces Details. */
  navigationSlot?: ReactNode;
  selectedVersion?: string;
  comparison?: ComparisonSession | null;
  onCloseComparison?: () => void;
  /** Hooks share the source/details workspace but intentionally omit preview. */
  renderable?: boolean;
  /** Asset pages control the normal one-pane view from their URL-backed tabs. */
  activePane?: WorkspacePane;
  onActivePaneChange?: (pane: WorkspacePane) => void;
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
  navigationSlot,
  selectedVersion,
  comparison,
  onCloseComparison,
  renderable = true,
  activePane,
  onActivePaneChange,
}: ComponentEditorProps) {
  const { t } = useTranslation();
  const shellNavigation = useShellNavigation();
  const queryClient = useQueryClient();
  const emulator = useDeviceEmulation();
  const filters = useDeviceFilters();
  const desktopLayout = useMediaQuery("(min-width: 1024px)");
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
  const [mobileTool, setMobileTool] = useState<"props" | "inspector" | null>(null);
  const [specimenOverrides, setSpecimenOverrides] = useState<Record<string, Record<string, unknown>>>({});
  const [overrideStatus, setOverrideStatus] = useState<Record<string, "idle" | "applying" | "applied" | "error">>({});
  const [overrideMessages, setOverrideMessages] = useState<Record<string, string>>({});
  const previewReloadKey = 0;
  const [uncontrolledPane, setUncontrolledPane] = useState<WorkspacePane>("files");
  const currentPane = activePane ?? uncontrolledPane;
  const [splitView, setSplitView] = useState(false);
  const [secondaryPane, setSecondaryPane] = useState<WorkspacePane>(renderable ? "preview" : "files");
  const [splitLayout, setSplitLayout] = useState<Record<string, number>>(loadSplitLayout);
  const [filesView, setFilesView] = useState<FilesView>("source");
  const [wordWrap, setWordWrap] = useState<"on" | "off">("on");
  const [fontSize, setFontSize] = useState(13);
  const savedToastTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const previewTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const previewReady = previewState === "ready";
  // A preview is usable once all specimens settle. Failed individual specimens
  // retain their own retry UI, so the composed region is honestly partial
  // rather than falsely ready when any of them fail.
  const previewExperienceState = previewState === "waiting"
    ? "loading"
    : previewState === "error"
    ? "error"
    : Object.keys(specimenErrors).length > 0
    ? "partial"
    : "ready";
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

  useEffect(() => {
    try {
      window.localStorage.setItem(PANEL_LAYOUT_STORAGE_KEY, JSON.stringify(splitLayout));
    } catch {
      // Split sizing is a convenience; editing never depends on storage access.
    }
  }, [splitLayout]);

  useEffect(() => {
    if (!desktopLayout) setSplitView(false);
  }, [desktopLayout]);

  useEffect(() => {
    if (!comparison) return;
    setFilesView("diff");
    selectPane("files");
  }, [comparison]);

  const selectPane = (pane: WorkspacePane) => {
    if (pane === "preview" && !renderable) return;
    if (onActivePaneChange) onActivePaneChange(pane);
    else setUncontrolledPane(pane);
  };

  const availablePanes = DEFAULT_PANE_ORDER.filter((pane) => pane !== "preview" || renderable);
  const visiblePanes = splitView ? [currentPane, secondaryPane].filter((pane, index, panes) => panes.indexOf(pane) === index) : [currentPane];

  const toggleSplitView = () => {
    if (!desktopLayout) return;
    if (!splitView && secondaryPane === currentPane) {
      setSecondaryPane(availablePanes.find((pane) => pane !== currentPane) ?? currentPane);
    }
    setSplitView((current) => !current);
  };

  const selectSplitPane = (index: number, pane: WorkspacePane) => {
    if (pane === "preview" && !renderable) return;
    if (index === 0) {
      selectPane(pane);
      if (pane === secondaryPane) setSecondaryPane(currentPane);
      return;
    }
    setSecondaryPane(pane === currentPane ? (availablePanes.find((candidate) => candidate !== currentPane) ?? pane) : pane);
  };

  const saveDesktopPanelLayout = (layout: Record<string, number>) => {
    if (!desktopLayout) return;
    setSplitLayout((current) => ({ ...current, ...layout }));
  };

  const selectFile = (path: string) => {
    setSelectedFile(path);
    setFilesView("source");
  };

  const paneHeader = (pane: WorkspacePane, index: number, label: string, icon: ReactNode) => {
    return (
      <header className="flex h-10 shrink-0 items-center justify-between gap-2 border-b border-app-border bg-app-surface px-2">
        <details className="relative min-w-0">
          <summary data-testid="components-editor-split-pane-switcher" data-pane={pane} className="flex cursor-pointer list-none items-center gap-1.5 rounded-control px-1 py-1 text-xs font-semibold text-app-foreground hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50">
            {icon}<span className="truncate">{label}</span>
          </summary>
          <div className="absolute left-0 z-20 mt-1 w-36 rounded-control border border-app-border bg-app-surface p-1 shadow-lg">
            {availablePanes.map((candidate) => <Button key={candidate} type="button" variant="secondary" className="h-8 w-full justify-start px-2 text-xs" onClick={() => selectSplitPane(index, candidate)}>{paneLabels[candidate]}</Button>)}
          </div>
        </details>
      </header>
    );
  };

  const specimens: Array<ComponentExample | undefined> = examples.length > 0 ? examples : [undefined];
  const comparisonActive = comparedSpecimens.size === 2;
  const visibleSpecimens = comparisonActive
    ? specimens.filter((example) => comparedSpecimens.has(specimenIdentity(example)))
    : activeSpecimen
    ? specimens.filter((example) => specimenIdentity(example) === activeSpecimen)
    : [specimens[0]];
  const activeExample = specimens.find((example) => specimenIdentity(example) === activeSpecimen);
  const activeSpecimenLabel = activeExample?.displayName || activeExample?.name;
  const paneLabels: Record<WorkspacePane, string> = {
    files: t(strings.components.editor.files),
    preview: t(strings.components.editor.previewMode),
    details: t(strings.components.editor.info),
  };
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
      <WorkspaceHeader
        title={<span data-testid={selectors.components.editor.title}>{libraryId}</span>}
        description={t(strings.components.editor.subtitle)}
        leading={shellNavigation.sidebarCollapsed ? <button type="button" onClick={shellNavigation.openSidebar} aria-label={t("nav.openDrawer", { defaultValue: "Open navigation" })} data-testid="workspace-header-open-sidebar" className="touch-target inline-flex items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"><Menu aria-hidden className="h-5 w-5" /></button> : undefined}
        actions={<div className="flex items-center gap-1.5">{dirty && <StatusBadge data-testid={selectors.components.editor.dirtyBadge} tone="warning">{t(strings.components.editor.dirty)}</StatusBadge>}{previewReady && (desktopLayout || currentPane === "preview") && <StatusBadge data-testid={selectors.components.editor.previewBadge} tone="success">{t(strings.components.editor.previewReady)}</StatusBadge>}{availablePanes.length > 1 && <Button data-testid="components-editor-split-view-toggle" type="button" variant={splitView ? "primary" : "secondary"} aria-pressed={splitView} onClick={toggleSplitView} className="hidden h-8 gap-1.5 px-2 text-xs lg:inline-flex"><PanelsLeftRight aria-hidden className="h-3.5 w-3.5" />{t("components.editor.splitView", { defaultValue: "Split view" })}</Button>}{renderable && <Button data-testid={selectors.components.editor.closeButton} onClick={onClose} className="h-8 gap-1.5 rounded-md px-2 text-xs" variant="secondary"><ArrowLeft aria-hidden className="h-3.5 w-3.5" />{t(strings.components.editor.close)}</Button>}</div>}
      />

      {navigationSlot && <div className="shrink-0 border-b border-app-border bg-app-surface px-4">{navigationSlot}</div>}

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
          <Group
                id="component-editor-panels"
                orientation={splitView && desktopLayout ? "horizontal" : "vertical"}
                defaultLayout={splitView && desktopLayout ? splitLayout : { primary: 100 }}
                onLayoutChanged={saveDesktopPanelLayout}
                className="h-full min-h-0"
              >
                {visiblePanes.map((pane, index) => (
                  <Fragment key={pane}>
                    <Panel
                      id={index === 0 ? "primary" : "secondary"}
                      minSize="15%"
                      defaultSize={splitView ? splitLayout[index === 0 ? "primary" : "secondary"] : 100}
                    >
                      <div className="h-full min-h-0">
                  {pane === "files" && (
                    <div data-testid={renderable ? selectors.components.editor.workspacePane : selectors.assets.hookSource} data-pane="files" className="flex h-full min-h-0 flex-col bg-app-background">
                      {splitView && paneHeader("files", index, paneLabels.files, <FileCode2 aria-hidden className="h-3.5 w-3.5" />)}
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
                    <ExperienceSurface
                      surfaceId="component-preview"
                      state={previewExperienceState}
                      data-testid={selectors.components.editor.workspacePane}
                      data-pane="preview"
                      className="flex h-full min-h-0 flex-col overflow-hidden bg-app-background"
                    >
                      {splitView && paneHeader("preview", index, paneLabels.preview, <Eye aria-hidden className="h-3.5 w-3.5" />)}
                      <div className="flex shrink-0 flex-wrap items-center gap-2 border-b border-app-border bg-app-surface px-2 py-1.5">
                        <nav className="flex max-w-full gap-1 overflow-x-auto" aria-label="Component states">
                          {specimens.map((example) => { const identity = specimenIdentity(example); const selected = identity === activeSpecimen; return <Button key={identity} type="button" variant={selected ? "primary" : "secondary"} className="h-8 shrink-0 px-2 text-xs" aria-current={selected ? "true" : undefined} onClick={() => { setComparedSpecimens(new Set()); activateSpecimen(identity); }}>{example?.displayName || example?.name || "Default"}</Button>; })}
                        </nav>
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
                                return (
                                  <section key={identity} data-testid={selectors.components.editor.exampleCard} data-specimen={identity} className={`min-w-0 overflow-hidden rounded-md border bg-app-surface ${isActive ? "border-app-primary ring-1 ring-app-primary/30" : "border-app-border"}`}>
                                    <header className="flex items-center justify-between gap-2 border-b border-app-border px-3 py-2">
                                      <h3 data-testid={selectors.components.editor.exampleTitle} className="min-w-0 truncate text-sm font-semibold text-app-foreground">{title}</h3>
                                      {examples.length > 1 && <Button data-testid={selectors.components.editor.exampleCompare} type="button" variant={comparedSpecimens.has(identity) ? "primary" : "secondary"} aria-pressed={comparedSpecimens.has(identity)} disabled={!comparedSpecimens.has(identity) && comparedSpecimens.size >= 2} className="h-7 px-2 text-xs" onClick={() => toggleComparison(identity)}>{t(strings.components.editor.compareSpecimen)}</Button>}
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
                                        sandbox="allow-scripts"
                                        ref={(frame) => registerPreviewFrame(identity, frame)}
                                        onLoad={() => {
                                          // Loading a later iframe must not steal the user's active
                                          // specimen: that remounts the keyed props editor and discards
                                          // an in-progress temporary override draft.
                                          setActiveSpecimen((current) => current ?? identity);
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
                      {activeSpecimen && <>
                        <div className="hidden lg:block border-t border-app-border bg-app-surface p-3"><div className="grid gap-3 xl:grid-cols-[minmax(18rem,0.8fr)_minmax(20rem,1.2fr)]"><PropsExperimentPanel key={activeSpecimen} example={activeExample} status={overrideStatus[activeSpecimen] ?? (specimenOverrides[activeSpecimen] ? "applied" : "idle")} message={overrideMessages[activeSpecimen]} onApply={applyPropsOverride} onReset={resetPropsOverride} /><InspectorPanel inspector={inspector} specimenLabel={activeSpecimenLabel} /></div></div>
                        <div className="flex shrink-0 gap-2 border-t border-app-border bg-app-surface p-2 lg:hidden"><Button type="button" className="h-9 flex-1 text-xs" onClick={() => setMobileTool("props")}>{t("components.editor.editProps", { defaultValue: "Edit props" })}</Button><Button type="button" variant="secondary" className="h-9 flex-1 text-xs" onClick={() => setMobileTool("inspector")}>{t("components.inspector.title", { defaultValue: "Inspect" })}</Button><Button type="button" variant="secondary" className="h-9 px-3 text-xs" onClick={resetPropsOverride}>{t("components.editor.reset", { defaultValue: "Reset" })}</Button></div>
                      </>}
                      <Dialog open={mobileTool !== null} onClose={() => setMobileTool(null)} title={mobileTool === "props" ? t(strings.components.editor.tryProps) : t("components.inspector.title", { defaultValue: "Inspect" })} closeLabel={t("common.close", { defaultValue: "Close" })} className="lg:hidden">
                        {mobileTool === "props" && activeSpecimen ? <PropsExperimentPanel key={activeSpecimen} example={activeExample} status={overrideStatus[activeSpecimen] ?? (specimenOverrides[activeSpecimen] ? "applied" : "idle")} message={overrideMessages[activeSpecimen]} onApply={applyPropsOverride} onReset={resetPropsOverride} /> : null}
                        {mobileTool === "inspector" ? <InspectorPanel inspector={inspector} specimenLabel={activeSpecimenLabel} /> : null}
                      </Dialog>
                    </ExperienceSurface>
                  )}
                  {pane === "details" && (
                    <aside data-testid={selectors.components.editor.workspacePane} data-pane="details" className="flex h-full min-h-0 flex-col overflow-hidden bg-app-surface">
                      {splitView && paneHeader("details", index, paneLabels.details, <Info aria-hidden className="h-3.5 w-3.5" />)}
                      <div data-testid={selectors.components.editor.infoDialog} className="min-h-0 flex-1 overflow-auto p-4">{metadataSlot ?? <p className="text-sm text-app-muted-foreground">{t(strings.components.editor.noInfo)}</p>}</div>
                    </aside>
                  )}
                      </div>
                    </Panel>
                    {index < visiblePanes.length - 1 && <Separator className="w-1 shrink-0 bg-app-border hover:bg-app-primary" />}
                  </Fragment>
                ))}
          </Group>
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
