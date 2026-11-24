import { Fragment, type ReactNode } from "react";
import type { AskAttachment } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { selectors } from "../../consts/selectors";
import { Literal } from "./Literal";
import { findAttachmentByPath, type AttachmentTarget } from "./viewer/attachments";

function safeHttpUrl(value: string): string | null {
  try {
    const url = new URL(value);
    return url.protocol === "http:" || url.protocol === "https:" ? url.href : null;
  } catch {
    return null;
  }
}

function filePath(value: string): string | null {
  const path = value.trim().replace(/[.,;!?]+$/, "");
  if (!path || /\s/.test(path) || /^(?:https?:|mailto:|tel:|webapp:|data:)/i.test(path)) return null;
  if (/^file:\/\//i.test(path)) return path;
  if (/^(?:~\/|\.{1,2}\/|\/)/.test(path)) return path;
  if (!path.includes("/") && !/\.[a-z0-9]{1,8}$/i.test(path)) return null;
  const extension = path.replace(/:\d+$/, "").match(/\.([a-z0-9]{1,8})$/i)?.[1]?.toLowerCase();
  if (extension && ["com", "org", "net", "edu", "gov", "io", "ai", "co", "dev", "app", "us", "uk"].includes(extension)) return null;
  return path;
}

export function isLocalFileReference(value: string): boolean {
  return filePath(value) !== null;
}

const DECISION_LINKS = /\[([^\]]+)\]\(([^)]+)\)|`([^`\n]+)`|(https?:\/\/[^\s<>()]+)|((?:(?:file:\/\/\/?|~\/|\.{1,2}\/|\/)?[\w.@+-]+(?:\/[\w.@+-]+)*(?:\.[a-z0-9]{1,8}|\/))(?::\d+)?)/gi;
// Keep inline evidence bounded: one image may use at most 1 MiB decoded.
const EMBEDDED_IMAGE = /!\[([^\]]+)\]\((data:image\/(?:png|jpeg|webp);base64,[A-Za-z0-9+/=]{1,1398104})\)/gi;

/** The local path a link token names, or null for a web link. */
function localReference(match: RegExpMatchArray): string | null {
  return filePath(match[2] ?? match[3] ?? match[5] ?? "");
}

export interface DecisionLinks {
  attachments?: readonly AskAttachment[];
  onOpenAttachment?: (target: AttachmentTarget) => void;
  onOpenLinkedArtifact?: (path: string) => void;
  openLinkedArtifactTitle?: string;
  openAttachmentTitle?: string;
  /** Lay links out inline at the surrounding line height (headings), instead of as 44px touch targets. */
  dense?: boolean;
}

const LINK_CLASS = "inline-flex min-h-11 max-w-full items-center break-words underline";

/**
 * Render safe web links and file references in decision prose. Attached files
 * and other allowed local artifacts both open in the Hub viewer.
 */
export function renderDecisionText(value: string, links: DecisionLinks = {}): ReactNode[] {
  const { attachments = [], onOpenAttachment, onOpenLinkedArtifact, openLinkedArtifactTitle, dense = false } = links;
  const parts: ReactNode[] = [];
  let cursor = 0;
  const tokens = [
    ...Array.from(value.matchAll(new RegExp(EMBEDDED_IMAGE.source, EMBEDDED_IMAGE.flags)), (match) => ({
      kind: "image" as const,
      index: match.index,
      full: match[0],
      alt: match[1] ?? "",
      src: match[2] ?? "",
    })),
    ...Array.from(value.matchAll(new RegExp(DECISION_LINKS.source, DECISION_LINKS.flags)), (match) => ({
      kind: "link" as const,
      index: match.index,
      full: match[0],
      match,
    })),
  ].sort((a, b) => a.index - b.index);
  for (const token of tokens) {
    const { index } = token;
    // The ordinary Markdown link pattern also matches the inner [alt](src)
    // portion of an image token. Ignore that overlapping text token.
    if (index < cursor) continue;
    if (index > cursor) parts.push(value.slice(cursor, index));
    cursor = index + token.full.length;
    if (token.kind === "image") {
      parts.push(
        <img
          key={`image-${index}`}
          src={token.src}
          alt={token.alt}
          loading="lazy"
          decoding="async"
          className="my-3 block h-auto max-h-[60vh] max-w-full rounded-control border border-app-border object-contain"
        />,
      );
      continue;
    }
    const { match } = token;
    const text = match[1] ?? match[3] ?? match[4] ?? match[5] ?? "";
    // The sender's own words stay prose; a bare path, URL or code span is shown as the literal it is.
    const label = match[1] === undefined ? <Literal inHeading={dense}>{text}</Literal> : text;
    const httpUrl = safeHttpUrl(match[2] ?? match[3] ?? match[4] ?? "");
    const localPath = httpUrl ? null : localReference(match);
    const attachment = localPath && onOpenAttachment ? findAttachmentByPath(attachments, localPath) : null;
    if (attachment && onOpenAttachment) {
      parts.push(
        <button
          key={`attachment-${index}`}
          type="button"
          data-testid={selectors.packet.textLink}
          title={links.openAttachmentTitle}
          onClick={() => onOpenAttachment(attachment)}
          className={`${LINK_CLASS} text-start`}
        >
          {label}
        </button>,
      );
      continue;
    }
    if (localPath && onOpenLinkedArtifact) {
      parts.push(
        <button key={`linked-artifact-${index}`} type="button" data-testid={selectors.packet.textLink} title={openLinkedArtifactTitle} onClick={() => onOpenLinkedArtifact(localPath)} className={`${LINK_CLASS} text-start`}>
          {label}
        </button>,
      );
      continue;
    }
    const href = httpUrl;
    parts.push(
      href ? (
        <a key={`link-${index}`} href={href} data-testid={selectors.packet.textLink} target="_blank" rel="noopener noreferrer" className={LINK_CLASS}>
          {label}
        </a>
      ) : (
        <Fragment key={`text-${index}`}>{label}</Fragment>
      ),
    );
  }
  if (cursor < value.length) parts.push(value.slice(cursor));
  return parts;
}

/** The link targets a text surface needs; the component supplies the localized titles. */
export type TextLinks = Omit<DecisionLinks, "openAttachmentTitle" | "openLinkedArtifactTitle">;
