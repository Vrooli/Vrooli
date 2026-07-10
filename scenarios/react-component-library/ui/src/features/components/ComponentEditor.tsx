import { type ReactNode, useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Editor, { type Monaco } from "@monaco-editor/react";
import type { editor } from "monaco-editor";
import { ArrowLeft, Code2, Eye, Info, Save } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Dialog } from "../../components/ui/dialog";
import { StatusBadge } from "../../components/ui/status-badge";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { API_BASE } from "../../api/client";
import { componentsClient, listComponentExamples, type ComponentExample } from "../../api/components";
import { useComponentInspector } from "../../hooks/useComponentInspector";
import { useDeviceEmulation } from "../../hooks/useDeviceEmulation";
import { useDeviceFilters } from "../../hooks/useDeviceFilters";
import { errorMessage } from "../../lib/errorMessage";
import { EmulatorToolbar, EmulatorViewport } from "./EmulatorChrome";
import { InspectorPanel } from "./InspectorPanel";
import { ThemeSwitcher } from "./ThemeSwitcher";

const PREVIEW_LOAD_TIMEOUT_MS = 8_000;

/**
 * Build the harness URL the iframe loads. The query param `v` is the
 * latest content sha256 — when the sha changes (i.e. after a save),
 * React's `src` diff forces the iframe to reload. The harness route is
 * served by the API at the same origin as the Connect transport.
 */
function harnessUrl(id: string, version: string, reloadKey = 0, example?: ComponentExample): string {
  const base = API_BASE.replace(/\/$/, "");
  const v = encodeURIComponent(version || "initial");
  const url = new URL(`${base}/preview/${encodeURIComponent(id)}/harness.html`);
  url.searchParams.set("v", v);
  url.searchParams.set("r", String(reloadKey));
  if (example) {
    url.searchParams.set("example", example.name);
    url.searchParams.set("version", example.version);
  }
  return url.toString();
}

interface ComponentEditorProps {
  id: string;
  libraryId: string;
  onClose: () => void;
  metadataSlot?: ReactNode;
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
export function ComponentEditor({ id, libraryId, onClose, metadataSlot }: ComponentEditorProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const emulator = useDeviceEmulation();
  const filters = useDeviceFilters();
  const previewFrameRef = useRef<HTMLIFrameElement | null>(null);
  const inspector = useComponentInspector(previewFrameRef);

  const contentQuery = useQuery({
    queryKey: ["components", "content", id],
    queryFn: () => componentsClient.getComponentContent({ id }),
  });

  const examplesQuery = useQuery({
    queryKey: ["components", "examples", id],
    queryFn: () => listComponentExamples({ componentId: id, limit: 200 }),
  });

  const [buffer, setBuffer] = useState<string>("");
  const [baselineSha, setBaselineSha] = useState<string>("");
  const [showSaved, setShowSaved] = useState(false);
  const [previewState, setPreviewState] = useState<"waiting" | "ready" | "error">("waiting");
  const [previewMessage, setPreviewMessage] = useState("");
  const [readyExamples, setReadyExamples] = useState<ReadonlySet<string>>(() => new Set());
  const [previewReloadKey, setPreviewReloadKey] = useState(0);
  const [mode, setMode] = useState<"preview" | "code">("code");
  const [infoOpen, setInfoOpen] = useState(false);
  const savedToastTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const previewTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const previewReady = previewState === "ready";

  useEffect(() => {
    const handler = (ev: MessageEvent) => {
      const data = ev.data as { type?: string; id?: string; message?: string; example?: string } | null;
      if (data && data.type === "preview-ready" && data.id === id) {
        setReadyExamples((current) => {
          const next = new Set(current);
          next.add(data.example || "__default__");
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

  // Push color-scheme into the harness whenever it changes, plus once
  // on preview-ready so a reload (new baselineSha) re-applies it.
  useEffect(() => {
    if (!previewReady) return;
    const win = previewFrameRef.current?.contentWindow;
    if (!win) return;
    win.postMessage(
      { type: "rcl-color-scheme", colorScheme: filters.colorScheme },
      "*",
    );
  }, [filters.colorScheme, previewReady]);

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

  const dirty = !!contentQuery.data && buffer !== contentQuery.data.content;

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
              {previewReady && mode === "preview" && (
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
              data-testid={selectors.components.editor.infoButton}
              type="button"
              variant="secondary"
              onClick={() => setInfoOpen(true)}
              aria-label={t(strings.components.editor.info)}
              title={t(strings.components.editor.info)}
              className="h-8 w-8 rounded-md p-0"
            >
              <Info aria-hidden className="h-4 w-4" />
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
            <Button
              data-testid={selectors.components.editor.saveButton}
              onClick={() => saveMutation.mutate()}
              disabled={!dirty || saveMutation.isPending || contentQuery.isLoading}
              className="h-8 gap-1.5 rounded-md px-2 text-xs"
            >
              <Save aria-hidden className="h-3.5 w-3.5" />
              {saveMutation.isPending
                ? t(strings.components.editor.saving)
                : t(strings.components.editor.save)}
            </Button>
          </div>
        </div>

        <div className="flex min-w-0 flex-wrap items-center gap-2 border-t border-app-border">
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
          </div>
          {mode === "preview" && (
            <ThemeSwitcher
              frameRef={previewFrameRef}
              previewReady={previewReady}
            />
          )}
          <EmulatorToolbar
            emulator={emulator}
            filters={filters}
          />
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
          <div
            data-testid={selectors.components.editor.surface}
            className={`h-full min-h-0 overflow-hidden ${mode === "code" ? "" : "hidden"}`}
          >
            <Editor
              height="100%"
              language="typescript"
              path={`${libraryId || id}.tsx`}
              value={buffer}
              onChange={(v) => setBuffer(v ?? "")}
              beforeMount={handleBeforeMount}
              onMount={handleMount}
              theme="vs-dark"
              options={{
                fontSize: 13,
                lineNumbers: "on",
                minimap: { enabled: false },
                scrollBeyondLastLine: false,
                tabSize: 2,
                insertSpaces: true,
                wordWrap: "on",
                automaticLayout: true,
              }}
            />
          </div>
          <div
            data-testid={selectors.components.editor.preview}
            className={`h-full min-h-0 overflow-hidden bg-app-background ${mode === "preview" ? "" : "hidden"}`}
          >
            <div className="relative h-full">
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
                      {examples.map((example, index) => (
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
                            src={harnessUrl(id, baselineSha, previewReloadKey, example)}
                            sandbox="allow-scripts allow-same-origin"
                            ref={index === 0 ? previewFrameRef : undefined}
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
                      src={harnessUrl(id, baselineSha, previewReloadKey)}
                      sandbox="allow-scripts allow-same-origin"
                      ref={previewFrameRef}
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
      {infoOpen && (
        <Dialog
          open={infoOpen}
          title={libraryId}
          description={baselineSha ? t(strings.components.editor.sha, { sha: baselineSha }) : undefined}
          onClose={() => setInfoOpen(false)}
          closeLabel={t(strings.components.editor.closeInfo)}
          className="max-w-md"
        >
          <div data-testid={selectors.components.editor.infoDialog}>
            {metadataSlot ?? (
              <p className="text-sm text-app-muted-foreground">
                {t(strings.components.editor.noInfo)}
              </p>
            )}
          </div>
        </Dialog>
      )}
    </section>
  );
}
