import { PREVIEW_KINDS, type ManifestFile, type PreviewKind } from "@vrooli/file-viewer";
import type { AskAttachment } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { attachmentUrl } from "../../../api/attachments";
import type { LinkedArtifact } from "../../../api/attachments";
import { isExplainer, isFolder, previewKindOf } from "./attachments";

/** The server's classification as the viewer's kind; anything it does not know is offered as a download. */
function viewerKind(previewKind: string, mediaType: string): PreviewKind {
  const kind = previewKindOf(previewKind, mediaType);
  return PREVIEW_KINDS.find((known) => known === kind) ?? "unsupported";
}

/**
 * Every evidence file the operator can open, as the viewer's manifest: a file
 * attachment sits at `name`, a folder's files at `name/relative/path`. The
 * sealed explainer page is shown in its own frame, not here. URLs are the
 * Hub's own copies on the content route (`?path=` addresses a folder entry),
 * so the viewer never reads a path on the requester's host.
 */
export function packetManifest(askId: string, attachments: readonly AskAttachment[], linkedArtifact?: LinkedArtifact): ManifestFile[] {
  const manifest = attachments.filter((attachment) => !isExplainer(attachment)).flatMap((attachment) => {
    if (!isFolder(attachment)) {
      return [{
        path: attachment.name,
        url: attachmentUrl(askId, attachment.name),
        kind: viewerKind(attachment.previewKind, attachment.mediaType),
        mimeType: attachment.mediaType,
        sizeBytes: Number(attachment.sizeBytes),
      }];
    }
    return attachment.entries.map((entry) => ({
      path: `${attachment.name}/${entry.path}`,
      url: attachmentUrl(askId, attachment.name, entry.path),
      kind: viewerKind(entry.previewKind, entry.mediaType),
      mimeType: entry.mediaType,
      sizeBytes: Number(entry.sizeBytes),
    }));
  });
  if (linkedArtifact) manifest.push({
    path: `linked-artifact/${linkedArtifact.name}`,
    url: linkedArtifact.contentUrl,
    kind: viewerKind(linkedArtifact.kind, linkedArtifact.mimeType),
    mimeType: linkedArtifact.mimeType,
    sizeBytes: linkedArtifact.sizeBytes,
  });
  return manifest;
}
