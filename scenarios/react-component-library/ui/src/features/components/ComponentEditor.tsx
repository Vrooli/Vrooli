import { type ReactNode, useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Editor, { type Monaco } from "@monaco-editor/react";
import type { editor } from "monaco-editor";
import { ArrowLeft, Code2, Eye, Info, Save, X } from "lucide-react";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { API_BASE } from "../../api/client";
import { componentsClient } from "../../api/components";
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
function harnessUrl(id: string, version: string, reloadKey = 0): string {
  const base = API_BASE.replace(/\/$/, "");
  const v = encodeURIComponent(version || "initial");
  return `${base}/preview/${encodeURIComponent(id)}/harness.html?v=${v}&r=${reloadKey}`;
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

  const [buffer, setBuffer] = useState<string>("");
  const [baselineSha, setBaselineSha] = useState<string>("");
  const [showSaved, setShowSaved] = useState(false);
  const [previewState, setPreviewState] = useState<"waiting" | "ready" | "error">("waiting");
  const [previewMessage, setPreviewMessage] = useState("");
  const [previewReloadKey, setPreviewReloadKey] = useState(0);
  const [mode, setMode] = useState<"preview" | "code">("code");
  const [infoOpen, setInfoOpen] = useState(false);
  const savedToastTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const previewTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const previewReady = previewState === "ready";

  useEffect(() => {
    const handler = (ev: MessageEvent) => {
      const data = ev.data as { type?: string; id?: string; message?: string } | null;
      if (data && data.type === "preview-ready" && data.id === id) {
        setPreviewState("ready");
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
  }, [baselineSha, previewReloadKey]);

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

  const handleMount = (monacoEditor: editor.IStandaloneCodeEditor, monaco: Monaco) => {
    monacoEditor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
      if (!saveMutation.isPending) saveMutation.mutate();
    });
  };

  const handlePreviewRetry = () => {
    setPreviewState("waiting");
    setPreviewMessage("");
    setPreviewReloadKey((current) => current + 1);
  };

  return (
    <section
      data-testid={selectors.components.editor.panel}
      aria-label={t(strings.components.editor.title, { libraryId })}
      className="relative flex min-h-0 flex-1 flex-col overflow-hidden bg-[#05070d]"
    >
      <header className="shrink-0 border-b border-white/10 bg-[#0f172a]">
        <div className="flex min-w-0 items-center justify-between gap-3 px-4 py-2">
          <div className="min-w-0">
            <h2
              data-testid={selectors.components.editor.title}
              className="truncate text-base font-semibold text-slate-100"
            >
              {libraryId}
            </h2>
            <div className="mt-0.5 flex min-w-0 flex-wrap items-center gap-2 text-xs text-slate-400">
              <span className="truncate">{t(strings.components.editor.subtitle)}</span>
              {baselineSha && (
                <span
                  data-testid={selectors.components.editor.shaHash}
                  className="font-mono text-slate-500"
                >
                  {t(strings.components.editor.sha, { sha: baselineSha.slice(0, 12) })}
                </span>
              )}
              {dirty && (
                <span
                  data-testid={selectors.components.editor.dirtyBadge}
                  className="rounded-full bg-amber-500/20 px-2 py-0.5 text-amber-200"
                >
                  {t(strings.components.editor.dirty)}
                </span>
              )}
              {previewReady && mode === "preview" && (
                <span
                  data-testid={selectors.components.editor.previewBadge}
                  className="rounded-full bg-emerald-500/20 px-2 py-0.5 text-emerald-200"
                >
                  {t(strings.components.editor.previewReady)}
                </span>
              )}
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-1.5">
            <Button
              data-testid={selectors.components.editor.infoButton}
              type="button"
              variant="outline"
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
              variant="outline"
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

        <div className="flex min-w-0 flex-wrap items-center gap-2 border-t border-white/10">
          <div
            data-testid={selectors.components.editor.modeSwitch}
            className="flex shrink-0 overflow-hidden rounded-md border border-white/10"
          >
            <Button
              data-testid={selectors.components.editor.previewModeButton}
              type="button"
              variant={mode === "preview" ? "default" : "outline"}
              className="h-7 gap-1.5 rounded-none px-2 text-xs"
              onClick={() => setMode("preview")}
            >
              <Eye aria-hidden className="h-3.5 w-3.5" />
              {t(strings.components.editor.previewMode)}
            </Button>
            <Button
              data-testid={selectors.components.editor.codeModeButton}
              type="button"
              variant={mode === "code" ? "default" : "outline"}
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
          className="p-4 text-slate-200"
        >
          {t(strings.components.editor.loading)}
        </p>
      )}

      {contentQuery.error && (
        <p
          data-testid={selectors.components.editor.error}
          className="p-4 text-red-400"
        >
          {errorMessage(contentQuery.error, t)}
        </p>
      )}

      {saveMutation.error && (
        <p
          data-testid={selectors.components.editor.error}
          className="p-4 text-red-400"
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
            className={`h-full min-h-0 overflow-hidden bg-black ${mode === "preview" ? "" : "hidden"}`}
          >
            <div className="relative h-full">
              <EmulatorViewport emulator={emulator} filters={filters}>
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
                  style={{
                    width: emulator.displayWidth,
                    height: emulator.displayHeight,
                  }}
                  className="block border-0 bg-white"
                />
                {previewState === "error" && (
                  <div
                    data-testid={selectors.components.editor.previewError}
                    className="absolute inset-3 flex items-center justify-center bg-slate-950/85 p-4 text-center"
                  >
                    <div className="max-w-md rounded-md border border-red-400/40 bg-red-950/90 p-4 text-red-100 shadow-xl">
                      <p className="text-sm">{previewMessage || t(strings.components.editor.previewFailed)}</p>
                      <Button
                        type="button"
                        variant="outline"
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
          className="absolute bottom-3 right-3 rounded-md bg-emerald-950/90 px-3 py-2 text-xs text-emerald-200 shadow-lg"
        >
          {t(strings.components.editor.saved)}
        </p>
      )}
      {infoOpen && (
        <div
          data-testid={selectors.components.editor.infoDialog}
          className="fixed inset-0 z-50 flex justify-end bg-black/55"
          role="dialog"
          aria-modal="true"
          aria-label={t(strings.components.editor.info)}
        >
          <aside className="flex h-full w-full max-w-md flex-col border-l border-white/10 bg-[#0f172a] text-slate-100 shadow-2xl">
            <div className="flex items-start justify-between gap-3 border-b border-white/10 px-4 py-3">
              <div className="min-w-0">
                <h3 className="truncate text-sm font-semibold">{libraryId}</h3>
                {baselineSha && (
                  <p className="mt-1 font-mono text-xs text-slate-400">
                    {t(strings.components.editor.sha, { sha: baselineSha })}
                  </p>
                )}
              </div>
              <Button
                type="button"
                variant="outline"
                className="h-8 w-8 rounded-md p-0"
                onClick={() => setInfoOpen(false)}
                aria-label={t(strings.components.editor.closeInfo)}
              >
                <X aria-hidden className="h-4 w-4" />
              </Button>
            </div>
            <div className="min-h-0 flex-1 overflow-auto p-4">
              {metadataSlot ?? (
                <p className="text-sm text-slate-400">
                  {t(strings.components.editor.noInfo)}
                </p>
              )}
            </div>
          </aside>
        </div>
      )}
    </section>
  );
}
