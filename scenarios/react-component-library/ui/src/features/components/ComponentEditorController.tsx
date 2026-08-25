/** @vrooliComponentSource navigation.master-detail */
import { Fragment, type ReactNode, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type Monaco } from "@monaco-editor/react";
import type { editor } from "monaco-editor";
import {
  ArrowLeft,
  Eye,
  Info,
  Maximize2,
  Menu,
  Minimize2,
  PanelsLeftRight,
  SlidersHorizontal,
} from "lucide-react";
import { Group, Panel, Separator } from "react-resizable-panels";

import { Button } from "../../components/Button";
import { IconButton } from "../../components/IconButton";
import { StatusBadge } from "../../components/StatusBadge";
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
import { useComponentInspector } from "../../hooks/useComponentInspector";
import { useDeviceEmulation } from "../../hooks/useDeviceEmulation";
import { useDeviceFilters } from "../../hooks/useDeviceFilters";
import { useMediaQuery } from "../../hooks/useMediaQuery";
import { errorMessage } from "../../lib/errorMessage";
import { EmulatorToolbar } from "./EmulatorChrome";
import { ComponentEditorStage } from "./ComponentEditorStage";
import { ComponentEditorSource } from "./ComponentEditorSource";
import {
  ComponentEditorMobileTools,
  ComponentEditorTools,
  type PreviewDiagnostics,
} from "./ComponentEditorTools";
import { ThemeSwitcher, type PreviewKit } from "./ThemeSwitcher";
import { DEFAULT_ADOPTION_TEMPLATE } from "./adoptionTemplates";
import { AssetWorkspace } from "../assets/AssetWorkspace";
import type { DiffRow } from "../../api/versions";
import { ExperienceSurface } from "../../components/ExperienceSurface/versions/1.0.0/ExperienceSurface";
import { WorkspaceHeader } from "../../components/WorkspaceHeader";
import { useShellNavigation } from "../../components/ShellNavigationContext";

const PREVIEW_LOAD_TIMEOUT_MS = 8_000;
const PANEL_LAYOUT_STORAGE_KEY = "rcl.component-editor.split-view.v1";
const DEFAULT_DESKTOP_PANEL_LAYOUT = { primary: 50, secondary: 50 };
const DEFAULT_PANE_ORDER = ["details", "files", "preview"] as const;

type WorkspacePane = (typeof DEFAULT_PANE_ORDER)[number];
type FilesView = "tree" | "source" | "diff";
type SpecimenIdentity = `${string}:${string}`;
type PreviewSpecimen = {
  id: string;
  componentId: string;
  libraryId: string;
  version: string;
  name: string;
  displayName: string;
  propsJson: string;
  environment: Record<string, string>;
  expectJson: string;
  sourcePath: string;
  storyId: string;
  description?: string;
};

type PreviewEvent = { story: string; name: string; args: unknown[]; ts: number };
const MAX_PREVIEW_EVENTS = 200;

function SourceLoadingSkeleton() {
  return (
    <div
      data-testid={selectors.components.editor.loading}
      role="status"
      aria-label="Loading source"
      className="flex h-full min-h-0 flex-col bg-app-background"
    >
      <div className="flex h-control-md shrink-0 items-center border-b border-app-border bg-app-surface px-space-2xs">
        <span className="h-3 w-16 animate-pulse rounded bg-app-surface-muted" />
      </div>
      <div className="flex shrink-0 gap-space-3xs border-b border-app-border bg-app-surface px-space-2xs py-space-2xs">
        <span className="h-control-compact w-8 animate-pulse rounded bg-app-surface-muted" />
        <span className="h-control-compact w-36 animate-pulse rounded bg-app-surface-muted" />
        <span className="h-control-compact w-28 animate-pulse rounded bg-app-surface-muted" />
      </div>
      <div className="flex min-h-0 flex-1 flex-col gap-space-2xs p-space-sm" aria-hidden="true">
        {["w-11/12", "w-8/12", "w-10/12", "w-6/12", "w-9/12", "w-7/12"].map((width) => (
          <span key={width} className={`h-3 animate-pulse rounded bg-app-surface-muted ${width}`} />
        ))}
      </div>
    </div>
  );
}

function specimenIdentity(example?: Pick<PreviewSpecimen, "version" | "name">): SpecimenIdentity {
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
  const [previewKit, setPreviewKit] = useState<PreviewKit>(() =>
    readPreviewPreference("rcl.preview.kit", "vrooli-default"),
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
  const [uncontrolledPane, setUncontrolledPane] = useState<WorkspacePane>("files");
  const currentPane = activePane ?? uncontrolledPane;
  const [splitView, setSplitView] = useState(false);
  const [secondaryPane, setSecondaryPane] = useState<WorkspacePane>(
    renderable ? "preview" : "files",
  );
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
    // Structural assets are judged as compositions, so the specimen owns the
    // initial vertical space. The tools remain one intentional toggle away.
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

  const postToPreviewFrames = useCallback((message: unknown) => {
    for (const frame of previewFramesRef.current) {
      frame.contentWindow?.postMessage(message, "*");
    }
  }, []);

  const postToSpecimen = useCallback((identity: SpecimenIdentity, message: unknown) => {
    specimenFramesRef.current.get(identity)?.contentWindow?.postMessage(message, "*");
  }, []);

  const registerPreviewFrame = useCallback(
    (identity: SpecimenIdentity, frame: HTMLIFrameElement | null) => {
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
    },
    [],
  );

  const activateSpecimen = useCallback(
    (identity: SpecimenIdentity) => {
      setActiveSpecimen(identity);
      previewFrameRef.current = specimenFramesRef.current.get(identity) ?? null;
      const story = identity.split(":").slice(1).join(":");
      if (story && story !== "__default__") onSelectedStoryChange?.(story);
    },
    [onSelectedStoryChange],
  );

  const retrySpecimen = useCallback((identity: SpecimenIdentity) => {
    setSpecimenErrors((current) => {
      return withoutRecordKey(current, identity);
    });
    setSpecimenRetries((current) => ({ ...current, [identity]: (current[identity] ?? 0) + 1 }));
  }, []);

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

  useEffect(() => {
    const handler = (ev: MessageEvent) => {
      const data = ev.data as {
        type?: string;
        id?: string;
        message?: string;
        story?: string;
        version?: string;
        name?: string;
        args?: unknown[];
        ts?: number;
        passed?: boolean;
        failures?: Array<{ message?: string }>;
      } | null;
      const identity: SpecimenIdentity = `${data?.version || "__current__"}:${data?.story || "__default__"}`;
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
        setOverrideMessages((current) => ({
          ...current,
          [identity]: data.message || t(strings.components.editor.propsRejected),
        }));
      } else if (data.type === "rcl-story-result" && data.passed === false) {
        const details = (data.failures ?? [])
          .map((failure) => failure.message)
          .filter(Boolean)
          .join(" ");
        setSpecimenErrors((current) => ({
          ...current,
          [identity]: details || "Story interactions or expectations failed.",
        }));
      } else if (data.type === "rcl-preview-event" && typeof data.name === "string") {
        const eventName = data.name;
        setPreviewEvents((current) =>
          [
            {
              story: data.story ?? "",
              name: eventName,
              args: Array.isArray(data.args) ? data.args : [],
              ts: typeof data.ts === "number" ? data.ts : Date.now(),
            },
            ...current,
          ].slice(0, MAX_PREVIEW_EVENTS),
        );
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
    setPreviewEvents([]);
  }, [baselineSha, previewKit, previewReloadKey]);

  const storySpecimens = useMemo<PreviewSpecimen[]>(
    () =>
      (storiesQuery.data?.stories ?? []).flatMap((contract) => {
        try {
          const definitions = JSON.parse(contract.storiesJson) as Array<{
            id?: unknown;
            name?: unknown;
            description?: unknown;
            args?: unknown;
            environment?: unknown;
            expect?: unknown;
          }>;
          if (!Array.isArray(definitions)) return [];
          return definitions.flatMap((definition) => {
            if (
              typeof definition.id !== "string" ||
              typeof definition.name !== "string" ||
              !definition.args ||
              typeof definition.args !== "object" ||
              Array.isArray(definition.args)
            )
              return [];
            const environment: Record<string, string> =
              definition.environment &&
              typeof definition.environment === "object" &&
              !Array.isArray(definition.environment)
                ? (Object.fromEntries(
                    Object.entries(definition.environment as Record<string, unknown>).filter(
                      ([, value]) => typeof value === "string",
                    ),
                  ) as Record<string, string>)
                : {};
            return [
              {
                id: `${contract.id}:${definition.id}`,
                componentId: contract.componentId,
                libraryId: contract.libraryId,
                version: contract.version,
                name: definition.id,
                displayName: definition.name,
                description:
                  typeof definition.description === "string" && definition.description.trim()
                    ? definition.description
                    : undefined,
                propsJson: JSON.stringify(definition.args),
                environment,
                expectJson: JSON.stringify(
                  Array.isArray(definition.expect) ? definition.expect : [],
                ),
                sourcePath: contract.sourcePath,
                storyId: definition.id,
              },
            ];
          });
        } catch {
          return [];
        }
      }),
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

  const readOnly = Boolean(selectedVersion) || selectedFile.endsWith(".json");
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
  const visiblePanes = splitView
    ? [currentPane, secondaryPane].filter((pane, index, panes) => panes.indexOf(pane) === index)
    : [currentPane];

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
    setSecondaryPane(
      pane === currentPane
        ? (availablePanes.find((candidate) => candidate !== currentPane) ?? pane)
        : pane,
    );
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
      <header className="flex h-control-md shrink-0 items-center justify-between gap-space-2xs border-b border-app-border bg-app-surface px-space-2xs">
        <details className="relative min-w-0">
          <summary
            data-testid="components-editor-split-pane-switcher"
            data-pane={pane}
            className="flex cursor-pointer list-none items-center gap-space-2xs rounded-control px-space-3xs py-space-3xs text-xs font-semibold text-app-foreground hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
          >
            {icon}
            <span className="truncate">{label}</span>
          </summary>
          <div className="absolute left-0 z-20 mt-space-3xs w-field rounded-control border border-app-border bg-app-surface p-space-3xs shadow-lg">
            {availablePanes.map((candidate) => (
              <Button
                key={candidate}
                type="button"
                variant="secondary"
                className="h-control-tight w-full justify-start px-space-2xs text-xs"
                onClick={() => selectSplitPane(index, candidate)}
              >
                {paneLabels[candidate]}
              </Button>
            ))}
          </div>
        </details>
      </header>
    );
  };

  // Do not mount a fabricated fallback while the indexed story contract is
  // still loading. That transient iframe is aborted when the real story
  // arrives and can leave the host waiting forever for a readiness message on
  // a frame that no longer exists. A fallback is valid only after a settled,
  // genuinely empty contract has been observed.
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
  const activeStoryContract: ComponentStory | undefined = (storiesQuery.data?.stories ?? []).find(
    (story) => story.version === (activeExample?.version || activeVersion),
  );
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
  const paneLabels: Record<WorkspacePane, string> = {
    files: t(strings.components.editor.files),
    preview: t(strings.components.editor.previewMode),
    details: t(strings.components.editor.info),
  };
  const applyPropsOverride = (
    props: Record<string, unknown>,
    environment?: Record<string, string>,
  ) => {
    if (!activeSpecimen) return;
    const example = specimens.find((candidate) => specimenIdentity(candidate) === activeSpecimen);
    setSpecimenOverrides((current) => ({ ...current, [activeSpecimen]: props }));
    setOverrideStatus((current) => ({ ...current, [activeSpecimen]: "applying" }));
    setOverrideMessages((current) => ({ ...current, [activeSpecimen]: "" }));
    postToSpecimen(activeSpecimen, {
      type: "rcl-preview-props-override",
      componentId: id,
      story: example?.storyId || "",
      version: example?.version || "",
      props,
      environment: environment ?? example?.environment ?? {},
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
      story: example?.storyId || "",
      version: example?.version || "",
    });
  };
  const togglePreviewTools = () => {
    setPreviewToolsCollapsed((collapsed) => !collapsed);
  };

  const editorToolProps = {
    activeSpecimen,
    activeExample,
    activeSpecimenLabel,
    storyContract: activeStoryContract,
    inspector,
    overrideStatus,
    specimenOverrides,
    overrideMessages,
    previewEvents,
    previewDiagnostics,
    onApply: applyPropsOverride,
    onReset: resetPropsOverride,
    onClearEvents: () => setPreviewEvents([]),
  };

  const togglePreviewFullscreen = async () => {
    const stage = previewStageRef.current;
    if (!stage) return;
    if (previewFullscreen) {
      if (document.fullscreenElement === stage && typeof document.exitFullscreen === "function")
        await document.exitFullscreen();
      setPreviewFullscreen(false);
      return;
    }
    try {
      if (typeof stage.requestFullscreen === "function") {
        await stage.requestFullscreen();
      }
    } catch {
      // A browser or embedded host may deny native fullscreen. The fixed
      // fallback still gives the operator a useful, viewport-sized preview.
    }
    setPreviewFullscreen(true);
  };

  return (
    <AssetWorkspace
      testId={selectors.components.editor.panel}
      label={t(strings.components.editor.title, { libraryId })}
    >
      {!chromeless && (
        <WorkspaceHeader
          title={<span data-testid={selectors.components.editor.title}>{libraryId}</span>}
          description={t(strings.components.editor.subtitle)}
          leading={
            shellNavigation.sidebarCollapsed ? (
              <button
                type="button"
                onClick={shellNavigation.openSidebar}
                aria-label={t("nav.openDrawer", { defaultValue: "Open navigation" })}
                data-testid="workspace-header-open-sidebar"
                className="touch-target inline-flex items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
              >
                <Menu aria-hidden className="h-icon-md w-icon-md" />
              </button>
            ) : undefined
          }
          actions={
            <div className="flex items-center gap-space-2xs">
              {dirty && (
                <StatusBadge data-testid={selectors.components.editor.dirtyBadge} tone="warning">
                  {t(strings.components.editor.dirty)}
                </StatusBadge>
              )}
              {previewReady && (desktopLayout || currentPane === "preview") && (
                <StatusBadge data-testid={selectors.components.editor.previewBadge} tone="success">
                  {t(strings.components.editor.previewReady)}
                </StatusBadge>
              )}
              {availablePanes.length > 1 && (
                <IconButton
                  data-testid="components-editor-split-view-toggle"
                  aria-label={t("components.editor.splitView", { defaultValue: "Split view" })}
                  aria-pressed={splitView}
                  onClick={toggleSplitView}
                  className={`hidden h-control-tight min-h-control-tight min-w-control-tight lg:inline-flex ${splitView ? "bg-app-primary text-app-primary-foreground" : "border border-app-border bg-app-surface"}`}
                >
                  <PanelsLeftRight aria-hidden className="h-icon-compact w-icon-compact" />
                </IconButton>
              )}
              {renderable && (
                <IconButton
                  data-testid={selectors.components.editor.closeButton}
                  aria-label={t(strings.components.editor.close)}
                  onClick={onClose}
                  className="h-touch min-h-touch min-w-touch border border-app-border bg-app-surface"
                >
                  <ArrowLeft aria-hidden className="h-icon-compact w-icon-compact" />
                </IconButton>
              )}
            </div>
          }
        />
      )}

      {!chromeless && navigationSlot && (
        <div className="shrink-0 border-b border-app-border bg-app-surface px-space-sm">
          {navigationSlot}
        </div>
      )}

      {contentQuery.isLoading && <SourceLoadingSkeleton />}

      {contentQuery.error && (
        <p data-testid={selectors.components.editor.error} className="p-space-sm text-app-danger">
          {errorMessage(contentQuery.error, t)}
        </p>
      )}

      {saveMutation.error && (
        <p data-testid={selectors.components.editor.error} className="p-space-sm text-app-danger">
          {errorMessage(saveMutation.error, t)}
        </p>
      )}

      {(storiesQuery.data?.warnings?.length ?? 0) > 0 && (
        <aside
          role="status"
          aria-label="Story contract warnings"
          className="border-b border-app-warning/30 bg-app-warning/10 px-space-sm py-space-xs text-xs text-app-warning"
        >
          <p className="font-medium">
            {storiesQuery.data?.warnings?.length} story contract warning(s); indexing remains available.
          </p>
          <ul className="mt-space-2xs list-disc space-y-space-3xs pl-space-md">
            {storiesQuery.data?.warnings?.map((warning) => (
              <li key={warning}>{warning}</li>
            ))}
          </ul>
        </aside>
      )}

      {contentQuery.data && (
        <div className="min-h-0 flex-1">
          {selectedVersion && (
            <p className="border-b border-app-border bg-app-warning/10 px-space-sm py-space-2xs text-xs text-app-warning">
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
                      <ComponentEditorSource
                        id={id}
                        libraryId={libraryId}
                        renderable={renderable}
                        splitView={splitView}
                        activeVersionFiles={activeVersionFiles}
                        selectedFile={selectedFile}
                        selectedTemplate={selectedTemplate}
                        filesView={filesView}
                        comparison={comparison}
                        buffer={buffer}
                        appResolvedTheme={appResolvedTheme}
                        fontSize={fontSize}
                        wordWrap={wordWrap}
                        readOnly={readOnly}
                        dirty={dirty}
                        savePending={saveMutation.isPending}
                        contentLoading={contentQuery.isLoading}
                        handleBeforeMount={handleBeforeMount}
                        handleMount={handleMount}
                        onSelectFile={selectFile}
                        onSelectTemplate={setSelectedTemplate}
                        onFilesViewChange={setFilesView}
                        onBufferChange={setBuffer}
                        onSave={() => saveMutation.mutate()}
                        onRevert={() => setBuffer(contentQuery.data.content)}
                        onToggleWordWrap={() =>
                          setWordWrap((current) => (current === "on" ? "off" : "on"))
                        }
                        onDecreaseFont={() => setFontSize((current) => Math.max(11, current - 1))}
                        onIncreaseFont={() => setFontSize((current) => Math.min(20, current + 1))}
                        onCloseComparison={() => {
                          onCloseComparison?.();
                          setFilesView("source");
                        }}
                      />
                    )}
                    {pane === "preview" && (
                      <div
                        ref={previewStageRef}
                        data-testid={selectors.components.editor.previewStage}
                        data-preview-fullscreen={previewFullscreen ? "true" : "false"}
                        className={
                          previewFullscreen
                            ? "fixed inset-0 z-50 h-full w-full bg-app-background"
                            : "h-full min-h-0"
                        }
                      >
                        <ExperienceSurface
                          surfaceId="component-preview"
                          state={previewExperienceState}
                          data-testid={selectors.components.editor.workspacePane}
                          data-preview-state={previewExperienceState}
                          data-pane="preview"
                          className="relative flex h-full min-h-0 min-w-0 w-full max-w-full flex-col overflow-hidden bg-app-background"
                        >
                          {splitView &&
                            paneHeader(
                              "preview",
                              index,
                              paneLabels.preview,
                              <Eye aria-hidden className="h-icon-compact w-icon-compact" />,
                            )}
                          <div
                            data-preview-floating-dock
                            role="toolbar"
                            aria-label="Preview controls"
                            className="absolute left-space-xs right-space-xs top-space-xs z-20 flex min-w-0 max-w-inline-overlay flex-wrap items-center gap-space-2xs rounded-panel border border-app-border/80 bg-app-surface/95 px-space-2xs py-space-2xs shadow-xl backdrop-blur"
                          >
                            {stageMode && (
                              <nav
                                className="order-last min-w-0 max-w-full basis-full overflow-x-auto border-t border-app-border/70 pt-space-2xs"
                                aria-label={t(strings.components.editor.storiesLabel)}
                              >
                                <div className="flex w-max gap-space-3xs">
                                  {specimens.map((example) => {
                                    const identity = specimenIdentity(example);
                                    const selected = identity === activeSpecimen;
                                    return (
                                      <Button
                                        key={identity}
                                        data-testid={selectors.components.editor.storyPickerItem}
                                        type="button"
                                        variant={selected ? "primary" : "secondary"}
                                        className="h-control-tight min-w-touch shrink-0 px-space-2xs text-xs"
                                        aria-current={selected ? "true" : undefined}
                                        title={example?.description}
                                        onClick={() => activateSpecimen(identity)}
                                      >
                                        {example?.displayName || example?.name || "Default"}
                                      </Button>
                                    );
                                  })}
                                </div>
                              </nav>
                            )}
                            {framePickerEnabled && (
                              <div className="flex min-w-0 items-center gap-space-3xs">
                                <label className="flex h-control items-center gap-space-3xs rounded-control border border-app-border/80 bg-app-surface px-space-2xs text-xs text-app-muted-foreground">
                                  <span className="sr-only">Preview frame</span>
                                  <select
                                    aria-label="Preview frame"
                                    data-testid="components-editor-frame-picker"
                                    className="h-full min-h-touch max-w-[12rem] bg-transparent text-app-foreground outline-none"
                                    value={frameOverride ? frameOverride.asset : ""}
                                    disabled={
                                      frameCandidatesQuery.isPending || frameCandidatesQuery.isError
                                    }
                                    onChange={(event) => {
                                      const next = compatibleFrameCandidates.find(
                                        (candidate) => candidate.asset === event.target.value,
                                      );
                                      setFrameSaveMessage("");
                                      setFrameOverride(next);
                                    }}
                                  >
                                    <option value="">
                                      {frameCandidatesQuery.isPending
                                        ? "Loading frames…"
                                        : frameCandidatesQuery.isError
                                          ? "Frames unavailable"
                                          : compatibleFrameCandidates.length === 0
                                            ? "No compatible frames"
                                            : "Story frame"}
                                    </option>
                                    {compatibleFrameCandidates.map((candidate) => (
                                      <option
                                        key={`${candidate.asset}:${candidate.region}`}
                                        value={candidate.asset}
                                      >
                                        {candidate.label} · {candidate.region}
                                      </option>
                                    ))}
                                  </select>
                                </label>
                                {frameOverride && (
                                  <Button
                                    type="button"
                                    variant="secondary"
                                    className="h-control-tight px-space-2xs text-xs"
                                    data-testid="components-editor-frame-save"
                                    disabled={persistFrameMutation.isPending}
                                    onClick={() => persistFrameMutation.mutate()}
                                  >
                                    {persistFrameMutation.isPending ? "Saving…" : "Save frame"}
                                  </Button>
                                )}
                                {frameSaveMessage && (
                                  <span
                                    role={persistFrameMutation.isError ? "alert" : "status"}
                                    className="max-w-[14rem] truncate text-xs text-app-muted-foreground"
                                    title={frameSaveMessage}
                                  >
                                    {frameSaveMessage}
                                  </span>
                                )}
                              </div>
                            )}
                            <IconButton
                              data-testid={selectors.components.editor.previewStageFullscreen}
                              type="button"
                              aria-label={
                                previewFullscreen
                                  ? t("components.editor.exitFullscreen", {
                                      defaultValue: "Exit full screen",
                                    })
                                  : t("components.editor.enterFullscreen", {
                                      defaultValue: "View preview full screen",
                                    })
                              }
                              aria-pressed={previewFullscreen}
                              onClick={() => {
                                void togglePreviewFullscreen();
                              }}
                              className="h-touch min-h-touch min-w-touch border border-app-border bg-app-surface"
                            >
                              {previewFullscreen ? (
                                <Minimize2 aria-hidden className="h-icon-compact w-icon-compact" />
                              ) : (
                                <Maximize2 aria-hidden className="h-icon-compact w-icon-compact" />
                              )}
                            </IconButton>
                            <ThemeSwitcher
                              previewReady={previewReady}
                              colorScheme={filters.colorScheme}
                              setColorScheme={filters.setColorScheme}
                              kit={previewKit}
                              setKit={setPreviewKit}
                              filters={filters}
                              compactOnMobile
                            />
                            <Button
                              type="button"
                              variant={stageMode ? "primary" : "secondary"}
                              className="h-control-tight px-space-2xs text-xs"
                              aria-pressed={stageMode}
                              data-testid="components-editor-stage-mode"
                              onClick={() => {
                                setStageMode((enabled) => !enabled);
                                setPreviewToolsCollapsed(true);
                                setComparedSpecimens(new Set());
                              }}
                            >
                              {stageMode ? "Focus" : "Canvas"}
                            </Button>
                            {!stageMode && specimens.length > 1 && (
                              <Button
                                type="button"
                                variant={comparisonActive ? "primary" : "secondary"}
                                className="h-control-tight px-space-2xs text-xs"
                                data-testid={selectors.components.editor.storySheetAll}
                                aria-pressed={comparisonActive}
                                onClick={selectAllComparison}
                              >
                                Story sheet ({Math.min(specimens.length, 4)})
                              </Button>
                            )}
                            <EmulatorToolbar emulator={emulator} compactOnMobile />
                            {activeSpecimen && (
                              <IconButton
                                data-testid={selectors.components.editor.previewToolsToggle}
                                aria-label={
                                  previewToolsCollapsed
                                    ? t("components.editor.showTools", {
                                        defaultValue: "Show controls",
                                      })
                                    : t("components.editor.hideTools", {
                                        defaultValue: "Hide controls",
                                      })
                                }
                                className="ml-auto h-touch min-h-touch min-w-touch border border-app-border bg-app-surface"
                                aria-expanded={!previewToolsCollapsed}
                                aria-controls="component-preview-tools"
                                onClick={togglePreviewTools}
                              >
                                <SlidersHorizontal
                                  aria-hidden
                                  className="h-icon-compact w-icon-compact"
                                />
                              </IconButton>
                            )}
                          </div>
                          <ComponentEditorStage
                            emulator={emulator}
                            filters={filters}
                            stageMode={stageMode}
                            comparisonActive={comparisonActive}
                            specimens={specimens}
                            visibleSpecimens={visibleSpecimens}
                            readyExamples={readyExamples}
                            previewMessage={previewMessage}
                            specimenErrors={specimenErrors}
                            specimenRetries={specimenRetries}
                            comparedSpecimens={comparedSpecimens}
                            activeSpecimen={activeSpecimen}
                            previewKit={previewKit}
                            frameEnabled={frameEnabled}
                            frameOverride={frameOverride}
                            id={id}
                            baselineSha={baselineSha}
                            previewReloadKey={previewReloadKey}
                            selectedVersion={selectedVersion}
                            resolvedPreviewTheme={resolvedPreviewTheme}
                            previewCanvasRef={previewCanvasRef}
                            toolsDocked={desktopLayout}
                            toolsOpen={desktopLayout && !previewToolsCollapsed}
                            onClearComparison={() => setComparedSpecimens(new Set())}
                            onSelectAllComparison={selectAllComparison}
                            onToggleComparison={toggleComparison}
                            onRetrySpecimen={retrySpecimen}
                            onRegisterPreviewFrame={registerPreviewFrame}
                            onPreviewLoad={(identity) => {
                              setActiveSpecimen((current) => current ?? identity);
                            }}
                            onPreviewError={(identity) =>
                              setSpecimenErrors((current) => ({
                                ...current,
                                [identity]: t(strings.components.editor.previewFailed),
                              }))
                            }
                            postToPreviewFrames={postToPreviewFrames}
                            onCloseTools={() => setPreviewToolsCollapsed(true)}
                            onEnterSpecimen={(identity) => {
                              setActiveSpecimen(identity);
                              setStageMode(true);
                              setComparedSpecimens(new Set());
                            }}
                            tools={<ComponentEditorTools {...editorToolProps} />}
                          />
                          {!desktopLayout && (
                            <ComponentEditorMobileTools
                              tool={previewToolsCollapsed ? null : "props"}
                              onClose={() => setPreviewToolsCollapsed(true)}
                              props={editorToolProps}
                            />
                          )}
                        </ExperienceSurface>
                      </div>
                    )}
                    {pane === "details" && (
                      <aside
                        data-testid={selectors.components.editor.workspacePane}
                        data-pane="details"
                        className="flex h-full min-h-0 flex-col overflow-hidden bg-app-surface"
                      >
                        {splitView &&
                          paneHeader(
                            "details",
                            index,
                            paneLabels.details,
                            <Info aria-hidden className="h-icon-compact w-icon-compact" />,
                          )}
                        <div
                          data-testid={selectors.components.editor.infoDialog}
                          className="min-h-0 flex-1 overflow-auto p-space-sm"
                        >
                          {metadataSlot ?? (
                            <p className="text-sm text-app-muted-foreground">
                              {t(strings.components.editor.noInfo)}
                            </p>
                          )}
                        </div>
                      </aside>
                    )}
                  </div>
                </Panel>
                {index < visiblePanes.length - 1 && (
                  <Separator className="w-separator shrink-0 bg-app-border hover:bg-app-primary" />
                )}
              </Fragment>
            ))}
          </Group>
        </div>
      )}

      {showSaved && (
        <p
          data-testid={selectors.components.editor.savedToast}
          className="absolute bottom-3 right-3 rounded-md bg-app-success/10 px-space-xs py-space-2xs text-xs text-app-success shadow-lg"
        >
          {t(strings.components.editor.saved)}
        </p>
      )}
    </AssetWorkspace>
  );
}
