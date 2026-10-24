import { useEffect, useRef, type Ref, type ReactNode } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { API_BASE } from "../../api/client";
import type { PreviewFrameCandidate } from "../../api/components";
import { selectors } from "../../consts/selectors";
import { useTranslation } from "../../i18n";
import type { PreviewKit } from "./ThemeSwitcher";
import type { PreviewSpecimen, SpecimenIdentity } from "./componentEditorStories";

type PreviewStorySheetProps = {
  specimens: Array<PreviewSpecimen | undefined>;
  readyExamples: ReadonlySet<string>;
  previewMessage: string;
  specimenErrors: Record<string, string>;
  specimenRetries: Record<string, number>;
  activeSpecimen: SpecimenIdentity | null;
  previewKit: PreviewKit;
  frameEnabled: boolean;
  frameOverride?: PreviewFrameCandidate;
  id: string;
  baselineSha: string;
  previewReloadKey: number;
  selectedVersion?: string;
  resolvedPreviewTheme: string;
  previewCanvasRef: Ref<HTMLDivElement>;
  tools: ReactNode;
  toolsDocked: boolean;
  toolsOpen: boolean;
  onRetrySpecimen: (identity: SpecimenIdentity) => void;
  onRegisterPreviewFrame: (identity: SpecimenIdentity, frame: HTMLIFrameElement | null) => void;
  onPreviewLoad: (identity: SpecimenIdentity) => void;
  onPreviewError: (identity: SpecimenIdentity) => void;
  postToPreviewFrames: (message: unknown) => void;
  onCloseTools: () => void;
};

function harnessUrl(
  id: string,
  contentVersion: string,
  reloadKey: number,
  example: PreviewSpecimen | undefined,
  selectedVersion: string | undefined,
  kit: PreviewKit,
  frameEnabled: boolean,
  frameOverride: PreviewFrameCandidate | undefined,
): string {
  const url = new URL(`${API_BASE.replace(/\/$/, "")}/preview/${encodeURIComponent(id)}/harness.html`);
  url.searchParams.set("v", encodeURIComponent(contentVersion || "initial"));
  url.searchParams.set("r", String(reloadKey));
  url.searchParams.set("kit", kit);
  url.searchParams.set("frame", frameEnabled ? "on" : "off");
  url.searchParams.set("sheet", "1");
  if (frameOverride) {
    url.searchParams.set("frameAsset", frameOverride.asset);
    url.searchParams.set("frameVersion", frameOverride.version);
    url.searchParams.set("frameRegion", frameOverride.region);
    url.searchParams.set("frameCapability", frameOverride.capability);
    url.searchParams.set("frameFixture", frameOverride.fixture);
  }
  if (selectedVersion) url.searchParams.set("version", selectedVersion);
  if (example) {
    url.searchParams.set("story", example.storyId || example.name);
    url.searchParams.set("version", selectedVersion || example.version);
  }
  return url.toString();
}

function StoryFrame({
  identity,
  example,
  src,
  error,
  onRegister,
  onLoad,
  onError,
  onRetry,
}: {
  identity: SpecimenIdentity;
  example?: PreviewSpecimen;
  src: string;
  error?: string;
  onRegister: (identity: SpecimenIdentity, frame: HTMLIFrameElement | null) => void;
  onLoad: (identity: SpecimenIdentity) => void;
  onError: (identity: SpecimenIdentity) => void;
  onRetry: (identity: SpecimenIdentity) => void;
}) {
  const frameRef = useRef<HTMLIFrameElement | null>(null);
  useEffect(() => {
    const frame = frameRef.current;
    if (!frame) return;
    const fit = () => {
      const document = frame.contentDocument;
      const root = document?.querySelector<HTMLElement>("[data-preview-sheet]") || document?.body;
      const height = root?.scrollHeight || document?.documentElement?.scrollHeight || 0;
      if (height > 0) frame.style.height = `${Math.max(96, height + 8)}px`;
    };
    const observer = frame.contentDocument ? new ResizeObserver(fit) : null;
    if (observer && frame.contentDocument?.documentElement) observer.observe(frame.contentDocument.documentElement);
    fit();
    return () => observer?.disconnect();
  }, [src]);

  return error ? (
    <div data-testid={selectors.components.editor.specimenError} className="flex min-h-24 flex-col items-center justify-center gap-space-xs rounded-control border border-dashed border-app-danger/50 bg-app-danger/5 p-space-sm text-center">
      <p className="text-xs text-app-danger">{error}</p>
      <Button data-testid={selectors.components.editor.specimenRetry} type="button" variant="secondary" className="h-control-tight px-space-xs text-xs" onClick={() => onRetry(identity)}>
        Retry preview
      </Button>
    </div>
  ) : (
    <iframe
      ref={(frame) => {
        frameRef.current = frame;
        onRegister(identity, frame);
      }}
      data-testid={selectors.components.editor.previewFrame}
      data-specimen={identity}
      title={`${example?.displayName || example?.name || "Story"} preview`}
      src={src}
      sandbox="allow-scripts allow-same-origin"
      loading="eager"
      onLoad={() => {
        onLoad(identity);
        requestAnimationFrame(() => {
          const document = frameRef.current?.contentDocument;
          const root = document?.querySelector<HTMLElement>("[data-preview-sheet]") || document?.body;
          const height = root?.scrollHeight || document?.documentElement?.scrollHeight || 0;
          if (height > 0 && frameRef.current) frameRef.current.style.height = `${Math.max(96, height + 8)}px`;
        });
      }}
      onError={() => onError(identity)}
      className="block min-h-24 w-full border-0 bg-app-background"
      style={{ height: "auto" }}
    />
  );
}

export function PreviewStorySheet({
  specimens,
  readyExamples,
  previewMessage,
  specimenErrors,
  specimenRetries,
  activeSpecimen,
  previewKit,
  frameEnabled,
  frameOverride,
  id,
  baselineSha,
  previewReloadKey,
  selectedVersion,
  resolvedPreviewTheme,
  previewCanvasRef,
  tools,
  toolsDocked,
  toolsOpen,
  onRetrySpecimen,
  onRegisterPreviewFrame,
  onPreviewLoad,
  onPreviewError,
  postToPreviewFrames,
  onCloseTools,
}: PreviewStorySheetProps) {
  const { t } = useTranslation();
  return (
    <div ref={previewCanvasRef} data-preview-sheet="workbench" data-testid={selectors.components.editor.gallery} className="min-h-0 flex-1 overflow-auto bg-app-background p-space-sm">
      <div className="mx-auto grid w-full max-w-screen-xl gap-space-sm" data-preview-capture-boundary="component-sheet">
        <header className="flex flex-wrap items-baseline justify-between gap-space-2xs border-b border-app-border pb-space-xs">
          <div>
            <h2 className="text-sm font-semibold text-app-foreground">Story sheet</h2>
            <p className="text-xs text-app-muted-foreground" aria-live="polite">
              {previewMessage || `${readyExamples.size} of ${specimens.length} stories ready`}
            </p>
          </div>
          <span className="text-xs text-app-muted-foreground">{specimens.length} stories · {previewKit} · {resolvedPreviewTheme}</span>
        </header>
        <div className="grid grid-cols-[repeat(auto-fit,minmax(min(100%,18rem),1fr))] items-start gap-space-sm">
          {specimens.map((example) => {
            const identity = `${example?.version || "__current__"}:${example?.storyId || "__default__"}` as SpecimenIdentity;
            const title = example?.displayName || example?.name || "Missing story";
            const role = example?.role || "unclassified";
            return (
              <article key={identity} data-testid={selectors.components.editor.exampleCard} data-specimen={identity} data-story={example?.storyId || "__missing__"} data-story-ready={readyExamples.has(identity) ? "true" : "false"} className={`min-w-0 rounded-panel border bg-app-surface p-space-xs shadow-sm ${activeSpecimen === identity ? "border-app-primary ring-1 ring-app-primary/30" : "border-app-border"}`}>
                <header className="mb-space-2xs flex items-center justify-between gap-space-xs">
                  <div className="min-w-0">
                    <h3 data-testid={selectors.components.editor.exampleTitle} className="truncate text-sm font-semibold text-app-foreground">{title}</h3>
                    <p data-testid={example?.description ? selectors.components.editor.storyDescription : undefined} className="text-[0.6875rem] text-app-muted-foreground"><span className="font-semibold uppercase">{role}</span>{example?.description ? ` · ${example.description}` : ""}</p>
                  </div>
                  <span data-testid={selectors.components.editor.exampleDimensions} className="shrink-0 font-mono text-[0.6875rem] text-app-muted-foreground" aria-label="Rendered story dimensions">auto</span>
                </header>
                {example ? (
                  <StoryFrame
                    identity={identity}
                    example={example}
                    src={harnessUrl(id, baselineSha, previewReloadKey + (specimenRetries[identity] ?? 0), example, selectedVersion, previewKit, frameEnabled, frameOverride)}
                    error={specimenErrors[identity]}
                    onRegister={onRegisterPreviewFrame}
                    onLoad={(storyIdentity) => {
                      onPreviewLoad(storyIdentity);
                      postToPreviewFrames({ type: "rcl-resolved-theme", theme: resolvedPreviewTheme });
                    }}
                    onError={onPreviewError}
                    onRetry={onRetrySpecimen}
                  />
                ) : (
                  <div className="flex min-h-24 items-center justify-center rounded-control border border-dashed border-app-border bg-app-surface-muted px-space-xs text-center text-xs text-app-muted-foreground">
                    Missing required story: {title} ({role})
                  </div>
                )}
              </article>
            );
          })}
        </div>
      </div>
      {activeSpecimen && toolsDocked && (
        <aside id="component-preview-tools" data-testid={selectors.components.editor.previewToolsPanel} aria-label={t("components.editor.showTools", { defaultValue: "Preview controls" })} className={`fixed bottom-space-xs right-space-xs top-24 z-30 w-stage-panel overflow-hidden rounded-panel border border-app-border bg-app-surface/98 shadow-2xl ${toolsOpen ? "flex flex-col" : "hidden"}`}>
          <div className="flex shrink-0 items-center justify-between gap-space-xs border-b border-app-border px-space-xs py-space-2xs">
            <p className="text-sm font-semibold text-app-foreground">{t("components.editor.previewControls", { defaultValue: "Preview controls" })}</p>
            <Button type="button" variant="secondary" className="h-control-tight px-space-2xs text-xs" onClick={onCloseTools}>{t("common.close", { defaultValue: "Close" })}</Button>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto p-space-xs">{tools}</div>
        </aside>
      )}
    </div>
  );
}
