/**
 * Simple markdown to HTML converter for basic rendering.
 * Handles headers, bold, italic, code blocks, and links.
 *
 * Shared across FilePreview and ExportTab to avoid duplication.
 */
import { slugify, dedupeSlug } from "./heading-utils";

/** Escape HTML special characters to prevent XSS when using dangerouslySetInnerHTML. */
function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

/**
 * A typed entity reference the renderer turns into a navigable link. `token`
 * is the exact inline-code text the agent emitted (e.g. `initiative:ship`);
 * `href` is the server-resolved detail path. Only references the server
 * confirmed exist are passed in, so an unresolved `type:name` span renders as
 * plain code, never a dead link.
 */
export interface InlineReferenceLink {
  token: string;
  href: string;
}

/** Build the escaped-token → href lookup the inline-code pass consults. Keys
 * are escaped the same way the content is, so they match the post-escape code
 * span text. Only relative ("/"-prefixed) hrefs are kept. */
function referenceLookup(references?: InlineReferenceLink[]): Map<string, string> | undefined {
  if (!references || references.length === 0) return undefined;
  const map = new Map<string, string>();
  for (const ref of references) {
    if (ref.href.startsWith("/")) {
      map.set(escapeHtml(ref.token), ref.href);
    }
  }
  return map.size > 0 ? map : undefined;
}

function applyInlineMarkdown(content: string, references?: Map<string, string>): string {
  return content
    // Inline code — typed references that resolved become navigable links;
    // everything else stays a code span.
    .replace(/`([^`]+)`/g, (_match, code: string) => {
      const href = references?.get(code);
      if (href) {
        return `<a href="${escapeHtml(href)}" data-entity-ref="true" class="rounded bg-slate-900 px-1.5 py-0.5 text-sm text-cyan-300 underline decoration-cyan-500/40 underline-offset-2 hover:text-cyan-200 hover:decoration-cyan-300">${code}</a>`;
      }
      return `<code class="bg-slate-900 px-1.5 py-0.5 rounded text-cyan-300 text-sm">${code}</code>`;
    })
    // Bold
    .replace(/\*\*([^*]+)\*\*/g, '<strong class="font-semibold text-slate-200">$1</strong>')
    // Italic
    .replace(/\*([^*]+)\*/g, '<em class="italic">$1</em>')
    // Links — only allow http(s) and relative URLs to prevent javascript: XSS
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_match, linkText: string, url: string) => {
      const safeUrl = /^(https?:\/\/|\/|#)/.test(url) ? url : "#";
      return `<a href="${safeUrl}" class="text-cyan-400 hover:underline" target="_blank" rel="noopener noreferrer">${linkText}</a>`;
    });
}

export function renderInlineMarkdown(content: string): string {
  return applyInlineMarkdown(escapeHtml(content));
}

export function renderMarkdown(content: string, references?: InlineReferenceLink[]): string {
  const seen = new Map<string, number>();
  const refMap = referenceLookup(references);

  // Escape HTML first to prevent XSS, then apply markdown transformations.
  return applyInlineMarkdown(escapeHtml(content), refMap)
    // Code blocks (multiline) — must come before header replacement to avoid matching inside fences
    .replace(/```(\w*)\n([\s\S]*?)```/g, '<pre class="bg-slate-900 rounded p-3 overflow-x-auto my-3"><code class="text-sm text-slate-300">$2</code></pre>')
    // Headers — emit id attributes for TOC scroll targets
    .replace(/^### (.+)$/gm, (_match, text: string) => {
      const id = dedupeSlug(slugify(text.trim()), seen);
      return `<h3 id="${id}" class="text-lg font-semibold text-slate-200 mt-4 mb-2">${text}</h3>`;
    })
    .replace(/^## (.+)$/gm, (_match, text: string) => {
      const id = dedupeSlug(slugify(text.trim()), seen);
      return `<h2 id="${id}" class="text-xl font-semibold text-slate-200 mt-6 mb-3">${text}</h2>`;
    })
    .replace(/^# (.+)$/gm, (_match, text: string) => {
      const id = dedupeSlug(slugify(text.trim()), seen);
      return `<h1 id="${id}" class="text-2xl font-bold text-slate-100 mt-6 mb-4">${text}</h1>`;
    })
    // Line breaks (paragraphs)
    .replace(/\n\n/g, '</p><p class="text-slate-300 mb-3">')
    // Wrap in paragraph
    .replace(/^(.+)$/gm, (match) => {
      if (match.startsWith('<')) return match;
      return `<p class="text-slate-300 mb-3">${match}</p>`;
    });
}
