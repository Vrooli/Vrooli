import { type ReactNode, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Editor, { type Monaco } from "@monaco-editor/react";
import type { editor } from "monaco-editor";
import { ArrowLeft, Code2, Eye, Info, Save } from "lucide-react";
import { Group, Panel, Separator, usePanelRef } from "react-resizable-panels";

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
import { AdoptionFileTree } from "./AdoptionFileTree";
import { ADOPTION_TEMPLATES, DEFAULT_ADOPTION_TEMPLATE } from "./adoptionTemplates";

const PREVIEW_LOAD_TIMEOUT_MS = 8_000;
const PANEL_LAYOUT_STORAGE_KEY = "rcl.component-editor.desktop-layout.v1";
const DEFAULT_DESKTOP_PANEL_LAYOUT = { preview: 42, code: 38, info: 20 };

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
export function ComponentEditor({ id, libraryId, onClose, metadataSlot, selectedVersion }: ComponentEditorProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const emulator = useDeviceEmulation();
  const filters = useDeviceFilters();
  const desktopLayout = useMediaQuery("(min-width: 1024px)");
  const { resolved: appResolvedTheme } = useTheme();
  const previewFrameRef = useRef<HTMLIFrameElement | null>(null);
  const previewFramesRef = useRef(new Set<HTMLIFrameElement>());
  const codePanelRef = usePanelRef();
  const infoPanelRef = usePanelRef();
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
  const [previewReloadKey, setPreviewReloadKey] = useState(0);
  const [mode, setMode] = useState<"preview" | "code" | "info">("code");
  const [codeCollapsed, setCodeCollapsed] = useState(false);
  const [infoCollapsed, setInfoCollapsed] = useState(false);
  const initialDesktopLayout = useMemo(() => {
    try {
      const saved = window.localStorage.getItem(PANEL_LAYOUT_STORAGE_KEY);
      if (!saved) return DEFAULT_DESKTOP_PANEL_LAYOUT;
      const parsed = JSON.parse(saved) as Record<string, unknown>;
      if (["preview", "code", "info"].every((key) => typeof parsed[key] === "number")) {
        return {
          preview: parsed.preview as number,
          code: parsed.code as number,
          info: parsed.info as number,
        };
      }
    } catch {
      // A stale or unavailable browser storage entry must never prevent editing.
    }
    return DEFAULT_DESKTOP_PANEL_LAYOUT;
  }, []);
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

  const registerPreviewFrame = useCallback((frame: HTMLIFrameElement | null) => {
    if (!frame) return;
    previewFramesRef.current.add(frame);
    if (!previewFrameRef.current) previewFrameRef.current = frame;
  }, []);

  useEffect(() => {
    const handler = (ev: MessageEvent) => {
      const data = ev.data as { type?: string; id?: string; message?: string; example?: string; version?: string } | null;
      if (data && data.type === "preview-ready" && data.id === id) {
        setReadyExamples((current) => {
          const next = new Set(current);
          next.add(`${data.version || "__current__"}:${data.example || "__default__"}`);
          return next;
        });
        setPreviewMessage("");
      } else if (data && data.type === "preview-error" && data.id === id) {
        setPreviewState("error");
        setPreviewMessage(data.message || t(strings.components.editor.previewFailed));
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
  }, [baselineSha, previewReloadKey]);

  const examples = examplesQuery.data?.examples ?? [];
  const expectedReadyCount = Math.max(1, examples.length);

  useEffect(() => {
    if (previewState !== "waiting") return;
    if (readyExamples.size >= expectedReadyCount) {
      setPreviewState("ready");
    }
  }, [expectedReadyCount, previewState, readyExamples]);

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

  const handlePreviewRetry = () => {
    setPreviewState("waiting");
    setPreviewMessage("");
    setReadyExamples(new Set());
    setPreviewReloadKey((current) => current + 1);
  };

  const saveDesktopPanelLayout = (layout: Record<string, number>) => {
    if (!desktopLayout) return;
    try {
      window.localStorage.setItem(PANEL_LAYOUT_STORAGE_KEY, JSON.stringify(layout));
    } catch {
      // Layout persistence is a convenience; keep resizing usable if storage is blocked.
    }
  };

  const toggleDesktopPanel = (
    panel: typeof codePanelRef,
    setCollapsed: (collapsed: boolean) => void,
  ) => {
    if (!desktopLayout || !panel.current) return;
    if (panel.current.isCollapsed()) {
      panel.current.expand();
      setCollapsed(false);
      return;
    }
    panel.current.collapse();
    setCollapsed(true);
  };

  return (
    <section
      data-testid={selectors.components.editor.panel}
      aria-label={t(strings.components.editor.title, { libraryId })}
      className="relative flex min-h-0 flex-1 flex-col overflow-hidden bg-app-background"
    >
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
            <Button
              type="button"
              variant="secondary"
              data-testid="component-editor-toggle-code-panel"
              aria-pressed={codeCollapsed}
              onClick={() => toggleDesktopPanel(codePanelRef, setCodeCollapsed)}
              className="hidden h-8 px-2 text-xs lg:inline-flex"
            >
              {codeCollapsed
                ? t("components.editor.showCode", { defaultValue: "Show code" })
                : t("components.editor.hideCode", { defaultValue: "Hide code" })}
            </Button>
            <Button
              type="button"
              variant="secondary"
              data-testid="component-editor-toggle-info-panel"
              aria-pressed={infoCollapsed}
              onClick={() => toggleDesktopPanel(infoPanelRef, setInfoCollapsed)}
              className="hidden h-8 px-2 text-xs lg:inline-flex"
            >
              {infoCollapsed
                ? t("components.editor.showInfo", { defaultValue: "Show info" })
                : t("components.editor.hideInfo", { defaultValue: "Hide info" })}
            </Button>
            <Button
              data-testid={selectors.components.editor.closeButton}
              onClick={onClose}
              className="h-8 gap-1.5 rounded-md px-2 text-xs"
              variant="secondary"
            >
              <ArrowLeft aria-hidden className="h-3.5 w-3.5" />
              {t(strings.components.editor.close)}
            </Button>
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
              onClick={() => setMode("preview")}
            >
              <Eye aria-hidden className="h-3.5 w-3.5" />
              {t(strings.components.editor.previewMode)}
            </Button>
            <Button
              data-testid={selectors.components.editor.codeModeButton}
              type="button"
              variant={mode === "code" ? "primary" : "secondary"}
              className="h-7 gap-1.5 rounded-none px-2 text-xs"
              onClick={() => setMode("code")}
            >
              <Code2 aria-hidden className="h-3.5 w-3.5" />
              {t(strings.components.editor.codeMode)}
            </Button>
            <Button
              type="button"
              variant={mode === "info" ? "primary" : "secondary"}
              className="h-7 gap-1.5 rounded-none px-2 text-xs lg:hidden"
              onClick={() => setMode("info")}
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
          <Group
            id="component-editor-panels"
            orientation={desktopLayout ? "horizontal" : "vertical"}
            defaultLayout={desktopLayout ? initialDesktopLayout : { [mode]: 100 }}
            onLayoutChanged={saveDesktopPanelLayout}
            className="h-full min-h-0"
          >
          <Panel id="code" minSize="20%" defaultSize="38%" collapsible collapsedSize="0%" panelRef={codePanelRef} className={mode === "code" ? "" : "max-lg:hidden"}>
          <div className="flex h-full min-h-0 flex-col">
            <div className="flex shrink-0 items-start justify-between gap-2 border-b border-app-border bg-app-surface px-2 py-1.5">
              <div className="min-w-0 flex-1 overflow-y-auto" style={{ maxHeight: "10rem" }}>
                <AdoptionFileTree
                  componentId={id}
                  version={selectedVersion}
                  files={activeVersionFiles}
                  selectedFile={selectedFile}
                  onSelectFile={setSelectedFile}
                  template={selectedTemplate}
                  templates={ADOPTION_TEMPLATES}
                  onSelectTemplate={setSelectedTemplate}
                />
              </div>
              <Button
                data-testid={selectors.components.editor.saveButton}
                onClick={() => saveMutation.mutate()}
                disabled={readOnly || !dirty || saveMutation.isPending || contentQuery.isLoading}
                className="h-7 shrink-0 gap-1.5 rounded-md px-2 text-xs"
              >
                <Save aria-hidden className="h-3.5 w-3.5" />
                {saveMutation.isPending
                  ? t(strings.components.editor.saving)
                  : t(strings.components.editor.save)}
              </Button>
            </div>
          <div
            data-testid={selectors.components.editor.surface}
            className="min-h-0 flex-1 overflow-hidden"
          >
            <Editor
              height="100%"
              language="typescript"
              path={selectedFile || `${libraryId || id}.tsx`}
              value={buffer}
              onChange={(v) => setBuffer(v ?? "")}
              beforeMount={handleBeforeMount}
              onMount={handleMount}
              theme={appResolvedTheme === "dark" ? "vs-dark" : "vs"}
              options={{
                fontSize: 13,
                lineNumbers: "on",
                minimap: { enabled: false },
                scrollBeyondLastLine: false,
                tabSize: 2,
                insertSpaces: true,
                wordWrap: "on",
                automaticLayout: true,
                readOnly,
              }}
          />
          </div>
          </div>
          </Panel>
          <Separator className="hidden w-1 shrink-0 bg-app-border hover:bg-app-primary lg:block" />
          <Panel id="preview" minSize="20%" defaultSize="42%" className={mode === "preview" ? "" : "max-lg:hidden"}>
          <div
            data-testid={selectors.components.editor.preview}
            className="flex h-full min-h-0 flex-col overflow-hidden bg-app-background"
          >
            <div className="flex shrink-0 flex-col gap-1.5 border-b border-app-border bg-app-surface px-2 py-1.5">
              <ThemeSwitcher
                postToFrames={postToPreviewFrames}
                previewReady={previewReady}
                colorScheme={filters.colorScheme}
                setColorScheme={filters.setColorScheme}
              />
              <EmulatorToolbar emulator={emulator} filters={filters} />
            </div>
            <div className="relative min-h-0 flex-1">
              <EmulatorViewport emulator={emulator} filters={filters}>
                <div
                  data-testid={selectors.components.editor.gallery}
                  className="h-full overflow-auto bg-app-background p-3"
                  style={{
                    width: emulator.displayWidth,
                    height: emulator.displayHeight,
                  }}
                >
                  {examples.length > 0 ? (
                    <div className="grid min-w-0 gap-3">
                      {examples.map((example) => (
                        <section
                          key={`${example.version}:${example.name}`}
                          data-testid={selectors.components.editor.exampleCard}
                          className="min-w-0 rounded-md border border-app-border bg-app-surface"
                        >
                          <header className="border-b border-app-border px-3 py-2">
                            <h3
                              data-testid={selectors.components.editor.exampleTitle}
                              className="truncate text-sm font-semibold text-app-foreground"
                            >
                              {example.displayName || example.name}
                            </h3>
                          </header>
                          <iframe
                            data-testid={selectors.components.editor.previewFrame}
                            title={`${t(strings.components.editor.previewHeading)} - ${example.displayName || example.name}`}
                            src={harnessUrl(id, baselineSha, previewReloadKey, example, selectedVersion)}
                            sandbox="allow-scripts allow-same-origin"
                            ref={registerPreviewFrame}
                            onLoad={() => postToPreviewFrames({ type: "rcl-resolved-theme", theme: resolvedPreviewTheme })}
                            onError={() => {
                              setPreviewState("error");
                              setPreviewMessage(t(strings.components.editor.previewFailed));
                            }}
                            className="block h-[260px] w-full border-0 bg-white"
                          />
                        </section>
                      ))}
                    </div>
                  ) : (
                    <iframe
                      data-testid={selectors.components.editor.previewFrame}
                      title={t(strings.components.editor.previewHeading)}
                      src={harnessUrl(id, baselineSha, previewReloadKey, undefined, selectedVersion)}
                      sandbox="allow-scripts allow-same-origin"
                      ref={registerPreviewFrame}
                      onLoad={() => postToPreviewFrames({ type: "rcl-resolved-theme", theme: resolvedPreviewTheme })}
                      onError={() => {
                        setPreviewState("error");
                        setPreviewMessage(t(strings.components.editor.previewFailed));
                      }}
                      className="block h-full w-full border-0 bg-white"
                    />
                  )}
                </div>
                {previewState === "error" && (
                  <div
                    data-testid={selectors.components.editor.previewError}
                    className="absolute inset-3 flex items-center justify-center bg-app-background/85 p-4 text-center"
                  >
                    <div className="max-w-md rounded-md border border-app-danger/40 bg-app-danger/10 p-4 text-app-danger shadow-xl">
                      <p className="text-sm">{previewMessage || t(strings.components.editor.previewFailed)}</p>
                      <Button
                        type="button"
                        variant="secondary"
                        className="mt-3 h-8 rounded-md px-3 text-xs"
                        data-testid={selectors.components.editor.previewRetryButton}
                        onClick={handlePreviewRetry}
                      >
                        {t(strings.components.editor.previewRetry)}
                      </Button>
                    </div>
                  </div>
                )}
              </EmulatorViewport>
            </div>
            <InspectorPanel inspector={inspector} />
          </div>
          </Panel>
          <Separator className="hidden w-1 shrink-0 bg-app-border hover:bg-app-primary lg:block" />
          <Panel id="info" minSize="15%" defaultSize="20%" collapsible collapsedSize="0%" panelRef={infoPanelRef} className={mode === "info" ? "" : "max-lg:hidden"}>
            <aside data-testid={selectors.components.editor.infoDialog} className="h-full overflow-auto border-l border-app-border bg-app-surface p-4">
              <h3 className="mb-3 text-sm font-semibold text-app-foreground">{t(strings.components.editor.info)}</h3>
              {metadataSlot ?? <p className="text-sm text-app-muted-foreground">{t(strings.components.editor.noInfo)}</p>}
            </aside>
          </Panel>
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
    </section>
  );
}
