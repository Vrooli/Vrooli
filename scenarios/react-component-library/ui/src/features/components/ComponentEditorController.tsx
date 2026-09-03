/** @vrooliComponentSource navigation.master-detail */
import { type ReactNode, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTheme } from "../../components/theme/useTheme";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import {
  componentsClient,
  listComponentStories,
  listPreviewFrames,
  persistPreviewFrame,
  type ComponentStory,
  type PreviewFrameCandidate,
} from "../../api/components";
import { ComponentVersionStatus } from "@vrooli/proto-types/react-component-library/v1/components/components_pb";
import { useComponentInspector } from "../../hooks/useComponentInspector";
import { useDeviceEmulation } from "../../hooks/useDeviceEmulation";
import { useDeviceFilters } from "../../hooks/useDeviceFilters";
import { useMediaQuery } from "../../hooks/useMediaQuery";
import { errorMessage } from "../../lib/errorMessage";
import { ComponentEditorView } from "./ComponentEditorView";
import type { PreviewDiagnostics } from "./ComponentEditorTools";
import type { PreviewKit } from "./ThemeSwitcher";
import { DEFAULT_ADOPTION_TEMPLATE } from "./adoptionTemplates";
import type { DiffRow } from "../../api/versions";
import { useShellNavigation } from "../../components/ShellNavigationContext";
import { configureEditorBeforeMount, configureEditorMount } from "./componentEditorMonaco";
import { parseStorySpecimens, specimenIdentity } from "./componentEditorStories";
import type { PreviewSpecimen, SpecimenIdentity } from "./ComponentEditorStage";
import { useComponentEditorPanes, type WorkspacePane } from "./useComponentEditorPanes";
import { useComponentPreviewMessaging, type PreviewEvent } from "./useComponentPreviewMessaging";
import { useComponentEditorTools } from "./useComponentEditorTools";

const PREVIEW_LOAD_TIMEOUT_MS = 8_000;

export interface ComparisonSession {
  fromLabel: string;
  toLabel: string;
  rows: DiffRow[];
}

function readPreviewPreference<T>(key: string, fallback: T): T {
  if (typeof window === "undefined") return fallback;
  try {
    const value = window.localStorage.getItem(key);
    return value === null ? fallback : (JSON.parse(value) as T);
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
interface ComponentEditorProps {
  id: string;
  libraryId: string;
  /** Manifest-owned released version used when no historical version is selected. */
  latestVersion?: string;
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
  selectedStory?: string;
  onSelectedStoryChange?: (story: string) => void;
  /** Lets the asset-page root expose preview readiness to external automation. */
  onPreviewExperienceStateChange?: (state: "loading" | "partial" | "ready" | "error") => void;
  /** Structural assets open in the single-specimen stage; ordinary components keep the gallery. */
  stageMode?: boolean;
  /** Standalone preview routes omit editor navigation and workspace chrome. */
  chromeless?: boolean;
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
export function ComponentEditorImpl({
  id,
  libraryId,
  latestVersion,
  onClose,
  metadataSlot,
  navigationSlot,
  selectedVersion,
  comparison,
  onCloseComparison,
  renderable = true,
  activePane,
  onActivePaneChange,
  selectedStory,
  onSelectedStoryChange,
  onPreviewExperienceStateChange,
  stageMode: initialStageMode,
  chromeless = false,
}: ComponentEditorProps) {
  const { t } = useTranslation();
  const shellNavigation = useShellNavigation();
  const queryClient = useQueryClient();
  const emulator = useDeviceEmulation();
  const filters = useDeviceFilters();
  const desktopLayout = useMediaQuery("(min-width: 1024px)");
  const { resolved: appResolvedTheme } = useTheme();
  const previewFrameRef = useRef<HTMLIFrameElement | null>(null);
  const previewStageRef = useRef<HTMLDivElement | null>(null);
  const previewCanvasRef = useRef<HTMLDivElement | null>(null);
  const previewFramesRef = useRef(new Set<HTMLIFrameElement>());
  const specimenFramesRef = useRef(new Map<SpecimenIdentity, HTMLIFrameElement>());
  const inspector = useComponentInspector(previewFrameRef);
  const [selectedFile, setSelectedFile] = useState("");
  const [selectedTemplate, setSelectedTemplate] = useState(DEFAULT_ADOPTION_TEMPLATE);
  const versionsQuery = useQuery({
    queryKey: ["components", "versions", id],
    queryFn: () => componentsClient.listComponentVersions({ componentId: id, limit: 100 }),
  });
  // Version list order is presentation/history data, not a version-selection
  // contract. The manifest's latest pointer is the only authoritative default;
  // falling back to the first row can select an old prerelease draft after an
  // index refresh.
  const activeVersion =
    selectedVersion || latestVersion || versionsQuery.data?.versions[0]?.version || "";
  const activeVersionFiles = ((versionsQuery.data?.versions ?? []).find(
    (version) => version.version === activeVersion,
  )?.files ?? []) as Array<{ path: string; isEntry: boolean }>;
  const activeVersionRecord = (versionsQuery.data?.versions ?? []).find(
    (version) => version.version === activeVersion,
  );

  const contentQuery = useQuery({
    queryKey: ["components", "content", id, selectedVersion ?? "current", selectedFile],
    queryFn: async (): Promise<{ content: string; sha256: string }> => {
      // Non-entry files are version-local artifacts. Resolve them through the
      // version projection that populated the Files tab so read-only metadata
      // and Preview harness source cannot accidentally come from a shared
      // current-source alias.
      if ((selectedFile.endsWith(".json") || selectedFile === "story.tsx") && activeVersion) {
        const artifact = await componentsClient.getComponentVersionContent({
          componentId: id,
          version: activeVersion,
          path: selectedFile,
        });
        return {
          content: artifact.content,
          sha256: artifact.version?.contentSha256 ?? "",
        };
      }
      if (!selectedVersion) {
        const current = await componentsClient.getComponentContent({
          id,
          ...(selectedFile ? { path: selectedFile } : {}),
        });
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

  const storiesQuery = useQuery({
    // The editor must resolve exactly the contract that owns the source being
    // previewed. An empty version used to fetch every historical contract;
    // before an index existed that degraded to a fabricated Default specimen.
    queryKey: ["components", "stories", id, activeVersion],
    queryFn: () => listComponentStories({ componentId: id, version: activeVersion, limit: 1 }),
    enabled: renderable && Boolean(activeVersion),
  });

  const [frameOverride, setFrameOverride] = useState<PreviewFrameCandidate>();
  const [frameSaveMessage, setFrameSaveMessage] = useState("");
  const frameCandidatesQuery = useQuery({
    queryKey: ["components", "preview-frames", id, activeVersion, selectedStory || ""],
    queryFn: () =>
      listPreviewFrames({
        componentId: id,
        version: activeVersion,
        storyId: selectedStory,
      }),
    enabled: renderable && Boolean(activeVersion),
  });
  useEffect(() => {
    setFrameOverride(undefined);
    setFrameSaveMessage("");
  }, [id, activeVersion, selectedStory]);

  const [buffer, setBuffer] = useState<string>("");
  const [baselineSha, setBaselineSha] = useState<string>("");
  const [showSaved, setShowSaved] = useState(false);
  const [previewState, setPreviewState] = useState<"waiting" | "ready" | "error">("waiting");
  const [previewMessage, setPreviewMessage] = useState("");
  const [readyExamples, setReadyExamples] = useState<ReadonlySet<string>>(() => new Set());
  const [specimenErrors, setSpecimenErrors] = useState<Record<string, string>>({});
  const [specimenRetries, setSpecimenRetries] = useState<Record<string, number>>({});
  const [comparedSpecimens, setComparedSpecimens] = useState<ReadonlySet<SpecimenIdentity>>(
    () => new Set(),
  );
  const [activeSpecimen, setActiveSpecimen] = useState<SpecimenIdentity | null>(null);
  const [previewToolsCollapsed, setPreviewToolsCollapsed] = useState(true);
  const [previewFullscreen, setPreviewFullscreen] = useState(false);
  const [stageMode, setStageMode] = useState(
    () => initialStageMode ?? readPreviewPreference("rcl.preview.view", "focus") === "focus",
  );
  const [specimenOverrides, setSpecimenOverrides] = useState<
    Record<string, Record<string, unknown>>
  >({});
  const [overrideStatus, setOverrideStatus] = useState<
    Record<string, "idle" | "applying" | "applied" | "error">
  >({});
  const [overrideMessages, setOverrideMessages] = useState<Record<string, string>>({});
  const [previewEvents, setPreviewEvents] = useState<PreviewEvent[]>([]);
  const [previewKit, setPreviewKit] = useState<PreviewKit>(
    () => readPreviewPreference("rcl.preview.kit", "vrooli-default") || "vrooli-default",
  );
  const [frameEnabled] = useState(() => readPreviewPreference("rcl.preview.frame", true));
  const previewReloadKey = 0;

  useEffect(() => {
    try {
      window.localStorage.setItem(
        "rcl.preview.view",
        JSON.stringify(stageMode ? "focus" : "canvas"),
      );
      window.localStorage.setItem("rcl.preview.kit", JSON.stringify(previewKit));
      window.localStorage.setItem("rcl.preview.frame", JSON.stringify(frameEnabled));
    } catch {
      // Preferences are best-effort in private and embedded browser contexts.
    }
  }, [frameEnabled, previewKit, stageMode]);
  const {
    currentPane,
    splitView,
    splitLayout,
    filesView,
    wordWrap,
    fontSize,
    setFilesView,
    setWordWrap,
    setFontSize,
    availablePanes,
    visiblePanes,
    toggleSplitView,
    selectSplitPane,
    saveDesktopPanelLayout,
    selectFile,
  } = useComponentEditorPanes({
    activePane,
    onActivePaneChange,
    renderable,
    desktopLayout,
    comparison,
    setSelectedFile,
  });
  const savedToastTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const previewTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const previewReady = previewState === "ready";
  const previewExperienceState =
    previewState === "waiting"
      ? "loading"
      : previewState === "error"
        ? "error"
        : Object.keys(specimenErrors).length > 0
          ? "partial"
          : "ready";
  useEffect(() => {
    onPreviewExperienceStateChange?.(previewExperienceState);
  }, [onPreviewExperienceStateChange, previewExperienceState]);
  useEffect(() => {
    if (!stageMode) return;
    setPreviewToolsCollapsed(true);
    const fitPreview = () => emulator.fitToPane(previewStageRef.current);
    const collapseFrame = window.requestAnimationFrame(() =>
      window.requestAnimationFrame(fitPreview),
    );
    const resizeObserver = new ResizeObserver(() => {
      window.requestAnimationFrame(() => emulator.fitToPane(previewStageRef.current));
    });
    const viewportFrame = previewStageRef.current?.querySelector<HTMLElement>(
      "[data-emulator-viewport-frame]",
    );
    if (viewportFrame) resizeObserver.observe(viewportFrame);
    else if (previewStageRef.current) resizeObserver.observe(previewStageRef.current);
    return () => {
      window.cancelAnimationFrame(collapseFrame);
      resizeObserver.disconnect();
    };
  }, [emulator.fitToPane, previewReady, stageMode]);

  useEffect(() => {
    if (!stageMode) return;
    const leaveSpecimen = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      setStageMode(false);
      setComparedSpecimens(new Set());
    };
    window.addEventListener("keydown", leaveSpecimen);
    return () => window.removeEventListener("keydown", leaveSpecimen);
  }, [stageMode]);

  const resolvedPreviewTheme =
    filters.colorScheme === "system" ? appResolvedTheme : filters.colorScheme;

  useEffect(() => {
    const handleFullscreenChange = () => {
      setPreviewFullscreen(document.fullscreenElement === previewStageRef.current);
    };
    document.addEventListener("fullscreenchange", handleFullscreenChange);
    return () => document.removeEventListener("fullscreenchange", handleFullscreenChange);
  }, []);

  const {
    postToPreviewFrames,
    postToSpecimen,
    registerPreviewFrame,
    activateSpecimen,
    retrySpecimen,
  } = useComponentPreviewMessaging({
    id,
    previewFrameRef,
    previewFramesRef,
    specimenFramesRef,
    setActiveSpecimen,
    onSelectedStoryChange,
    setReadyExamples,
    setSpecimenErrors,
    setSpecimenRetries,
    setOverrideStatus,
    setOverrideMessages,
    setPreviewEvents,
    previewFailedMessage: t(strings.components.editor.previewFailed),
    propsRejectedMessage: t(strings.components.editor.propsRejected),
  });
  const toggleComparison = useCallback(
    (identity: SpecimenIdentity) => {
      setComparedSpecimens((current) => {
        const next = new Set(current);
        if (next.has(identity)) next.delete(identity);
        else if (next.size < 4) next.add(identity);
        return next;
      });
      activateSpecimen(identity);
    },
    [activateSpecimen],
  );
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
    setPreviewEvents([]);
  }, [baselineSha, previewKit, previewReloadKey]);

  const storySpecimens = useMemo(
    () => parseStorySpecimens(storiesQuery.data?.stories ?? []),
    [storiesQuery.data?.stories],
  );
  const examples = storySpecimens;
  const selectAllComparison = useCallback(() => {
    const selected = examples.slice(0, 4).map((example) => specimenIdentity(example));
    setComparedSpecimens(new Set(selected));
    if (selected[0]) activateSpecimen(selected[0]);
  }, [activateSpecimen, examples]);
  // Only the active specimen is mounted in the normal workspace; comparison
  // mounts exactly two. Waiting for every indexed example left the region in
  // loading forever even after the visible preview had announced readiness.
  const expectedReadyCount = comparedSpecimens.size >= 2 ? comparedSpecimens.size : 1;

  useEffect(() => {
    // A transient loading/fallback specimen can outlive the story query when
    // the editor is entered through client-side catalog navigation. Reconcile
    // by identity against the settled contract instead of treating any
    // non-null value as a valid selection.
    if (
      examples.length === 0 ||
      (activeSpecimen && examples.some((example) => specimenIdentity(example) === activeSpecimen))
    )
      return;
    const restored = selectedStory
      ? examples.find((example) => example.storyId === selectedStory)
      : undefined;
    activateSpecimen(specimenIdentity(restored ?? examples[0]));
  }, [activeSpecimen, activateSpecimen, examples, selectedStory]);

  useEffect(() => {
    if (previewState !== "waiting") return;
    if (readyExamples.size + Object.keys(specimenErrors).length >= expectedReadyCount) {
      setPreviewState("ready");
    }
  }, [expectedReadyCount, previewState, readyExamples, specimenErrors]);

  useEffect(() => {
    if (previewState !== "waiting") return;
    if (contentQuery.isError) {
      setPreviewMessage(errorMessage(contentQuery.error, t));
      setPreviewState("error");
      return;
    }
    if (storiesQuery.isError) {
      setPreviewMessage(errorMessage(storiesQuery.error, t));
      setPreviewState("error");
      return;
    }
  }, [
    contentQuery.error,
    contentQuery.isError,
    previewState,
    storiesQuery.error,
    storiesQuery.isError,
    t,
  ]);

  useEffect(() => {
    if (!contentQuery.data || previewState !== "waiting") return undefined;
    if (previewTimeoutRef.current) clearTimeout(previewTimeoutRef.current);
    previewTimeoutRef.current = setTimeout(() => {
      setPreviewMessage(t(strings.components.editor.previewTimeout));
      setPreviewState("error");
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

  const persistFrameMutation = useMutation({
    mutationFn: async () => {
      if (!frameOverride) throw new Error("Select a compatible frame first");
      const storyId = selectedStory || "";
      if (!storyId) throw new Error("Select a story before saving its frame");
      return persistPreviewFrame({
        componentId: id,
        version: activeVersion,
        storyId,
        asset: frameOverride.asset,
        frameVersion: frameOverride.version,
        region: frameOverride.region,
        capability: frameOverride.capability,
        fixture: frameOverride.fixture,
      });
    },
    onSuccess: (response) => {
      setFrameSaveMessage(`Saved to draft ${response.version}`);
      setFrameOverride(undefined);
      void queryClient.invalidateQueries({ queryKey: ["components", "versions", id] });
      void queryClient.invalidateQueries({ queryKey: ["components", "stories", id] });
      void queryClient.invalidateQueries({ queryKey: ["components", "preview-frames", id] });
    },
    onError: (error) => setFrameSaveMessage(errorMessage(error, t)),
  });

  const releasedVersion = Boolean(selectedVersion) && activeVersionRecord?.status !== ComponentVersionStatus.DRAFT;
  const readOnly = releasedVersion || selectedFile.endsWith(".json");
  const dirty = !readOnly && !!contentQuery.data && buffer !== contentQuery.data.content;

  const handleBeforeMount = configureEditorBeforeMount;
  const handleMount = (monacoEditor: Parameters<typeof configureEditorMount>[0], monaco: Parameters<typeof configureEditorMount>[1]) =>
    configureEditorMount(monacoEditor, monaco, () => !saveMutation.isPending && saveMutation.mutate());

  const specimens: Array<PreviewSpecimen | undefined> = storiesQuery.isLoading
    ? []
    : examples.length > 0
      ? examples
      : [undefined];
  const comparisonActive = !stageMode && comparedSpecimens.size >= 2;
  const visibleSpecimens = stageMode
    ? specimens
        .filter(
          (example) =>
            specimenIdentity(example) === (activeSpecimen ?? specimenIdentity(specimens[0])),
        )
        .slice(0, 1)
    : comparisonActive
      ? specimens.filter((example) => comparedSpecimens.has(specimenIdentity(example)))
      : specimens;
  const activeExample = specimens.find((example) => specimenIdentity(example) === activeSpecimen);
  const activeStoryContract: ComponentStory | undefined = (storiesQuery.data?.stories ?? []).find((story) => story.version === (activeExample?.version || activeVersion));
  const compatibleFrameCandidates = (frameCandidatesQuery.data?.candidates ?? []).filter(
    (candidate) => candidate.compatible,
  );
  const framePickerEnabled = renderable && Boolean(activeVersion);
  const activeSpecimenLabel = activeExample?.displayName || activeExample?.name;
  const activeSpecimenError = activeSpecimen ? specimenErrors[activeSpecimen] : undefined;
  const previewDiagnostics: PreviewDiagnostics = {
    iframeUrl: previewFrameRef.current?.src ?? "",
    componentId: id,
    storyId: activeExample?.storyId ?? "",
    version: activeExample?.version ?? activeVersion,
    kit: previewKit,
    theme: resolvedPreviewTheme,
    frame: frameOverride?.asset ?? "default",
    ...(activeSpecimenError ? { error: activeSpecimenError } : {}),
  };
  const paneLabels: Record<WorkspacePane, string> = { files: t(strings.components.editor.files), preview: t(strings.components.editor.previewMode), details: t(strings.components.editor.info) };
  const { editorToolProps, togglePreviewFullscreen } = useComponentEditorTools({
    activeSpecimen,
    specimens,
    id,
    postToSpecimen,
    setSpecimenOverrides,
    setOverrideStatus,
    setOverrideMessages,
    setPreviewEvents,
    activeExample,
    activeSpecimenLabel,
    storyContract: activeStoryContract,
    inspector,
    specimenOverrides,
    overrideStatus,
    overrideMessages,
    previewEvents,
    previewDiagnostics,
    previewStageRef,
    previewFullscreen,
    setPreviewFullscreen,
  });
  const togglePreviewTools = () => setPreviewToolsCollapsed((collapsed) => !collapsed);

  return (
    <ComponentEditorView
      model={{
        t,
        selectors,
        strings,
        chromeless,
        storiesQuery,
        libraryId,
        shellNavigation,
        dirty,
        previewReady,
        desktopLayout,
        currentPane,
        availablePanes,
        splitView,
        toggleSplitView,
        renderable,
        onClose,
        navigationSlot,
        contentQuery,
        activeVersionFiles,
        selectedVersion,
        splitLayout,
        saveDesktopPanelLayout,
        visiblePanes,
        visibleSpecimens,
        readyExamples,
        previewMessage,
        specimenErrors,
        specimenRetries,
        comparedSpecimens,
        frameEnabled,
        baselineSha,
        previewReloadKey,
        resolvedPreviewTheme,
        previewCanvasRef,
        id,
        activeVersion,
        releasedVersion,
        selectedFile,
        selectedTemplate,
        filesView,
        comparison,
        buffer,
        appResolvedTheme,
        fontSize,
        wordWrap,
        readOnly,
        saveMutation,
        handleBeforeMount,
        handleMount,
        selectFile,
        setSelectedTemplate,
        setFilesView,
        setBuffer,
        setWordWrap,
        setFontSize,
        onCloseComparison,
        previewStageRef,
        previewFullscreen,
        showSaved,
        previewExperienceState,
        paneLabels,
        selectSplitPane,
        specimens,
        activeSpecimen,
        specimenIdentity,
        activateSpecimen,
        framePickerEnabled,
        frameCandidatesQuery,
        frameOverride,
        compatibleFrameCandidates,
        setFrameSaveMessage,
        setFrameOverride,
        frameSaveMessage,
        persistFrameMutation,
        filters,
        previewKit,
        setPreviewKit,
        stageMode,
        setStageMode,
        setPreviewToolsCollapsed,
        setComparedSpecimens,
        comparisonActive,
        toggleComparison,
        selectAllComparison,
        emulator,
        togglePreviewTools,
        togglePreviewFullscreen,
        previewToolsCollapsed,
        editorToolProps,
        setActiveSpecimen,
        setSpecimenErrors,
        retrySpecimen,
        registerPreviewFrame,
        postToPreviewFrames,
        metadataSlot,
      }}
    />
  );
}
