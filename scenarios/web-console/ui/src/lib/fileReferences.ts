const externalSchemes = /^(https?:\/\/|mailto:|tel:)/i;

export function isExternalHref(href: string): boolean {
  return externalSchemes.test(href.trim());
}

export function looksLikeFileReference(href: string): boolean {
  const value = href.trim();
  if (!value || isExternalHref(value) || value.startsWith("#")) {
    return false;
  }
  if (value.startsWith("file://")) {
    return true;
  }
  if (value.startsWith("/")) {
    return hasFileLikeShape(value);
  }
  return hasFileLikeShape(value);
}

function hasFileLikeShape(value: string): boolean {
  const withoutLine = value.replace(/:\d+$/, "");
  return (
    withoutLine.includes("/") ||
    withoutLine.includes("\\") ||
    /\.[a-z0-9]+$/i.test(withoutLine)
  );
}

// ---------------------------------------------------------------------------
// Prose path detection — bare paths in running text, with no backticks or
// link syntax to signal intent. Deliberately much stricter than
// looksLikeFileReference (which only filters already-marked-up references):
// a token must be anchored (/, ~/, ./, ../, file://) with 2+ segments or an
// extension, or be relative with 2+ segments AND an extension. This keeps
// `and/or`, `TCP/IP`, `node.js`, and bare domains out.
// ---------------------------------------------------------------------------

export interface ProsePathMatch {
  /** Index of the first character of the path within the input text. */
  start: number;
  /** Index just past the last character of the path. */
  end: number;
  /** The matched path token, including any trailing `:line` suffix. */
  path: string;
}

// Candidate tokens: runs of path-ish characters with at least one slash,
// optionally prefixed by file:// / ~ / . / .. and suffixed by :line.
const PROSE_PATH_CANDIDATE = /(?:file:\/\/|~|\.{1,2})?\/?[\w.@+-]+(?:\/[\w.@+-]+)*\/?(?::\d+)?/g;

const EXTENSION_RE = /\.[A-Za-z0-9]{1,8}$/;

function isStrictProsePath(token: string, trailingSlash = false): boolean {
  const withoutLine = token.replace(/:\d+$/, "");
  const anchored = /^(?:file:\/\/|~\/|\.{1,2}\/|\/)/.test(withoutLine);
  const body = withoutLine.replace(/^(?:file:\/\/|~\/|\.{1,2}\/|\/)/, "").replace(/\/$/, "");
  if (!body) return false;
  const segments = body.split("/").filter(Boolean);
  const lastSegment = segments[segments.length - 1] ?? "";
  if (anchored) {
    return segments.length >= 2 || EXTENSION_RE.test(lastSegment);
  }
  // Unanchored relative paths normally need both depth and an extension —
  // the guard that keeps `and/or` and `TCP/IP` out of the link layer.
  //
  // An explicit trailing slash is the one exception: it is an unambiguous
  // "this is a directory" signal that running prose never produces by
  // accident, so `docs/evidence/` qualifies where `docs/evidence` cannot.
  // Directories are previewable, so without this the same directory would be
  // a link when written one way and dead text when written another.
  if (trailingSlash) {
    return segments.length >= 2;
  }
  return segments.length >= 2 && EXTENSION_RE.test(lastSegment);
}

// Extensions that are far more likely to be a bare domain than a filename
// when they appear on a single slash-less token (`vrooli.com`).
const BARE_DOMAIN_TLDS = new Set(["com", "org", "net", "edu", "gov", "io", "ai", "co", "dev", "app", "us", "uk"]);

/**
 * Whether backticked inline-code text should offer click-to-preview. Stricter
 * than looksLikeFileReference (which serves authored [text](href) links where
 * the link syntax itself signals intent): the whole token must be a strict
 * path per matchProseFilePaths, or a single slash-less filename with an
 * extension (`README.md`) that doesn't look like a bare domain.
 */
export function looksLikeInlineFileReference(text: string): boolean {
  const value = text.trim();
  if (!value || /\s/.test(value) || isExternalHref(value)) return false;
  const matches = matchProseFilePaths(value);
  const only = matches.length === 1 ? matches[0] : undefined;
  // matchProseFilePaths reports the path without a trailing slash, so a
  // directory written as `docs/evidence/` matches one character short of the
  // full token. Compare against the slash-trimmed value so it still counts as
  // covering the whole chip.
  const covered = value.replace(/\/+$/, "").length;
  if (only && only.start === 0 && only.end === covered) {
    return true;
  }
  if (value.includes("/") || value.includes("\\")) return false;
  const ext = value.replace(/:\d+$/, "").match(/\.([A-Za-z0-9]{1,8})$/)?.[1]?.toLowerCase();
  return !!ext && !BARE_DOMAIN_TLDS.has(ext);
}

/**
 * Finds strongly path-shaped tokens in plain prose. Returns non-overlapping
 * matches in order. Callers are expected to have already excluded code spans
 * and explicit links.
 */
export function matchProseFilePaths(text: string): ProsePathMatch[] {
  const matches: ProsePathMatch[] = [];
  PROSE_PATH_CANDIDATE.lastIndex = 0;
  let m: RegExpExecArray | null;
  while ((m = PROSE_PATH_CANDIDATE.exec(text)) !== null) {
    let token = m[0];
    const start = m.index;

    // Word boundary: reject matches glued to a preceding path-ish character
    // (e.g. the tail of a URL that GFM did not autolink).
    const before = start > 0 ? text[start - 1] : "";
    if (before && /[\w.@:~/-]/.test(before)) continue;

    // Trim trailing sentence punctuation ("see src/App.tsx.", "…in a/b,").
    while (/[.,;!?]$/.test(token)) token = token.slice(0, -1);
    // A trailing slash is an explicit directory signal. Remember it, then drop
    // it so the rest of validation sees a clean token; the match itself
    // excludes the slash so the preview receives the path without it.
    const trailingSlash = /\/+$/.test(token);
    token = token.replace(/\/+$/, "");
    if (!token || !isStrictProsePath(token, trailingSlash)) continue;

    matches.push({ start, end: start + token.length, path: token });
  }
  return matches;
}
