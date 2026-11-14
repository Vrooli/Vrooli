import {
  AskAttachment_Kind,
  AskAttachment_Role,
  type AskAttachment,
} from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

/** Where the viewer is: one attachment, and for a folder the file inside it. */
export interface AttachmentTarget {
  name: string;
  path: string;
  /** A heading in the file to land on (Markdown), written as the requester's CLI slugs it. */
  anchor?: string;
}

export const isFolder = (attachment: AskAttachment) => attachment.kind === AskAttachment_Kind.FOLDER;
export const isExplainer = (attachment: AskAttachment) => attachment.role === AskAttachment_Role.EXPLAINER;

/** Everything except the sealed explainer page, which has its own place at the top. */
export const evidenceAttachments = (attachments: readonly AskAttachment[]): AskAttachment[] =>
  attachments.filter((attachment) => !isExplainer(attachment));

/** The explainer attachment, when the ask has one. */
export const explainerOf = (attachments: readonly AskAttachment[]): AskAttachment | undefined =>
  attachments.find(isExplainer);

export const attachmentLabel = (attachment: AskAttachment): string => attachment.label || attachment.name;

/** The path the viewer opens for a target: `name` for a file, `name/entry` inside a folder. */
export const targetPath = (target: AttachmentTarget): string => (target.path ? `${target.name}/${target.path}` : target.name);

const stripLocation = (value: string): string =>
  value
    .trim()
    .replace(/^file:\/\//i, "")
    .replace(/:\d+(?::\d+)?$/, "")
    .replace(/^\.\//, "");

const baseName = (value: string): string => value.slice(value.lastIndexOf("/") + 1);

/**
 * The attachment a path in the decision text refers to, or null. Text written
 * on the requester's host names files by their original path, so a path is
 * matched against each attachment's name and `source_path`, and against files
 * inside a folder attachment. Ambiguous matches resolve to nothing: sending the
 * operator to the wrong file is worse than falling back to Web Console.
 */
export function findAttachmentByPath(attachments: readonly AskAttachment[], raw: string): AttachmentTarget | null {
  const path = stripLocation(raw);
  if (!path) return null;

  const named = attachments.find((attachment) => attachment.name === path);
  if (named) return { name: named.name, path: "" };
  const inFolder = path.indexOf("/");
  if (inFolder > 0) {
    const folder = attachments.find((attachment) => attachment.name === path.slice(0, inFolder) && isFolder(attachment));
    const entry = folder?.entries.find((item) => item.path === path.slice(inFolder + 1));
    if (folder && entry) return { name: folder.name, path: entry.path };
  }

  const exact = attachments.filter((attachment) => attachment.sourcePath === path);
  if (exact.length === 1 && exact[0]) return { name: exact[0].name, path: "" };

  const inside = attachments.flatMap((attachment) => {
    if (!isFolder(attachment) || !attachment.sourcePath || !path.startsWith(`${attachment.sourcePath}/`)) return [];
    const relative = path.slice(attachment.sourcePath.length + 1);
    return attachment.entries.some((entry) => entry.path === relative) ? [{ name: attachment.name, path: relative }] : [];
  });
  if (inside.length === 1 && inside[0]) return inside[0];

  // A relative or bare reference ("docs/plan.md", "plan.md") that is the tail
  // of exactly one attachment's original path.
  const tails = attachments.filter(
    (attachment) =>
      Boolean(attachment.sourcePath) &&
      !isFolder(attachment) &&
      (attachment.sourcePath.endsWith(`/${path}`) || (!path.includes("/") && baseName(attachment.sourcePath) === path)),
  );
  return tails.length === 1 && tails[0] ? { name: tails[0].name, path: "" } : null;
}

const KIND_BY_PREFIX: readonly [string, string][] = [
  ["image/", "image"],
  ["video/", "video"],
  ["audio/", "audio"],
];

/** The server's classification, with a conservative fallback when it is missing or unknown. */
export function previewKindOf(previewKind: string, mediaType: string): string {
  if (previewKind) return previewKind;
  return KIND_BY_PREFIX.find(([prefix]) => mediaType.startsWith(prefix))?.[1] ?? "unsupported";
}

/** An attachment reference resolved to the file it shows. */
export interface ResolvedAttachment {
  attachment: AskAttachment;
  target: AttachmentTarget;
  kind: string;
  mediaType: string;
}

/**
 * Resolve a reference such as a choice's `attachment` (`name`, or
 * `folder/entry/path`) to its file. Returns null when it names nothing, so a
 * bad reference from a requester degrades to a choice without a picture.
 */
export function resolveAttachmentRef(attachments: readonly AskAttachment[], reference: string): ResolvedAttachment | null {
  const target = findAttachmentByPath(attachments, reference);
  const attachment = target && attachments.find((item) => item.name === target.name);
  if (!target || !attachment) return null;
  const entry = target.path ? attachment.entries.find((item) => item.path === target.path) : undefined;
  return {
    attachment,
    target,
    kind: previewKindOf(entry?.previewKind ?? attachment.previewKind, entry?.mediaType ?? attachment.mediaType),
    mediaType: entry?.mediaType ?? attachment.mediaType,
  };
}

/** Kinds a plain `<img>` can show inline without running anything. */
export const isInlineImage = (kind: string): boolean => kind === "image" || kind === "svg";
