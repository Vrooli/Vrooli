import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQueries } from "@tanstack/react-query";
import { AskAttachment_Kind, type AskAttachment } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { Button } from "@vrooli/react-component-library/Button/2";

import { selectors } from "../../../consts/selectors";
import { fetchAttachment } from "../../../api/attachments";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { useTheme } from "../../../theme/themeContext";
import { BRIDGE_PROTOCOL, MIN_FRAME_HEIGHT, parseBridgeMessage } from "./bridge";
import frameBridgeJs from "./frameBridge.js?raw";
import { buildExplainerDocument } from "./document";
import kitCss from "./kit/kit.css?raw";
import kitJs from "./kit/kit.js?raw";
import { pageColorScheme, pageTheme } from "./theme";
import { useExplainerPage } from "./useExplainerPage";
import { renderKitTemplate, type KitTemplateError } from "./kitTemplate";
import { conversationsClient } from "../../../api/notifications";

interface ExplainerFrameProps {
  askId: string;
  subject?: string;
  kind?: string;
  recurrence?: unknown;
  templateName?: string;
  templateData?: import("@bufbuild/protobuf").JsonObject;
  answerChanges?: Record<string, { label: string; value: unknown }>;
  explainer?: AskAttachment;
  attachments: readonly AskAttachment[];
  /** The page asked to open an attachment (already known to be a valid name). */
  onOpen: (name: string, path: string) => void;
  /** The page asked to fill one question of the Hub form; the operator still sends. */
  onPrefill: (questionId: string, value: unknown) => void;
  onAnswerChange?: (change: { type: "answer"; operation: "set"; key: string; label: string; value: unknown } | { type: "answer"; operation: "remove"; key: string }) => void;
  onAnswerCapability?: (interactive: boolean) => void;
  onOpenAsk?: (askId: string) => void;
}

// Explainer context is a read-only convenience, not a bulk ask browser. Bound
// both the template references inspected and the corresponding live reads.
const MAX_NAMED_ASK_REFERENCES = 20;
const MAX_DYNAMIC_ASSET_BYTES = 12 * 1024 * 1024;

/**
 * The agent's own HTML page, sealed. It is untrusted code, so it runs in an
 * `allow-scripts` sandbox without `allow-same-origin` (opaque origin: none of
 * the Hub's cookies, storage or API), under a CSP that allows no network. It
 * can inform and pre-fill; only the Hub's own form answers.
 */
export function ExplainerFrame({ askId, subject = "", kind, recurrence, templateName, templateData, answerChanges, explainer, attachments, onOpen, onPrefill, onAnswerChange, onAnswerCapability, onOpenAsk }: ExplainerFrameProps) {
  const { t } = useTranslation();
  const { resolved: theme } = useTheme();
  const page = useExplainerPage(askId, subject, explainer, attachments);
  const namedAskIds = useMemo(() => {
    const data = templateData as Record<string, unknown> | undefined;
    const asks = Array.isArray(data?.asks) ? data.asks : [];
    return [...new Set(asks.slice(0, MAX_NAMED_ASK_REFERENCES).flatMap((item) => {
      if (!item || typeof item !== "object") return [];
      const value = item as Record<string, unknown>;
      const id = value.askId ?? value.ask_id ?? value.id;
      return typeof id === "string" && id.trim().length > 0 ? [id] : [];
    }))];
  }, [templateData]);
  const namedAsksTruncated = Array.isArray((templateData as Record<string, unknown> | undefined)?.asks)
    && ((templateData as Record<string, unknown>).asks as unknown[]).length > MAX_NAMED_ASK_REFERENCES;
  const namedAskQueries = useQueries({ queries: namedAskIds.map((id) => ({
    queryKey: ["ask", id, subject],
    queryFn: async () => (await conversationsClient.getAsk({ askId: id })).ask,
    staleTime: 30_000,
  })) });
  const namedAsksContext = useMemo(() => Object.fromEntries(namedAskIds.map((id, index) => {
    const ask = namedAskQueries[index]?.data;
    return [id, ask ? { state: ask.state, askedAt: ask.createdAt, answeredAt: ask.answeredAt, question: ask.question } : { state: "unknown", question: "" }];
  })), [namedAskIds, namedAskQueries]);
  const kitPage = useMemo(() => templateName ? renderKitTemplate(templateName, templateData) : null, [templateName, templateData]);
  const kitErrorMessages: Record<KitTemplateError, string> = {
    noData: t(strings.packet.kitTemplate.noData),
    unavailable: t(strings.packet.kitTemplate.unavailable),
    schemaUnreadable: t(strings.packet.kitTemplate.schemaUnreadable),
    dataInvalid: t(strings.packet.kitTemplate.dataInvalid),
    slotCount: t(strings.packet.kitTemplate.slotCount),
    tooLarge: t(strings.packet.kitTemplate.tooLarge),
  };
  const pageHtml = templateName && !explainer ? kitPage?.html || "" : page.data?.html || "";
  const frame = useRef<HTMLIFrameElement>(null);
  const [height, setHeight] = useState(MIN_FRAME_HEIGHT * 2);
  const [noteTarget, setNoteTarget] = useState<{ key: string; label: string } | null>(null);
  // The theme is part of the document only when it is built. A later switch is
  // posted to the running page, so toggling dark mode never reloads it.
  const initialTheme = useRef(theme);
  const srcDoc = useMemo(
    () =>
      pageHtml
        ? buildExplainerDocument(pageHtml, {
            kitCss,
            kitJs,
            bridgeJs: frameBridgeJs,
            theme: initialTheme.current,
            dataUrls: templateName ? new Map<string, string>() : page.data?.dataUrls ?? new Map<string, string>(),
            deferred: templateName ? [] : page.data?.deferred ?? [],
            context: { kind: kind ?? "decision", recurrence: recurrence ?? { count: 0, dates: [], previousAnswers: [] }, asks: namedAsksContext, asksTruncated: namedAsksTruncated },
          })
        : "",
    [page.data, pageHtml, templateName, kind, recurrence, namedAsksContext, namedAsksTruncated],
  );

  // The frame's own color-scheme is the theme the page is shown in, so the
  // page's prefers-color-scheme and the canvas behind it agree with its colours.
  const scheme = useMemo(() => (pageHtml ? pageColorScheme(pageHtml, kitCss) : "both"), [pageHtml]);
  const frameTheme = pageTheme(scheme, theme);

  const known = useRef(attachments);
  useEffect(() => {
    known.current = attachments;
  }, [attachments]);

  const resolveAsset = useCallback(async (request: Extract<import("./bridge").BridgeMessage, { type: "asset" }>) => {
    const reply = (dataUrl: string | null) => frame.current?.contentWindow?.postMessage({ hub: BRIDGE_PROTOCOL, type: "asset-resolve", requestId: request.requestId, dataUrl }, "*");
    const attachment = known.current.find((item) => item.name === request.name);
    const entry = attachment?.entries.find((item) => item.path === request.path);
    if (!attachment || !entry || attachment.kind !== AskAttachment_Kind.FOLDER) {
      reply(null);
      return;
    }
    try {
      const blob = await (await fetchAttachment(askId, request.name, request.path)).blob();
      if (blob.size > MAX_DYNAMIC_ASSET_BYTES) {
        reply(null);
        return;
      }
      reply(await blobToDataUrl(blob, entry.mediaType));
    } catch {
      reply(null);
    }
  }, [askId]);

  useEffect(() => {
    const onMessage = (event: MessageEvent) => {
      if (event.source === frame.current?.contentWindow && event.data?.type === "hub:answer-capability" && typeof event.data.interactive === "boolean") {
        onAnswerCapability?.(event.data.interactive);
        return;
      }
      const message = parseBridgeMessage(event, frame.current?.contentWindow ?? null);
      if (!message) return;
      if (message.type === "swipe") window.dispatchEvent(new CustomEvent("notification-hub:packet-swipe", { detail: message.direction }));
      else if (message.type === "height") setHeight(message.height);
      else if (message.type === "open") {
        if (known.current.some((attachment) => attachment.name === message.name)) onOpen(message.name, message.path);
      } else if (message.type === "open-ask") onOpenAsk?.(message.askId);
      else if (message.type === "note") setNoteTarget({ key: message.key, label: message.label });
      else if (message.type === "asset") void resolveAsset(message);
      else if (message.type === "prefill") onPrefill(message.questionId, message.value);
      else onAnswerChange?.(message);
    };
    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, [onOpen, onOpenAsk, onPrefill, onAnswerChange, onAnswerCapability, resolveAsset]);

  useEffect(() => {
    frame.current?.contentWindow?.postMessage({ hub: BRIDGE_PROTOCOL, type: "theme", theme }, "*");
  }, [theme]);

  useEffect(() => {
    const remove = (event: Event) => {
      const detail = (event as CustomEvent<{ askId: string; key: string }>).detail;
      if (detail.askId === askId && typeof detail.key === "string") frame.current?.contentWindow?.postMessage({ hub: BRIDGE_PROTOCOL, type: "answer-remove", key: detail.key }, "*");
    };
    window.addEventListener("hub:answer-remove", remove);
    return () => window.removeEventListener("hub:answer-remove", remove);
  }, [askId]);

  return (
    // On a phone the page runs the full width of the screen, so its own text sits
    // at the same gutter as the question above it; a panel with a frame, padding
    // and a header stacked the gutters until the text started a tenth of the way in.
    // The page is the main content: no label strip above it. It is still sealed
    // (sandboxed, no network, no form submission); the frame title says whose page it is.
    <section data-testid={selectors.packet.explainer} aria-label={t(strings.packet.explainer.label)} className="-mx-[var(--page-gutter)] overflow-hidden border-y border-app-border bg-app-surface md:mx-0 md:rounded-panel md:border">
      {templateName && kitPage?.error && <p role="alert" className="px-[var(--page-gutter)] py-4 text-sm text-app-danger md:px-3">{kitErrorMessages[kitPage.error]}</p>}
      {!templateName && page.isPending && <p role="status" className="px-[var(--page-gutter)] py-6 text-sm text-app-muted-foreground md:px-3">{t(strings.packet.explainer.loading)}</p>}
      {!templateName && page.isError && (
        <div role="alert" className="flex flex-col items-start gap-3 px-[var(--page-gutter)] py-4 text-sm md:px-3">
          <p>{t(strings.packet.explainer.failed)}</p>
          <div className="flex flex-wrap gap-2">
            <Button type="button" variant="outline" onClick={() => void page.refetch()}>
              {t(strings.packet.explainer.retry)}
            </Button>
            <Button type="button" variant="outline" onClick={() => explainer && onOpen(explainer.name, "")}>
              {t(strings.packet.explainer.openFile)}
            </Button>
          </div>
        </div>
      )}
      {srcDoc && (
        <iframe
          ref={frame}
          onLoad={() => frame.current?.contentWindow?.postMessage({ hub: BRIDGE_PROTOCOL, type: "answer-restore", changes: answerChanges ?? {} }, "*")}
          title={t(strings.packet.explainer.frameTitle, { name: explainer?.label || explainer?.name || templateName || "page" })}
          sandbox="allow-scripts"
          // The kit's Copy button for a command the operator must run; without
          // it the kit selects the text so a long-press copies it.
          allow="clipboard-write"
          referrerPolicy="no-referrer"
          srcDoc={srcDoc}
          style={{ height, colorScheme: frameTheme }}
          data-page-scheme={scheme}
          className="block w-full border-0 bg-transparent"
        />
      )}
      {noteTarget && <ItemNoteSheet askId={askId} target={noteTarget} current={readItemNote(answerChanges, noteTarget.key)} onClose={() => setNoteTarget(null)} onSave={(value) => {
        const existing = readItemNotes(answerChanges);
        const { [noteTarget.key]: _removed, ...remaining } = existing;
        const next = !value.text.trim() && value.attachments.length === 0
          ? remaining
          : { ...remaining, [noteTarget.key]: { label: noteTarget.label, text: value.text, attachments: value.attachments } };
        onAnswerChange?.({ type: "answer", operation: "set", key: "notes", label: `Notes · ${Object.keys(next).length}`, value: next });
        setNoteTarget(null);
      }} />}
    </section>
  );
}

function blobToDataUrl(blob: Blob, mediaType: string): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => reject(reader.error ?? new Error("could not read attachment"));
    reader.onload = () => resolve(typeof reader.result === "string" ? reader.result : "");
    reader.readAsDataURL(blob.type ? blob : new Blob([blob], { type: mediaType }));
  });
}

type NoteAttachment = { name: string; sha256: string; mediaType: string; sizeBytes: number };
type ItemNote = { label: string; text: string; attachments: NoteAttachment[] };
type ItemNotes = Record<string, ItemNote>;

function readItemNotes(changes: ExplainerFrameProps["answerChanges"]): ItemNotes {
  const value = changes?.notes?.value;
  return value && typeof value === "object" && !Array.isArray(value) ? value as ItemNotes : {};
}

function readItemNote(changes: ExplainerFrameProps["answerChanges"], key: string): ItemNote {
  const value = readItemNotes(changes)[key];
  return value && typeof value.text === "string" && Array.isArray(value.attachments) ? value : { label: "", text: "", attachments: [] };
}

async function digest(file: File): Promise<string> {
  const bytes = new Uint8Array(await file.arrayBuffer());
  const hash = await crypto.subtle.digest("SHA-256", bytes);
  return [...new Uint8Array(hash)].map((byte) => byte.toString(16).padStart(2, "0")).join("");
}

function ItemNoteSheet({ askId, target, current, onClose, onSave }: { askId: string; target: { key: string; label: string }; current: ItemNote; onClose: () => void; onSave: (value: { text: string; attachments: NoteAttachment[] }) => void }) {
  const { t } = useTranslation();
  const [text, setText] = useState(current.text);
  const [attachments, setAttachments] = useState(current.attachments);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const addFiles = async (files: FileList | null) => {
    if (!files?.length) return;
    setBusy(true);
    setError("");
    try {
      const additions: NoteAttachment[] = [];
      for (const file of Array.from(files)) {
      if (!file.size || file.size > 12 * 1024 * 1024) throw new Error(t(strings.packet.noteSheet.fileTooLarge, { name: file.name }));
        const sha256 = await digest(file);
        const content = new Uint8Array(await file.arrayBuffer());
        await conversationsClient.putAttachmentBlob({ sha256, mediaType: file.type || "application/octet-stream", content });
        additions.push({ name: file.name.slice(0, 96), sha256, mediaType: file.type || "application/octet-stream", sizeBytes: file.size });
      }
      setAttachments((currentFiles) => [...currentFiles, ...additions].slice(0, 8));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t(strings.packet.noteSheet.fileError));
    } finally {
      setBusy(false);
    }
  };
  return <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/50 p-3 sm:items-center" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose(); }}>
    <section role="dialog" aria-modal="true" aria-labelledby="item-note-title" className="flex w-full max-w-lg flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4 text-app-foreground shadow-xl">
      <h2 id="item-note-title" className="text-lg font-semibold">{t(strings.packet.noteSheet.title, { label: target.label })}</h2>
      <label className="flex flex-col gap-1 text-sm font-medium">{t(strings.packet.noteSheet.yourNote)}<textarea rows={4} maxLength={4000} value={text} onChange={(event) => setText(event.target.value)} className="min-h-24 rounded-control border border-app-border bg-app-surface px-3 py-2 text-app-foreground" /></label>
      <label className="flex flex-col gap-1 text-sm font-medium">{t(strings.packet.noteSheet.attach)}<input type="file" multiple disabled={busy || attachments.length >= 8} onChange={(event) => { void addFiles(event.currentTarget.files); event.currentTarget.value = ""; }} className="rounded-control border border-app-border bg-app-surface p-2 text-sm" /></label>
      {attachments.length > 0 && <ul className="flex flex-col gap-1 text-sm">{attachments.map((file) => <li key={file.sha256} className="flex items-center justify-between gap-2"><span className="truncate">{file.name}</span><button type="button" className="min-h-9 px-2 text-app-primary" onClick={() => setAttachments((currentFiles) => currentFiles.filter((item) => item.sha256 !== file.sha256))}>{t(strings.packet.noteSheet.remove)}</button></li>)}</ul>}
      {error && <p role="alert" className="text-sm text-app-danger">{error}</p>}
      <div className="flex justify-end gap-2"><button type="button" className="min-h-11 rounded-control border border-app-border px-4" onClick={onClose}>{t(strings.packet.noteSheet.cancel)}</button><button type="button" disabled={busy} className="min-h-11 rounded-control bg-app-primary px-4 font-semibold text-white disabled:opacity-60" onClick={() => onSave({ text, attachments })}>{busy ? t(strings.packet.noteSheet.uploading) : t(strings.packet.noteSheet.save)}</button></div>
      <span className="sr-only">{t(strings.packet.noteSheet.ask, { id: askId })}</span>
    </section>
  </div>;
}
