import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Editor, { type Monaco } from "@monaco-editor/react";
import type { editor } from "monaco-editor";

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
import { EmulatorChrome } from "./EmulatorChrome";
import { InspectorPanel } from "./InspectorPanel";
import { ThemeSwitcher } from "./ThemeSwitcher";

/**
 * Build the harness URL the iframe loads. The query param `v` is the
 * latest content sha256 — when the sha changes (i.e. after a save),
 * React's `src` diff forces the iframe to reload. The harness route is
 * served by the API at the same origin as the Connect transport.
 */
function harnessUrl(id: string, version: string): string {
  const base = API_BASE.replace(/\/$/, "");
  const v = encodeURIComponent(version || "initial");
  return `${base}/preview/${encodeURIComponent(id)}/harness.html?v=${v}`;
}

interface ComponentEditorProps {
  id: string;
  libraryId: string;
  onClose: () => void;
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
export function ComponentEditor({ id, libraryId, onClose }: ComponentEditorProps) {
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
  const [previewReady, setPreviewReady] = useState(false);
  const [mode, setMode] = useState<"preview" | "code">("code");
  const savedToastTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    const handler = (ev: MessageEvent) => {
      const data = ev.data as { type?: string; id?: string } | null;
      if (data && data.type === "preview-ready" && data.id === id) {
        setPreviewReady(true);
      }
    };
    window.addEventListener("message", handler);
    return () => window.removeEventListener("message", handler);
  }, [id]);

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
    setPreviewReady(false);
  }, [baselineSha]);

  useEffect(() => {
    if (contentQuery.data) {
      setBuffer(contentQuery.data.content);
      setBaselineSha(contentQuery.data.sha256);
    }
  }, [contentQuery.data]);

  useEffect(() => {
    return () => {
      if (savedToastTimerRef.current) clearTimeout(savedToastTimerRef.current);
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

  return (
    <section
      data-testid={selectors.components.editor.panel}
      aria-label={t(strings.components.editor.title, { libraryId })}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <header className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <h2
            data-testid={selectors.components.editor.title}
            className="text-sm font-medium text-slate-200"
          >
            {t(strings.components.editor.title, { libraryId })}
          </h2>
          {dirty && (
            <span
              data-testid={selectors.components.editor.dirtyBadge}
              className="rounded-full bg-amber-500/20 px-2 py-0.5 text-xs text-amber-200"
            >
              {t(strings.components.editor.dirty)}
            </span>
          )}
          {baselineSha && (
            <span
              data-testid={selectors.components.editor.shaHash}
              className="font-mono text-xs text-slate-500"
            >
              {t(strings.components.editor.sha, { sha: baselineSha.slice(0, 12) })}
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <div
            data-testid={selectors.components.editor.modeSwitch}
            className="flex overflow-hidden rounded-control border border-white/10"
          >
            <Button
              type="button"
              variant={mode === "preview" ? "default" : "outline"}
              className="h-8 rounded-none px-3 text-xs"
              onClick={() => setMode("preview")}
            >
              {t(strings.components.editor.previewMode)}
            </Button>
            <Button
              type="button"
              variant={mode === "code" ? "default" : "outline"}
              className="h-8 rounded-none px-3 text-xs"
              onClick={() => setMode("code")}
            >
              {t(strings.components.editor.codeMode)}
            </Button>
          </div>
          <Button
            data-testid={selectors.components.editor.closeButton}
            onClick={onClose}
            className="h-8 px-3 text-xs"
            variant="outline"
          >
            {t(strings.components.editor.close)}
          </Button>
          <Button
            data-testid={selectors.components.editor.saveButton}
            onClick={() => saveMutation.mutate()}
            disabled={!dirty || saveMutation.isPending || contentQuery.isLoading}
            className="h-8 px-3 text-xs"
          >
            {saveMutation.isPending
              ? t(strings.components.editor.saving)
              : t(strings.components.editor.save)}
          </Button>
        </div>
      </header>

      {contentQuery.isLoading && (
        <p
          data-testid={selectors.components.editor.loading}
          className="mt-3 text-slate-200"
        >
          {t(strings.components.editor.loading)}
        </p>
      )}

      {contentQuery.error && (
        <p
          data-testid={selectors.components.editor.error}
          className="mt-3 text-red-400"
        >
          {errorMessage(contentQuery.error, t)}
        </p>
      )}

      {saveMutation.error && (
        <p
          data-testid={selectors.components.editor.error}
          className="mt-3 text-red-400"
        >
          {errorMessage(saveMutation.error, t)}
        </p>
      )}

      {contentQuery.data && (
        <div className="mt-3 grid gap-3">
          <div
            data-testid={selectors.components.editor.surface}
            className={`overflow-hidden rounded-lg border border-white/10 ${mode === "code" ? "" : "hidden"}`}
          >
            <Editor
              height={480}
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
            className={`overflow-hidden rounded-lg border border-white/10 bg-black/40 ${mode === "preview" ? "" : "hidden"}`}
          >
            <div className="flex items-center justify-between border-b border-white/5 px-3 py-2 text-xs text-slate-400">
              <span>{t(strings.components.editor.previewHeading)}</span>
              <span
                data-testid={selectors.components.editor.previewBadge}
                className={
                  previewReady
                    ? "rounded-full bg-emerald-500/20 px-2 py-0.5 text-emerald-200"
                    : "rounded-full bg-slate-500/20 px-2 py-0.5 text-slate-300"
                }
              >
                {previewReady
                  ? t(strings.components.editor.previewReady)
                  : t(strings.components.editor.previewWaiting)}
              </span>
            </div>
            {previewReady && (
              <ThemeSwitcher frameRef={previewFrameRef} previewReady={previewReady} />
            )}
            <div className="h-[440px]">
              <EmulatorChrome emulator={emulator} filters={filters}>
                <iframe
                  data-testid={selectors.components.editor.previewFrame}
                  title={t(strings.components.editor.previewHeading)}
                  src={harnessUrl(id, baselineSha)}
                  sandbox="allow-scripts allow-same-origin"
                  ref={previewFrameRef}
                  style={{
                    width: emulator.displayWidth,
                    height: emulator.displayHeight,
                  }}
                  className="block border-0 bg-white"
                />
              </EmulatorChrome>
            </div>
            <InspectorPanel inspector={inspector} />
          </div>
        </div>
      )}

      {showSaved && (
        <p
          data-testid={selectors.components.editor.savedToast}
          className="mt-2 text-xs text-emerald-300"
        >
          {t(strings.components.editor.saved)}
        </p>
      )}
    </section>
  );
}
