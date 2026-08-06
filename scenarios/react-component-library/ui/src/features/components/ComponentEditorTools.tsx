/** @vrooliComponentSource overlays.dialog */
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
type ActiveStory = { storyId?: string; displayName?: string; name?: string; propsJson?: string; environment?: Record<string, string> };
type OverrideState = Record<string, "idle" | "applying" | "applied" | "error">;

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
  onApply: (props: Record<string, unknown>, environment?: Record<string, string>) => void;
  onReset: () => void;
  onClearEvents: () => void;
}

function PropsTool({ activeSpecimen, activeExample, activeSpecimenLabel, storyContract, overrideStatus, specimenOverrides, overrideMessages, onApply, onReset }: ToolProps) {
  const initialArgs = activeExample?.propsJson ? parseArgs(activeExample.propsJson) : {};
  return <PropsExperimentPanel key={activeSpecimen ?? "none"} storyId={activeExample?.storyId} storyName={activeSpecimenLabel} initialArgs={initialArgs} initialEnvironment={activeExample?.environment} storyContract={storyContract} status={activeSpecimen ? overrideStatus[activeSpecimen] ?? (specimenOverrides[activeSpecimen] ? "applied" : "idle") : "idle"} message={activeSpecimen ? overrideMessages[activeSpecimen] : undefined} onApply={onApply} onReset={onReset} />;
}

function parseArgs(raw: string): Record<string, unknown> {
  try {
    const parsed: unknown = JSON.parse(raw);
    return parsed && typeof parsed === "object" && !Array.isArray(parsed) ? Object.fromEntries(Object.entries(parsed)) : {};
  } catch {
    return {};
  }
}

export function ComponentEditorTools(props: ToolProps) {
  const { t } = useTranslation();
  const { activeExample, activeSpecimenLabel, inspector, previewEvents, onClearEvents } = props;
  return <><div className="grid gap-space-xs xl:grid-cols-[minmax(18rem,0.8fr)_minmax(20rem,1.2fr)]"><PropsTool {...props} /><InspectorPanel inspector={inspector} specimenLabel={activeSpecimenLabel} /></div><section className="mt-space-xs rounded-md border border-app-border p-space-xs" aria-label={t(strings.components.editor.events)}><div className="mb-space-2xs flex items-center justify-between gap-space-2xs"><h3 className="text-sm font-semibold">{t(strings.components.editor.events)}</h3><Button data-testid={selectors.components.editor.previewEventsClear} type="button" variant="secondary" className="h-7 px-space-2xs text-xs" onClick={onClearEvents}>{t(strings.components.editor.clearEvents)}</Button></div><ol className="max-h-48 space-y-space-3xs overflow-auto font-mono text-xs">{previewEvents.filter((event) => !activeExample?.storyId || event.story === activeExample.storyId).map((event, index) => <li key={`${event.ts}-${index}`} data-testid={selectors.components.editor.previewEventItem} className="break-words rounded bg-app-muted/50 px-space-2xs py-space-3xs"><span className="font-semibold">{event.name}</span>{event.args.length ? `(${event.args.map((arg) => JSON.stringify(arg)).join(", ")})` : "()"}</li>)}{previewEvents.length === 0 ? <li data-testid={selectors.components.editor.previewEventsEmpty} className="font-sans text-app-muted-foreground">{t(strings.components.editor.noEvents)}</li> : null}</ol></section></>;
}

export function ComponentEditorMobileTools({ tool, onClose, props }: { tool: "props" | "inspector" | null; onClose: () => void; props: ToolProps }) {
  const { t } = useTranslation();
  const { activeSpecimen, activeSpecimenLabel, inspector } = props;
  return <Dialog open={tool !== null} onClose={onClose} title={tool === "props" ? t(strings.components.editor.tryProps) : t("components.inspector.title", { defaultValue: "Inspect" })} closeLabel={t("common.close", { defaultValue: "Close" })} className="lg:hidden">{tool === "props" && activeSpecimen ? <PropsTool {...props} /> : null}{tool === "inspector" ? <InspectorPanel inspector={inspector} specimenLabel={activeSpecimenLabel} /> : null}</Dialog>;
}
