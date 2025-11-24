// Minimal Markdown-to-HTML converter for previews without extra deps.
const escapeHtml = (str: string) =>
  str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');

export function markdownToHtml(markdown: string | undefined | null): string {
  if (!markdown) return '';

  const codeBlocks: string[] = [];
  let working = markdown.replace(/```([\s\S]*?)```/g, (_match, code) => {
    const index = codeBlocks.length;
    codeBlocks.push(escapeHtml(code.trim()));
    return `__CODE_BLOCK_${index}__`;
  });

  working = escapeHtml(working);

  // Inline formatting
  working = working.replace(/`([^`]+)`/g, '<code>$1</code>');
  working = working.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  working = working.replace(/\*([^*]+)\*/g, '<em>$1</em>');

  // Simple list handling
  const lines = working.split(/\r?\n/);
  const htmlLines: string[] = [];
  let inList = false;
  lines.forEach((line) => {
    if (line.trim().startsWith('- ')) {
      if (!inList) {
        htmlLines.push('<ul class="list-disc pl-4 mb-2">');
        inList = true;
      }
      htmlLines.push(`<li>${line.trim().slice(2)}</li>`);
    } else {
      if (inList) {
        htmlLines.push('</ul>');
        inList = false;
      }
      htmlLines.push(line);
    }
  });
  if (inList) {
    htmlLines.push('</ul>');
  }
  working = htmlLines.join('\n');

  // Line breaks
  working = working
    .replace(/\r?\n\r?\n/g, '<br /><br />')
    .replace(/\r?\n/g, '<br />');

  // Restore code fences
  codeBlocks.forEach((code, idx) => {
    working = working.replace(
      `__CODE_BLOCK_${idx}__`,
      `<pre class="bg-black/40 border border-white/10 rounded p-3 overflow-auto"><code>${code}</code></pre>`
    );
  });

  return working;
}
