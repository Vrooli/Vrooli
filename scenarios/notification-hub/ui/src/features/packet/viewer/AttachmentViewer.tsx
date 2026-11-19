import { useEffect, useMemo, useState } from "react";
import { FileViewer, FileViewerProvider, createManifestPreviewSource, formatBytes, useFilePreviewController, type ViewerActionContext } from "@vrooli/file-viewer";
import "@vrooli/file-viewer/styles.css";
import { Button } from "@vrooli/react-component-library/Button/2";
import type { AskAttachment } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { ChevronLeft, ChevronRight, Download, ExternalLink, MessageSquarePlus, Pin } from "lucide-react";

import { sessionFetch } from "../../../api/notifications";
import { fetchAttachment, type LinkedArtifact } from "../../../api/attachments";
import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { TextArea } from "../form/TextArea";
import { attachmentNoteFor, withAttachmentNote, type PacketDraft } from "../model/draft";
import { ANSWER_LIMITS } from "../model/questions";
import { evidenceAttachments, targetPath, type AttachmentTarget } from "./attachments";
import { isRegionFeedbackFolder, parseRegionFeedbackBundle, REGION_FEEDBACK_MANIFEST, validateRegionEntries, type ParsedRegionFeedback } from "./regionFeedback";
import { RegionFeedbackCompare } from "./RegionFeedbackCompare";
import { packetManifest } from "./manifest";
import { viewerLoaders } from "./viewerLoaders";

interface AttachmentViewerProps {
  askId: string;
  attachments: readonly AskAttachment[];
  linkedArtifact?: LinkedArtifact;
  /** Where the viewer opens: a file, a folder, or a heading inside a Markdown file. */
  target: AttachmentTarget;
  onClose: () => void;
  draft: PacketDraft;
  onDraftChange: (next: PacketDraft) => void;
  /** The operator can still answer, so they can pin notes to what they are looking at. */
  canNote: boolean;
}

/**
 * Looks at the packet's files inside the Hub with the shared file viewer, which
 * renders every kind of file and browses folders. The Hub supplies what is
 * specific to a decision: the files themselves (its own stored copies, served
 * on the content route), moving between attachments (beside the title), and
 * one quiet bar under the file with its format and size, a download, a warning
 * when the original changed since, and "Comment here", which pins a note to the
 * file on screen so it travels with the answer. The viewer's own path line and
 * file-info bar are off: the path is a location on the requester's host, which
 * the operator cannot use.
 */
export function AttachmentViewer({ askId, attachments, linkedArtifact, target, onClose, draft, onDraftChange, canNote }: AttachmentViewerProps) {
  const { i18n } = useTranslation();
  // The scope is the ask: a refetch that builds new attachment objects keeps the open file.
  const source = useMemo(
    () => createManifestPreviewSource({ scope: askId, files: packetManifest(askId, attachments, linkedArtifact), fetch: sessionFetch }),
    [askId, attachments, linkedArtifact],
  );
  const controller = useFilePreviewController(source);
  const { openPreview, close } = controller;
  const { name, path, anchor } = target;
  const regionFolder = attachments.find((item) => item.name === name && isRegionFeedbackFolder(item));
  const isRegionManifest = Boolean(regionFolder && path === REGION_FEEDBACK_MANIFEST);
  useEffect(() => {
    if (isRegionManifest) return close;
    void openPreview(targetPath({ name, path }), undefined, anchor ? { anchor } : undefined);
    return close;
  }, [openPreview, close, name, path, anchor, isRegionManifest]);

  const evidence = useMemo(() => evidenceAttachments(attachments), [attachments]);
  const [regionFeedback, setRegionFeedback] = useState<ParsedRegionFeedback | null>(null);
  const [regionFeedbackError, setRegionFeedbackError] = useState(false);
  useEffect(() => {
    setRegionFeedback(null);
    setRegionFeedbackError(false);
    if (!isRegionManifest || !regionFolder) return;
    const abort = new AbortController();
    void fetchAttachment(askId, regionFolder.name, REGION_FEEDBACK_MANIFEST, abort.signal)
      .then(async (response) => {
        const contentLength = Number(response.headers.get("content-length") || "0");
        if (contentLength > 1 << 20) throw new Error("region manifest exceeds 1 MiB");
        const content = await response.text();
        if (content.length > 1 << 20) throw new Error("region manifest exceeds 1 MiB");
        const raw: unknown = JSON.parse(content);
        const parsed = parseRegionFeedbackBundle(raw);
        if (!parsed) throw new Error("invalid region feedback manifest");
        parsed.issues.push(...validateRegionEntries(parsed.bundle, regionFolder));
        setRegionFeedback(parsed);
      })
      .catch(() => { if (!abort.signal.aborted) setRegionFeedbackError(true); });
    return () => abort.abort();
  }, [askId, isRegionManifest, regionFolder]);
  const onOpen = (attachmentName: string) => void openPreview(attachmentName);
  const renderHeaderActions = (context: ViewerActionContext) => <ViewerPosition context={context} attachments={evidence} onOpen={onOpen} />;
  const renderFooter = (context: ViewerActionContext) => (
    <ViewerFooter
      // A new file starts with the note closed.
      key={context.path}
      context={context}
      attachments={evidence}
      canNote={canNote}
      draft={draft}
      onDraftChange={onDraftChange}
    />
  );

  if (isRegionManifest && regionFolder) {
    return (
      <FileViewerProvider locale={i18n.resolvedLanguage} loaders={viewerLoaders} fileInfo="host">
        <RegionFeedbackCompare
          askId={askId}
          attachment={regionFolder}
          parsed={regionFeedback}
          failed={regionFeedbackError}
          onClose={onClose}
          draft={draft}
          onDraftChange={onDraftChange}
          canNote={canNote}
        />
      </FileViewerProvider>
    );
  }

  return (
    <FileViewerProvider locale={i18n.resolvedLanguage} loaders={viewerLoaders} fileInfo="host">
      <FileViewer
        source={source}
        controller={controller}
        showPath={false}
        renderHeaderActions={renderHeaderActions}
        renderFooter={renderFooter}
        onClose={() => {
          close();
          onClose();
        }}
      />
    </FileViewerProvider>
  );
}

/** Where the viewer is, as the packet knows it: the attachment, and inside a folder the file. */
function locate(viewerPath: string, attachments: readonly AskAttachment[]) {
  const [name = "", ...rest] = viewerPath.split("/").filter(Boolean);
  const index = attachments.findIndex((attachment) => attachment.name === name);
  return { name, entry: rest.join("/"), index, attachment: attachments[index] };
}

interface ViewerPositionProps {
  context: ViewerActionContext;
  attachments: readonly AskAttachment[];
  onOpen: (attachmentName: string) => void;
}

/** Previous and next attachment, with "n of N", beside the file's title. */
function ViewerPosition({ context, attachments, onOpen }: ViewerPositionProps) {
  const { t } = useTranslation();
  const { index, attachment } = locate(context.path, attachments);
  if (!attachment || attachments.length < 2) return null;
  const previous = attachments[index - 1];
  const next = attachments[index + 1];
  const button = "touch-target inline-flex items-center justify-center rounded-control hover:bg-app-surface-muted disabled:opacity-40";
  return (
    <div className="flex items-center">
      <button type="button" data-testid={selectors.packet.viewerPrevious} disabled={!previous} onClick={() => previous && onOpen(previous.name)} aria-label={t(strings.packet.viewer.previous)} className={button}>
        <ChevronLeft size={22} aria-hidden="true" className="rtl:rotate-180" />
      </button>
      <span className="whitespace-nowrap px-1 text-sm text-app-muted-foreground">{t(strings.packet.viewer.position, { current: index + 1, total: attachments.length })}</span>
      <button type="button" data-testid={selectors.packet.viewerNext} disabled={!next} onClick={() => next && onOpen(next.name)} aria-label={t(strings.packet.viewer.next)} className={button}>
        <ChevronRight size={22} aria-hidden="true" className="rtl:rotate-180" />
      </button>
    </div>
  );
}

/** "XLSX", or the viewer's own kind when the file has no extension. */
function formatOf(basename: string, kind: string): string {
  const dot = basename.lastIndexOf(".");
  return dot > 0 && dot < basename.length - 1 ? basename.slice(dot + 1).toUpperCase() : kind;
}

interface ViewerFooterProps {
  context: ViewerActionContext;
  attachments: readonly AskAttachment[];
  canNote: boolean;
  draft: PacketDraft;
  onDraftChange: (next: PacketDraft) => void;
}

/** One quiet bar under the file: what it is, a download, the changed warning and "Comment here". */
function ViewerFooter({ context, attachments, canNote, draft, onDraftChange }: ViewerFooterProps) {
  const { t } = useTranslation();
  const [commenting, setCommenting] = useState(false);
  const { name, entry, attachment } = locate(context.path, attachments);
  if (!attachment) return null;

  const { model } = context;
  const status = attachment.sourceStatus;
  const changed = status === "changed" || status === "missing";
  // A note and a download belong to a file, not to a folder listing.
  const file = model !== null && model.kind !== "directory" ? model : null;
  const canComment = canNote && file !== null;
  if (!changed && !file) return null;

  const note = attachmentNoteFor(draft, name, entry);
  const label = entry ? `${attachment.label || name} / ${entry}` : attachment.label || name;
  const about = file ? [attachment.label && !entry ? attachment.label : "", formatOf(file.basename, file.kind), formatBytes(file.sizeBytes)].filter(Boolean).join(" · ") : "";

  return (
    <div className="flex flex-col gap-2 bg-app-surface px-3 pt-2 pb-3">
      {changed && (
        <p role="status" className="rounded-control bg-app-warning/15 px-3 py-2 text-sm">
          {t(status === "changed" ? strings.packet.viewer.changed : strings.packet.viewer.missing)}
        </p>
      )}
      {commenting && canComment ? (
        <div className="flex flex-col gap-2">
          <TextArea
            label={t(strings.packet.viewer.commentLabel, { name: label })}
            testId={selectors.packet.viewerNote}
            value={note}
            rows={3}
            maxLength={ANSWER_LIMITS.attachmentNoteChars}
            onChange={(value) => onDraftChange(withAttachmentNote(draft, name, entry, value))}
          />
          <p className="text-xs text-app-muted-foreground">{t(strings.packet.viewer.commentHint)}</p>
          <div className="flex gap-2">
            {note && (
              <Button
                type="button"
                variant="outline"
                className="border border-app-border"
                onClick={() => {
                  onDraftChange(withAttachmentNote(draft, name, entry, ""));
                  setCommenting(false);
                }}
              >
                {t(strings.packet.viewer.commentRemove)}
              </Button>
            )}
            <Button data-testid={selectors.packet.viewerSaveNote} type="button" variant="primary" className="min-h-11 flex-1" onClick={() => setCommenting(false)}>
              {t(strings.packet.viewer.commentDone)}
            </Button>
          </div>
        </div>
      ) : (
        file && (
          <div className="flex items-center gap-1">
            <p className="min-w-0 flex-1 truncate text-xs text-app-muted-foreground" title={about}><bdi>{about}</bdi></p>
            <a href={file.blobHref} download={file.basename} aria-label={t(strings.packet.viewer.download)} title={t(strings.packet.viewer.download)} className="touch-target inline-flex shrink-0 items-center justify-center rounded-control hover:bg-app-surface-muted">
              <Download size={18} aria-hidden="true" />
            </a>
            {/* A file read full-screen in its own tab is a desktop habit; a phone has the drawer. */}
            <a href={file.blobHref} target="_blank" rel="noreferrer" data-testid={selectors.packet.viewerOpenTab} aria-label={t(strings.packet.viewer.openInNewTab)} title={t(strings.packet.viewer.openInNewTab)} className="touch-target hidden shrink-0 items-center justify-center rounded-control hover:bg-app-surface-muted md:inline-flex">
              <ExternalLink size={18} aria-hidden="true" className="rtl:-scale-x-100" />
            </a>
            {canComment && (
              <Button data-testid={selectors.packet.viewerComment} type="button" variant="outline" className="min-h-11 shrink-0 border border-app-border" onClick={() => setCommenting(true)}>
                {note ? <Pin size={16} aria-hidden="true" className="me-1.5" /> : <MessageSquarePlus size={16} aria-hidden="true" className="me-1.5" />}
                {note ? t(strings.packet.viewer.hasNote) : t(strings.packet.viewer.comment)}
              </Button>
            )}
          </div>
        )
      )}
    </div>
  );
}
