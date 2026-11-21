import { useState } from "react";
import type { JsonObject, JsonValue } from "@bufbuild/protobuf";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Paperclip } from "lucide-react";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { conversationsClient } from "../../../api/notifications";
import { answerImageUrl } from "../../../api/attachments";
import { AnswerImages } from "../form/AnswerImages";
import type { PacketDraft } from "../model/draft";
import { PacketNavigationButton, type PacketNavigationState } from "../PacketNavigation";
import { useMediaQuery } from "../../../hooks/useMediaQuery";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";

interface Props {
  ask: Ask;
  draft: PacketDraft;
  onDraftChange: (update: (current: PacketDraft) => PacketDraft) => void;
  onRefresh: () => void;
  navigation?: PacketNavigationState;
  interactive?: boolean;
}

function isJsonValue(value: unknown): value is JsonValue {
  if (value === null || typeof value === "string" || typeof value === "boolean") return true;
  if (typeof value === "number") return Number.isFinite(value);
  if (Array.isArray(value)) return value.every(isJsonValue);
  return Boolean(value) && typeof value === "object" && Object.values(value).every(isJsonValue);
}

function changesJson(draft: PacketDraft): JsonObject {
  const output: JsonObject = {};
  for (const [key, change] of Object.entries(draft.answerChanges)) {
    if (isJsonValue(change.value)) output[key] = change.value;
  }
  return output;
}

export function StatusAnswerBar({ ask, draft, onDraftChange, onRefresh, navigation, interactive = false }: Props) {
  const { t } = useTranslation();
  const [busy, setBusy] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState("");
  const [reviewing, setReviewing] = useState(false);
  const phone = useMediaQuery("(max-width: 63.999rem)");
  const itemNotes = draft.answerChanges.notes?.value;
  const noteCount = itemNotes && typeof itemNotes === "object" ? Object.keys(itemNotes).length : 0;
  const changes = Object.entries(draft.answerChanges).filter(([key]) => key !== "notes");
  const verdictCount = changes.filter(([key, change]) => key.startsWith("evidence.") && (change.value === "looks-right" || change.value === "off")).length;
  const changeCount = changes.length - verdictCount;
  const hasUpdates = changeCount > 0 || noteCount > 0;
  const [noting, setNoting] = useState(false);
  const removeChange = (key: string) => {
    window.dispatchEvent(new CustomEvent("hub:answer-remove", { detail: { askId: ask.id, key } }));
    onDraftChange((current) => {
    const { [key]: _removed, ...answerChanges } = current.answerChanges;
      return { ...current, answerChanges };
    });
  };
  const removeNote = (noteKey: string) => onDraftChange((current) => {
    const value = current.answerChanges.notes?.value;
    if (!value || typeof value !== "object") return current;
    const { [noteKey]: _removed, ...rest } = value as Record<string, unknown>;
    const { notes: existingNotes, ...otherChanges } = current.answerChanges;
    const answerChanges = Object.keys(rest).length && existingNotes
      ? { ...otherChanges, notes: { ...existingNotes, value: rest } }
      : otherChanges;
    return { ...current, answerChanges };
  });
  // One chip per thing you changed: page choices, verdicts, and each item note.
  const chips: { id: string; label: string; remove: () => void }[] = [
    ...changes.map(([key, change]) => ({ id: key, label: change.label, remove: () => removeChange(key) })),
    ...(itemNotes && typeof itemNotes === "object"
      ? Object.entries(itemNotes as Record<string, { label?: string }>).map(([key, note]) => ({ id: `notes.${key}`, label: t(strings.packet.statusBar.noteChip, { label: note.label || key }), remove: () => removeNote(key) }))
      : []),
  ];
  const total = chips.length;
  const asked = ask.recurrence?.count ?? 0;
  const clearAll = () => {
    for (const [key] of changes) window.dispatchEvent(new CustomEvent("hub:answer-remove", { detail: { askId: ask.id, key } }));
    onDraftChange((current) => ({ ...current, answerChanges: {} }));
  };
  const sent = ask.state === "answered";
  const open = ask.state === "pending" || ask.state === "escalated";

  const send = async () => {
    if (!open || busy || uploading) return;
    setBusy(true);
    setError("");
    try {
      await conversationsClient.answer({
        askId: ask.id,
        answer: "",
        note: draft.note,
        changes: changesJson(draft),
        images: draft.answerImages.map((image) => ({ name: image.name, sha256: image.sha256, mediaType: image.mediaType, sizeBytes: BigInt(image.sizeBytes) })),
      });
      onRefresh();
    } catch (cause) {
    setError(cause instanceof Error ? cause.message : t(strings.packet.statusBar.uploadError));
    } finally {
      setBusy(false);
    }
  };

  const markRead = async () => {
    if (!open || busy || uploading) return;
    setBusy(true);
    setError("");
    try {
      await conversationsClient.markRead({ askId: ask.id });
      onRefresh();
    } catch (cause) {
    setError(cause instanceof Error ? cause.message : t(strings.packet.statusBar.readError));
    } finally {
      setBusy(false);
    }
  };

  return (
    <section aria-label={t(strings.packet.statusBar.label)} className="fixed inset-x-3 bottom-[max(0.75rem,env(safe-area-inset-bottom))] z-20 flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-3 shadow-lg lg:sticky lg:inset-auto lg:top-4 lg:p-4 lg:shadow-none">
      {sent ? (
        <p role="status" className="text-sm text-app-muted-foreground">{ask.answeredAt ? t(strings.packet.statusBar.sentWithTime, { time: new Date(ask.answeredAt).toLocaleString() }) : t(strings.packet.statusBar.sentError)}</p>
      ) : ask.state === "read" ? (
        <p role="status" className="text-sm text-app-muted-foreground">{t(strings.packet.statusBar.read)}</p>
      ) : (
        <>
          {phone && <div className="flex items-center justify-between gap-2">
            {navigation && <PacketNavigationButton navigation={navigation} direction="previous" />}
            {(interactive || changeCount > 0 || noteCount > 0) && <button type="button" className="min-h-11 flex-1 text-start text-sm" aria-expanded={reviewing} onClick={() => setReviewing(!reviewing)}>{noteCount ? t(strings.packet.statusBar.notes, { count: noteCount }) : changeCount ? t(strings.packet.statusBar.changes, { count: changeCount }) : verdictCount ? t(strings.packet.statusBar.verdicts, { count: verdictCount }) : t(strings.packet.statusBar.review)}</button>}
            {hasUpdates ? <Button type="button" variant="primary" disabled={busy || uploading} onClick={() => void send()}>{t(strings.packet.statusBar.send)}</Button> : <Button type="button" variant="primary" disabled={busy || uploading} onClick={() => void markRead()}>{t(strings.packet.statusBar.markRead)}</Button>}
            {navigation && <PacketNavigationButton navigation={navigation} direction="next" />}
          </div>}
          {phone && !reviewing && changeCount > 0 && <div aria-label={t(strings.packet.statusBar.selectedChanges)} className="flex flex-wrap gap-2">
            {Object.entries(draft.answerChanges).map(([key, change]) => <span key={key} className="inline-flex items-center gap-1 rounded-full border border-app-primary/50 bg-app-primary/10 px-3 py-1 text-sm">{change.label}<button type="button" aria-label={t(strings.packet.statusBar.remove, { label: change.label })} onClick={() => {
              window.dispatchEvent(new CustomEvent("hub:answer-remove", { detail: { askId: ask.id, key } }));
              onDraftChange((current) => { const { [key]: _removed, ...answerChanges } = current.answerChanges; return { ...current, answerChanges }; });
            }}>×</button></span>)}
          </div>}
          {(!phone || reviewing) && <div className="flex flex-col gap-3">
          <h2 className="flex items-center gap-2 font-semibold">
            {interactive || total > 0 ? t(strings.packet.statusBar.yourChanges) : t(strings.packet.statusBar.nothingToChoose)}
            {total > 0 && <span className="inline-grid min-w-5 place-items-center rounded-full bg-app-primary px-1.5 text-xs text-app-primary-foreground">{total}</span>}
          </h2>
          {chips.length > 0 ? (
            <div aria-label={t(strings.packet.statusBar.selectedChanges)} className="flex flex-wrap gap-1.5">
              {chips.map((chip) => (
                <span key={chip.id} className="inline-flex max-w-full items-center gap-1 rounded-full bg-app-primary/10 py-0.5 pe-0.5 ps-3 text-sm">
                  <span className="min-w-0 truncate">{chip.label}</span>
                  <button type="button" aria-label={t(strings.packet.statusBar.remove, { label: chip.label })} onClick={chip.remove} className="inline-flex min-h-8 min-w-8 items-center justify-center rounded-full text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground">×</button>
                </span>
              ))}
            </div>
          ) : interactive ? <p className="text-sm text-app-muted-foreground">{t(strings.packet.statusBar.hint)}</p> : null}
          {noting || draft.note || draft.answerImages.length ? (
            <div className="flex flex-col gap-2">
              <textarea aria-label={t(strings.packet.statusBar.note)} placeholder={t(strings.packet.statusBar.notePlaceholder)} value={draft.note} rows={3} maxLength={4000} onChange={(event) => onDraftChange((current) => ({ ...current, note: event.target.value }))} className="min-h-20 rounded-control border border-app-border bg-app-surface px-3 py-2 text-sm text-app-foreground" />
              <AnswerImages images={draft.answerImages} disabled={busy} onBusyChange={setUploading} onChange={onDraftChange} />
            </div>
          ) : (
            <button type="button" onClick={() => setNoting(true)} className="flex min-h-11 items-center justify-between rounded-control border border-dashed border-app-border px-3 text-start text-sm text-app-muted-foreground hover:border-app-primary hover:text-app-foreground">
              <span>{t(strings.packet.statusBar.addNote)}</span><Paperclip size={15} aria-hidden="true" />
            </button>
          )}
          {(asked > 1 || navigation) && (
            <p className="flex flex-col gap-0.5 border-t border-app-border pt-2 text-xs text-app-muted-foreground">
              {asked > 1 && <span>{t(strings.packet.statusBar.recurrence, { count: asked })}</span>}
              {navigation && navigation.total > 1 && <span>{t(strings.packet.statusBar.skip)}</span>}
            </p>
          )}
          <div className="flex gap-2 [&>*]:flex-1">
            {total > 0 && <Button type="button" variant="outline" disabled={busy || uploading} onClick={clearAll}>{t(strings.packet.statusBar.clear)}</Button>}
            {total > 0 || draft.note.trim() || draft.answerImages.length
              ? <Button type="button" variant="primary" disabled={busy || uploading} onClick={() => void send()}>{busy ? t(strings.packet.statusBar.sending) : total > 0 ? t(strings.packet.statusBar.sendCount, { count: total }) : t(strings.packet.statusBar.send)}</Button>
              : <Button type="button" variant="primary" disabled={busy || uploading} onClick={() => void markRead()}>{t(strings.packet.statusBar.markRead)}</Button>}
          </div>
          </div>}
        </>
      )}
      {ask.note && <p className="whitespace-pre-wrap text-sm">{ask.note}</p>}
      {ask.answerImages.map((image) => <img key={image.name} src={answerImageUrl(ask.id, image.name)} alt={image.name} className="max-h-64 max-w-full rounded-panel border border-app-border object-contain" />)}
      {error && <p role="alert" className="text-sm text-app-danger">{error}</p>}
    </section>
  );
}
