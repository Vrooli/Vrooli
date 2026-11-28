import { useEffect, useMemo, useState } from "react";
import type { AskAttachment } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { Button } from "@vrooli/react-component-library/Button/2";
import { X } from "lucide-react";

import { attachmentUrl } from "../../../api/attachments";
import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { TextArea } from "../form/TextArea";
import { withAttachmentNote, type PacketDraft } from "../model/draft";
import { ANSWER_LIMITS } from "../model/questions";
import { regionEntryPath, type CaptureRef, type FeedbackRegion, type ParsedRegionFeedback, type RegionFeedbackIssue } from "./regionFeedback";

interface RegionFeedbackCompareProps {
  askId: string;
  attachment: AskAttachment;
  parsed: ParsedRegionFeedback | null;
  failed: boolean;
  onClose: () => void;
  draft: PacketDraft;
  onDraftChange: (next: PacketDraft) => void;
  canNote: boolean;
}

export function RegionFeedbackCompare({ askId, attachment, parsed, failed, onClose, draft, onDraftChange, canNote }: RegionFeedbackCompareProps) {
  const { t } = useTranslation();
  const [side, setSide] = useState<"before" | "after">("after");
  const [selected, setSelected] = useState<string | null>(null);
  const [text, setText] = useState("");
  const bundle = parsed?.bundle;
  const region = bundle?.regions.find((item) => item.id === selected);
  const notePath = region ? regionEntryPath(region.id) : "";
  const savedNote = region ? draft.attachmentNotes.find((note) => note.attachment === attachment.name && note.path === notePath)?.note ?? "" : "";
  const canSubmit = text.trim().length > 0 && text.trim().length <= ANSWER_LIMITS.attachmentNoteChars;
  const sideLabel = (key: "before" | "after") => key === "before" ? t(strings.packet.regionFeedback.before) : t(strings.packet.regionFeedback.after);
  const fidelityLabel = (value: "captured" | "composited" | "rendered" | "sketch") => {
    if (value === "captured") return t(strings.packet.regionFeedback.captured);
    if (value === "composited") return t(strings.packet.regionFeedback.composited);
    if (value === "rendered") return t(strings.packet.regionFeedback.rendered);
    return t(strings.packet.regionFeedback.sketch);
  };
  const issueText = (issue: RegionFeedbackIssue) => {
    switch (issue.code) {
      case "routeMismatch": return t(strings.packet.regionFeedback.issues.routeMismatch);
      case "stateMismatch": return t(strings.packet.regionFeedback.issues.stateMismatch);
      case "dataMismatch": return t(strings.packet.regionFeedback.issues.dataMismatch);
      case "viewportMismatch": return t(strings.packet.regionFeedback.issues.viewportMismatch);
      case "sameBuild": return t(strings.packet.regionFeedback.issues.sameBuild);
      case "imageMismatch": return t(strings.packet.regionFeedback.issues.imageMismatch, { fidelity: fidelityLabel(issue.fidelity) });
      case "regionEntryMissing": return t(strings.packet.regionFeedback.issues.regionEntryMissing, { regionId: issue.regionId });
      case "regionCropMismatch": return t(strings.packet.regionFeedback.issues.regionCropMismatch, { regionId: issue.regionId });
    }
  };

  useEffect(() => setText(savedNote), [savedNote]);
  useEffect(() => {
    const handle = (event: KeyboardEvent) => { if (event.key === "Escape") onClose(); };
    window.addEventListener("keydown", handle);
    return () => window.removeEventListener("keydown", handle);
  }, [onClose]);

  const imageByPath = useMemo(() => (capture: CaptureRef) =>
    attachmentUrl(askId, attachment.name, capture.image.path), [askId, attachment.name]);
  const unsupported = failed || Boolean(parsed?.issues.some((item) => item.code === "imageMismatch" || item.code === "regionEntryMissing" || item.code === "regionCropMismatch"));

  const chooseRegion = (item: FeedbackRegion) => {
    setSelected(item.id);
    const saved = draft.attachmentNotes.find((note) => note.attachment === attachment.name && note.path === regionEntryPath(item.id))?.note ?? "";
    setText(saved);
  };

  const submit = () => {
    if (!region || !canSubmit || !canNote) return;
    onDraftChange(withAttachmentNote(draft, attachment.name, regionEntryPath(region.id), text.trim()));
  };

  return (
    <div role="dialog" aria-modal="true" aria-label={t(strings.packet.regionFeedback.title)} className="fixed inset-0 z-[100] flex flex-col bg-app-surface text-app-foreground">
      <header className="flex min-h-14 items-center justify-between gap-3 border-b border-app-border px-4">
        <div className="min-w-0">
          <h1 className="truncate text-base font-semibold">{t(strings.packet.regionFeedback.title)}</h1>
          {bundle && <p className="truncate text-xs text-app-muted-foreground">{bundle.route}</p>}
        </div>
        <Button type="button" variant="outline" aria-label={t(strings.packet.regionFeedback.close)} onClick={onClose} className="min-h-11 min-w-11 shrink-0 border border-app-border px-3">
          <X size={18} aria-hidden="true" />
        </Button>
      </header>
      {unsupported || !bundle ? (
        <p role="alert" className="m-4 rounded-panel border border-app-warning/40 p-4 text-sm">{t(strings.packet.regionFeedback.unavailable)}</p>
      ) : (
        <div className="flex min-h-0 flex-1 flex-col overflow-y-auto">
          {parsed.issues.length > 0 && (
            <div role="status" className="m-3 rounded-panel border border-app-warning/50 bg-app-warning/10 p-3 text-sm">
              {parsed.issues.map((issue, index) => <p key={`${issue.code}-${index}`}>{t(strings.packet.regionFeedback.comparisonWarning, { reason: issueText(issue) })}</p>)}
            </div>
          )}
          <div className="flex gap-2 px-3 pt-2 md:hidden" role="tablist" aria-label={t(strings.packet.regionFeedback.title)}>
            {(["before", "after"] as const).map((key) => (
              <button key={key} role="tab" aria-selected={side === key} type="button" onClick={() => setSide(key)} className={`min-h-11 flex-1 rounded-control border px-3 text-sm font-semibold ${side === key ? "border-app-primary bg-app-primary/10 text-app-primary" : "border-app-border"}`}>
                {sideLabel(key)}
              </button>
            ))}
          </div>
          <div className="grid gap-3 p-3 md:grid-cols-2 md:gap-4 md:p-5">
            {(["before", "after"] as const).map((key) => {
              const capture = bundle[key];
              const isAfter = key === "after";
              return (
                <figure key={key} className={`relative m-0 min-w-0 overflow-hidden rounded-panel border border-app-border bg-app-surface-muted ${side === key ? "block" : "hidden md:block"}`}>
                  <figcaption className="flex flex-wrap items-center justify-between gap-2 border-b border-app-border px-3 py-2 text-sm">
                    <span className="font-semibold">{sideLabel(key)}</span>
                    <span data-fidelity={capture.fidelity} className="rounded-pill bg-app-primary/10 px-2.5 py-1 text-xs font-medium text-app-primary">{t(strings.packet.regionFeedback.fidelity, { label: fidelityLabel(capture.fidelity) })}</span>
                  </figcaption>
                  <div className="relative">
                    <img src={imageByPath(capture)} alt={`${sideLabel(key)}: ${bundle.route}`} className="block h-auto w-full" />
                    {isAfter && bundle.regions.map((item) => (
                      <button
                        key={item.id}
                        type="button"
                        aria-label={t(strings.packet.regionFeedback.regionId, { id: item.id })}
                        title={`${item.label} · ${item.component}`}
                        onClick={() => chooseRegion(item)}
                        style={{ left: `${item.bounds.x * 100}%`, top: `${item.bounds.y * 100}%`, width: `${item.bounds.width * 100}%`, height: `${item.bounds.height * 100}%` }}
                        className={`absolute border-2 ${selected === item.id ? "border-app-primary bg-app-primary/20" : "border-white bg-app-primary/10 hover:bg-app-primary/25"}`}
                      />
                    ))}
                  </div>
                  <div className="grid grid-cols-2 gap-x-2 gap-y-1 px-3 py-2 text-xs text-app-muted-foreground">
                    <span>{t(strings.packet.regionFeedback.route, { route: capture.route })}</span>
                    <span>{t(strings.packet.regionFeedback.viewport, capture.viewport)}</span>
                    <span className="col-span-2 break-all">{t(strings.packet.regionFeedback.build, { build: capture.build_id })}</span>
                  </div>
                </figure>
              );
            })}
          </div>
          <section aria-labelledby="region-feedback-list" className="px-3 pb-4 md:px-5">
            <h2 id="region-feedback-list" className="mb-2 text-sm font-semibold">{t(strings.packet.regionFeedback.regionsTitle)}</h2>
            <div className="grid gap-2 sm:grid-cols-2">
              {bundle.regions.map((item) => (
                <button key={item.id} type="button" onClick={() => chooseRegion(item)} aria-pressed={selected === item.id} className={`min-h-12 rounded-control border p-3 text-start text-sm ${selected === item.id ? "border-app-primary bg-app-primary/10" : "border-app-border"}`}>
                  <span className="block font-medium">{t(strings.packet.regionFeedback.region, { label: item.label, component: item.component })}</span>
                  <span className="mt-1 block font-mono text-xs text-app-muted-foreground">{t(strings.packet.regionFeedback.regionId, { id: item.id })}</span>
                </button>
              ))}
            </div>
            {region && (
              <div className="mt-4 rounded-panel border border-app-border p-4">
                <div className="mb-3 flex items-start gap-3 text-sm text-app-muted-foreground">
                  <img src={attachmentUrl(askId, attachment.name, region.crop.path)} alt="" className="h-16 w-16 rounded-control border border-app-border object-cover" />
                  <p>{region.label} · {region.component} · {region.node_id}</p>
                </div>
                {canNote ? (
                  <>
                    <TextArea label={t(strings.packet.regionFeedback.commentLabel, { region: region.label })} value={text} onChange={setText} maxLength={ANSWER_LIMITS.attachmentNoteChars} rows={3} testId={selectors.packet.viewerNote} />
                    <p className="mt-2 text-xs text-app-muted-foreground">{t(strings.packet.regionFeedback.commentHint)}</p>
                    <Button type="button" variant="primary" className="mt-3 min-h-11 w-full" disabled={!canSubmit} onClick={submit}>{t(savedNote ? strings.packet.regionFeedback.noteSaved : strings.packet.regionFeedback.save)}</Button>
                  </>
                ) : <p className="text-sm">{t(strings.packet.regionFeedback.notAnswerable)}</p>}
              </div>
            )}
          </section>
        </div>
      )}
    </div>
  );
}
