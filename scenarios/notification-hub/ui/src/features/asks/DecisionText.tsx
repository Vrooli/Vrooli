import type { ReactNode } from "react";

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

function fileViewerHref(base: string, path: string): string | null {
  try {
    const url = new URL(base);
    if (url.protocol !== "https:" && url.protocol !== "http:") return null;
    url.searchParams.set("file", path);
    return url.href;
  } catch {
    return null;
  }
}

/** Resolve one local artifact path into Web Console's existing preview entry. */
export function fileReferenceHref(value: string, webConsoleUrl = ""): string | null {
  const path = filePath(value);
  return path && webConsoleUrl ? fileViewerHref(webConsoleUrl, path) : null;
}

const DECISION_LINKS = /\[([^\]]+)\]\(([^)]+)\)|`([^`\n]+)`|(https?:\/\/[^\s<>()]+)|((?:(?:file:\/\/\/?|~\/|\.{1,2}\/|\/)?[\w.@+-]+(?:\/[\w.@+-]+)*(?:\.[a-z0-9]{1,8}|\/))(?::\d+)?)/gi;
// Keep inline evidence bounded: one image may use at most 1 MiB decoded.
const EMBEDDED_IMAGE = /!\[([^\]]+)\]\((data:image\/(?:png|jpeg|webp);base64,[A-Za-z0-9+/=]{1,1398104})\)/gi;

export function hasLocalFileReference(value: string): boolean {
  const matches = value.matchAll(new RegExp(DECISION_LINKS.source, DECISION_LINKS.flags));
  for (const match of matches) {
    if (filePath(match[2] ?? match[3] ?? match[5] ?? "")) return true;
  }
  return false;
}

/**
 * Render safe web links and local file references in decision prose. Local
 * paths hand off to Web Console's existing file preview surface; when that
 * sibling is unavailable, the path stays visible instead of becoming a dead
 * link. Other Markdown remains plain text by design.
 */
export function renderDecisionText(value: string, webConsoleUrl = ""): ReactNode[] {
  const parts: ReactNode[] = [];
  let cursor = 0;
  const tokens = [
    ...Array.from(value.matchAll(new RegExp(EMBEDDED_IMAGE.source, EMBEDDED_IMAGE.flags)), (match) => ({
      kind: "image" as const,
      index: match.index!,
      full: match[0],
      alt: match[1],
      src: match[2],
    })),
    ...Array.from(value.matchAll(new RegExp(DECISION_LINKS.source, DECISION_LINKS.flags)), (match) => ({
      kind: "link" as const,
      index: match.index!,
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
      cursor = index + token.full.length;
      continue;
    }
    const { match } = token;
    const markdownLabel = match[1];
    const destination = match[2];
    const codeToken = match[3];
    const bareUrl = match[4];
    const barePath = match[5];
    const label = markdownLabel ?? codeToken ?? bareUrl ?? barePath ?? "";
    const httpUrl = safeHttpUrl(destination ?? codeToken ?? bareUrl ?? "");
    const localPath = httpUrl ? null : filePath(destination ?? codeToken ?? barePath ?? "");
    const fileHref = localPath && webConsoleUrl ? fileViewerHref(webConsoleUrl, localPath) : null;
    const href = httpUrl ?? fileHref;
    parts.push(href
      ? <a key={`link-${index}`} href={href} target="_blank" rel="noopener noreferrer" title={fileHref ? "Open file in Web Console" : undefined} className="inline-flex min-h-11 max-w-full items-center break-all underline">{label}</a>
      : label);
    cursor = index + token.full.length;
  }
  if (cursor < value.length) parts.push(value.slice(cursor));
  return parts;
}
