import { useState } from "react";
import type { JsonObject, JsonValue } from "@bufbuild/protobuf";
import { Button } from "@vrooli/react-component-library/Button/2";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { conversationsClient } from "../../../api/notifications";
import { answerImageUrl } from "../../../api/attachments";
import { AnswerImages } from "../form/AnswerImages";
import type { PacketDraft } from "../model/draft";

interface Props {
  ask: Ask;
  draft: PacketDraft;
  onDraftChange: (update: (current: PacketDraft) => PacketDraft) => void;
  onRefresh: () => void;
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

export function StatusAnswerBar({ ask, draft, onDraftChange, onRefresh }: Props) {
  const [busy, setBusy] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState("");
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
      setError(cause instanceof Error ? cause.message : "Your update was not confirmed. Check the status before sending again.");
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
      setError(cause instanceof Error ? cause.message : "The update could not be marked read.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <section aria-label="Status update response" className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4">
      {sent ? (
        <p role="status" className="text-sm text-app-muted-foreground">Your steering was sent{ask.answeredAt ? ` at ${new Date(ask.answeredAt).toLocaleString()}` : ""}.</p>
      ) : ask.state === "read" ? (
        <p role="status" className="text-sm text-app-muted-foreground">Marked as read.</p>
      ) : (
        <>
          {Object.entries(draft.answerChanges).length > 0 && (
            <div aria-label="Selected page changes" className="flex flex-wrap gap-2">
              {Object.entries(draft.answerChanges).map(([key, change]) => (
                <span key={key} className="inline-flex max-w-full items-center gap-2 rounded-full border border-app-primary/50 bg-app-primary/10 px-3 py-1.5 text-sm">
                  <span>{change.label}</span>
                  <Button type="button" size="sm" variant="ghost" aria-label={`Remove ${change.label}`} onClick={() => onDraftChange((current) => {
                    const answerChanges = { ...current.answerChanges };
                    delete answerChanges[key];
                    return { ...current, answerChanges };
                  })}>×</Button>
                </span>
              ))}
            </div>
          )}
          <label className="flex flex-col gap-1 text-sm font-medium">
            Note
            <textarea value={draft.note} rows={3} maxLength={4000} onChange={(event) => onDraftChange((current) => ({ ...current, note: event.target.value }))} className="min-h-24 rounded-control border border-app-border bg-app-surface px-3 py-2 text-app-foreground" />
          </label>
          <AnswerImages images={draft.answerImages} disabled={busy} onBusyChange={setUploading} onChange={onDraftChange} />
          <div className="flex flex-wrap gap-2">
            <Button type="button" variant="primary" disabled={busy || uploading} onClick={() => void send()}>{busy ? "Sending…" : "Send"}</Button>
            <Button type="button" variant="outline" disabled={busy || uploading} onClick={() => void markRead()}>Mark read</Button>
          </div>
        </>
      )}
      {ask.note && <p className="whitespace-pre-wrap text-sm">{ask.note}</p>}
      {ask.answerImages.map((image) => <img key={image.name} src={answerImageUrl(ask.id, image.name)} alt={image.name} className="max-h-64 max-w-full rounded-panel border border-app-border object-contain" />)}
      {error && <p role="alert" className="text-sm text-app-danger">{error}</p>}
    </section>
  );
}
