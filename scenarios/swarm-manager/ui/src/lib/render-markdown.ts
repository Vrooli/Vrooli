/**
 * Simple markdown to HTML converter for basic rendering.
 * Handles headers, bold, italic, code blocks, and links.
 *
 * Shared across FilePreview and ExportTab to avoid duplication.
 */
import { slugify, dedupeSlug } from "./heading-utils";

export function renderMarkdown(content: string): string {
  const seen = new Map<string, number>();

  return content
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
    // Inline code
    .replace(/`([^`]+)`/g, '<code class="bg-slate-900 px-1.5 py-0.5 rounded text-cyan-300 text-sm">$1</code>')
    // Bold
    .replace(/\*\*([^*]+)\*\*/g, '<strong class="font-semibold text-slate-200">$1</strong>')
    // Italic
    .replace(/\*([^*]+)\*/g, '<em class="italic">$1</em>')
    // Links
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" class="text-cyan-400 hover:underline" target="_blank" rel="noopener noreferrer">$1</a>')
    // Line breaks (paragraphs)
    .replace(/\n\n/g, '</p><p class="text-slate-300 mb-3">')
    // Wrap in paragraph
    .replace(/^(.+)$/gm, (match) => {
      if (match.startsWith('<')) return match;
      return `<p class="text-slate-300 mb-3">${match}</p>`;
    });
}
