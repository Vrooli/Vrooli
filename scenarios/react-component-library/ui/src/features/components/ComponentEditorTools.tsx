/** @vrooliComponentSource overlays.dialog */
import { useState } from "react";
import { Button } from "../../components/Button";
import { Dialog } from "../../components/Dialog";
import type { ComponentStory } from "../../api/components";
import type { UseComponentInspectorReturn } from "../../hooks/useComponentInspector";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { InspectorPanel } from "./InspectorPanel";
import { PropsExperimentPanel } from "./PropsExperimentPanel";

export type EditorPreviewEvent = { story: string; name: string; args: unknown[]; ts: number };
type ActiveStory = {
  storyId?: string;
  displayName?: string;
  name?: string;
  description?: string;
  propsJson?: string;
  environment?: Record<string, string>;
};
type OverrideState = Record<string, "idle" | "applying" | "applied" | "error">;

export type PreviewDiagnostics = {
  iframeUrl: string;
  componentId: string;
  storyId: string;
  version: string;
  kit: string;
  theme: string;
  frame: string;
  error?: string;
};

interface ToolProps {
  activeSpecimen: string | null;
  activeExample?: ActiveStory;
  activeSpecimenLabel?: string;
  storyContract?: ComponentStory;
  inspector: UseComponentInspectorReturn;
  overrideStatus: OverrideState;
  specimenOverrides: Record<string, Record<string, unknown>>;
  overrideMessages: Record<string, string>;
  previewEvents: EditorPreviewEvent[];
  previewDiagnostics: PreviewDiagnostics;
  onApply: (props: Record<string, unknown>, environment?: Record<string, string>) => void;
  onReset: () => void;
  onClearEvents: () => void;
}

function PropsTool({
  activeSpecimen,
  activeExample,
  activeSpecimenLabel,
  storyContract,
  overrideStatus,
  specimenOverrides,
  overrideMessages,
  onApply,
  onReset,
}: ToolProps) {
  const initialArgs = activeExample?.propsJson ? parseArgs(activeExample.propsJson) : {};
  return (
    <PropsExperimentPanel
      key={activeSpecimen ?? "none"}
      storyId={activeExample?.storyId}
      storyName={activeSpecimenLabel}
      storyDescription={activeExample?.description}
      initialArgs={initialArgs}
      initialEnvironment={activeExample?.environment}
      storyContract={storyContract}
      status={
        activeSpecimen
          ? (overrideStatus[activeSpecimen] ??
            (specimenOverrides[activeSpecimen] ? "applied" : "idle"))
          : "idle"
      }
      message={activeSpecimen ? overrideMessages[activeSpecimen] : undefined}
      onApply={onApply}
      onReset={onReset}
    />
  );
}

function parseArgs(raw: string): Record<string, unknown> {
  try {
    const parsed: unknown = JSON.parse(raw);
    return parsed && typeof parsed === "object" && !Array.isArray(parsed)
      ? Object.fromEntries(Object.entries(parsed))
      : {};
  } catch {
    return {};
  }
}

export function ComponentEditorTools(props: ToolProps) {
  const { t } = useTranslation();
  const {
    activeExample,
    activeSpecimenLabel,
    inspector,
    previewEvents,
    onClearEvents,
    previewDiagnostics,
  } = props;
  const [diagnosticsCopied, setDiagnosticsCopied] = useState(false);
  const diagnosticsText = JSON.stringify(previewDiagnostics, null, 2);
  const copyDiagnostics = async () => {
    try {
      await navigator.clipboard.writeText(diagnosticsText);
      setDiagnosticsCopied(true);
      window.setTimeout(() => setDiagnosticsCopied(false), 1800);
    } catch {
      setDiagnosticsCopied(false);
    }
  };
  return (
    <>
      <div className="grid gap-space-xs xl:grid-cols-[minmax(18rem,0.8fr)_minmax(20rem,1.2fr)]">
        <PropsTool {...props} />
        <InspectorPanel inspector={inspector} specimenLabel={activeSpecimenLabel} />
      </div>
      <section
        className="mt-space-xs rounded-md border border-app-border p-space-xs"
        aria-label={t(strings.components.editor.events)}
      >
        <div className="mb-space-2xs flex items-center justify-between gap-space-2xs">
          <h3 className="text-sm font-semibold">{t(strings.components.editor.events)}</h3>
          <Button
            data-testid={selectors.components.editor.previewEventsClear}
            type="button"
            variant="secondary"
            className="h-control-compact px-space-2xs text-xs"
            onClick={onClearEvents}
          >
            {t(strings.components.editor.clearEvents)}
          </Button>
        </div>
        <ol className="max-h-content-short space-y-space-3xs overflow-auto font-mono text-xs">
          {previewEvents
            .filter((event) => !activeExample?.storyId || event.story === activeExample.storyId)
            .map((event, index) => (
              <li
                key={`${event.ts}-${index}`}
                data-testid={selectors.components.editor.previewEventItem}
                className="break-words rounded bg-app-muted/50 px-space-2xs py-space-3xs"
              >
                <span className="font-semibold">{event.name}</span>
                {event.args.length
                  ? `(${event.args.map((arg) => JSON.stringify(arg)).join(", ")})`
                  : "()"}
              </li>
            ))}
          {previewEvents.length === 0 ? (
            <li
              data-testid={selectors.components.editor.previewEventsEmpty}
              className="font-sans text-app-muted-foreground"
            >
              {t(strings.components.editor.noEvents)}
            </li>
          ) : null}
        </ol>
      </section>
      <section
        data-testid={selectors.components.editor.previewDiagnostics}
        className="mt-space-xs rounded-md border border-app-border p-space-xs"
        aria-label={t(strings.components.editor.previewDiagnostics)}
      >
        <div className="mb-space-2xs flex items-center justify-between gap-space-2xs">
          <div>
            <h3 className="text-sm font-semibold">
              {t(strings.components.editor.previewDiagnostics)}
            </h3>
            <p className="text-xs text-app-muted-foreground">
              {t(strings.components.editor.previewDiagnosticsDescription)}
            </p>
          </div>
          <Button
            data-testid={selectors.components.editor.previewDiagnosticsCopy}
            type="button"
            variant="secondary"
            className="h-control-compact shrink-0 px-space-2xs text-xs"
            onClick={() => void copyDiagnostics()}
          >
            {diagnosticsCopied
              ? t(strings.components.editor.previewDiagnosticsCopied)
              : t(strings.components.editor.previewDiagnosticsCopy)}
          </Button>
        </div>
        <pre className="max-h-content-short overflow-auto whitespace-pre-wrap break-all rounded bg-app-muted/50 p-space-2xs font-mono text-xs text-app-muted-foreground">
          {diagnosticsText}
        </pre>
        {diagnosticsCopied ? (
          <span
            data-testid={selectors.components.editor.previewDiagnosticsCopied}
            role="status"
            className="sr-only"
          >
            {t(strings.components.editor.previewDiagnosticsCopied)}
          </span>
        ) : null}
      </section>
    </>
  );
}

export function ComponentEditorMobileTools({
  tool,
  onClose,
  props,
}: {
  tool: "props" | "inspector" | null;
  onClose: () => void;
  props: ToolProps;
}) {
  const { t } = useTranslation();
  const { activeSpecimen, activeSpecimenLabel, inspector } = props;
  return (
    <Dialog
      open={tool !== null}
      onClose={onClose}
      title={
        tool === "props"
          ? t(strings.components.editor.tryProps)
          : t("components.inspector.title", { defaultValue: "Inspect" })
      }
      closeLabel={t("common.close", { defaultValue: "Close" })}
      className="lg:hidden"
    >
      {tool === "props" && activeSpecimen ? <PropsTool {...props} /> : null}
      {tool === "inspector" ? (
        <InspectorPanel inspector={inspector} specimenLabel={activeSpecimenLabel} />
      ) : null}
    </Dialog>
  );
}
