/**
 * A deliberately small parser for the report strings the investigation agent
 * emits (see `prompts/anomaly-check.md`). Those reports are a narrow slice of
 * markdown: `### **Heading**`, `**Key**: value` field lines, `-` bullets,
 * fenced blocks and `` `code` `` spans.
 *
 * This is NOT a markdown implementation and must not grow into one. It exists
 * so `ReportBody` can build React elements from tokens instead of injecting
 * HTML — the report text is machine-generated from host state and is treated as
 * untrusted, so escaping has to be structural.
 *
 * The one hard rule: anything unrecognised falls through as readable plain
 * text. Losing a line is worse than rendering an unstyled one.
 */

export type ReportSpan =
  | { type: 'text'; text: string }
  | { type: 'code'; text: string }
  | { type: 'strong'; spans: ReportSpan[] };

export type ReportBlock =
  | { kind: 'heading'; level: number; spans: ReportSpan[] }
  | { kind: 'field'; label: string; spans: ReportSpan[]; items: ReportSpan[][] }
  | { kind: 'list'; items: ReportSpan[][] }
  | { kind: 'code'; text: string }
  | { kind: 'paragraph'; lines: ReportSpan[][] };

const HEADING_RE = /^(#{1,6})\s+(.*)$/;
const FIELD_RE = /^\*\*([^*]+)\*\*\s*:\s*(.*)$/;
const BULLET_RE = /^\s*[-*+]\s+(.*)$/;
const ORDERED_RE = /^\s*\d+[.)]\s+(.*)$/;
const FENCE_RE = /^\s*```/;

/**
 * Parse the inline syntax of a single line into spans.
 *
 * A delimiter only counts when it is closed on the same line; an unterminated
 * `**` or `` ` `` stays in the output as the literal characters the author
 * typed, which keeps malformed input readable instead of silently truncated.
 */
export function parseInline(input: string): ReportSpan[] {
  const spans: ReportSpan[] = [];
  let buffer = '';
  let index = 0;

  const flush = () => {
    if (buffer) {
      spans.push({ type: 'text', text: buffer });
      buffer = '';
    }
  };

  while (index < input.length) {
    const char = input.charAt(index);

    if (char === '`') {
      const end = input.indexOf('`', index + 1);
      if (end > index + 1) {
        flush();
        spans.push({ type: 'code', text: input.slice(index + 1, end) });
        index = end + 1;
        continue;
      }
    } else if (char === '*' && input.charAt(index + 1) === '*') {
      const end = input.indexOf('**', index + 2);
      if (end > index + 2) {
        flush();
        // The slice cannot contain `**`, so this recursion is one level deep.
        spans.push({ type: 'strong', spans: parseInline(input.slice(index + 2, end)) });
        index = end + 2;
        continue;
      }
    }

    buffer += char;
    index += 1;
  }

  flush();
  return spans;
}

/** Return the text of a list item, or null when the line is not one. */
function listItemText(line: string): string | null {
  // A capture group is `string | undefined` under `noUncheckedIndexedAccess`,
  // and `?? null` is the honest widening: a matched-but-empty group is not the
  // same as no match, and both must read as "not a list item" here.
  const bullet = BULLET_RE.exec(line);
  if (bullet) return bullet[1] ?? null;
  const ordered = ORDERED_RE.exec(line);
  if (ordered) return ordered[1] ?? null;
  return null;
}

/**
 * These reports write a field label on its own line and hang the bullets under
 * it (`**Key Findings**:` followed by `- ...`). Rejoin the two so the renderer
 * can present one labelled group rather than an orphan label.
 */
function absorbListsIntoFields(blocks: ReportBlock[]): ReportBlock[] {
  const merged: ReportBlock[] = [];
  for (const block of blocks) {
    const previous = merged[merged.length - 1];
    if (
      block.kind === 'list' &&
      previous?.kind === 'field' &&
      previous.spans.length === 0 &&
      previous.items.length === 0
    ) {
      previous.items = block.items;
      continue;
    }
    merged.push(block);
  }
  return merged;
}

/** Tokenise a report string into block-level structure. */
export function parseReport(input: string): ReportBlock[] {
  const blocks: ReportBlock[] = [];
  if (!input) return blocks;

  const lines = input.replace(/\r\n?/g, '\n').split('\n');
  let paragraph: ReportSpan[][] = [];
  let index = 0;

  const flushParagraph = () => {
    if (paragraph.length > 0) {
      blocks.push({ kind: 'paragraph', lines: paragraph });
      paragraph = [];
    }
  };

  while (index < lines.length) {
    // Bound by the loop condition, but the compiler cannot see that through an
    // index expression, so bind once and fall back to the empty line rather
    // than asserting. An out-of-range read would be a bug, and treating it as
    // a blank line ends the block instead of crashing the whole report.
    const line = lines[index] ?? '';

    if (FENCE_RE.test(line)) {
      flushParagraph();
      index += 1;
      const body: string[] = [];
      while (index < lines.length) {
        const fenced = lines[index] ?? '';
        if (FENCE_RE.test(fenced)) break;
        body.push(fenced);
        index += 1;
      }
      // Consume the closing fence when there is one; an unterminated block
      // still emits its content rather than dropping it.
      if (index < lines.length) index += 1;
      blocks.push({ kind: 'code', text: body.join('\n') });
      continue;
    }

    if (!line.trim()) {
      flushParagraph();
      index += 1;
      continue;
    }

    const heading = HEADING_RE.exec(line);
    if (heading) {
      flushParagraph();
      blocks.push({
        kind: 'heading',
        level: (heading[1] ?? '').length,
        spans: parseInline((heading[2] ?? '').trim()),
      });
      index += 1;
      continue;
    }

    if (listItemText(line) !== null) {
      flushParagraph();
      const items: ReportSpan[][] = [];
      while (index < lines.length) {
        const text = listItemText(lines[index] ?? '');
        if (text === null) break;
        items.push(parseInline(text.trim()));
        index += 1;
      }
      blocks.push({ kind: 'list', items });
      continue;
    }

    const field = FIELD_RE.exec(line.trim());
    if (field) {
      flushParagraph();
      blocks.push({
        kind: 'field',
        label: (field[1] ?? '').trim(),
        spans: parseInline((field[2] ?? '').trim()),
        items: [],
      });
      index += 1;
      continue;
    }

    paragraph.push(parseInline(line.trim()));
    index += 1;
  }

  flushParagraph();
  return absorbListsIntoFields(blocks);
}

function spansToPlainText(spans: ReportSpan[]): string {
  return spans
    .map((span) => (span.type === 'strong' ? spansToPlainText(span.spans) : span.text))
    .join('');
}

/**
 * Flatten a report to a single readable line — for places that have room for a
 * summary but not for structure, such as a timeline row. Markup characters are
 * dropped; the words are not.
 */
export function reportToPlainText(input: string): string {
  const parts: string[] = [];
  for (const block of parseReport(input)) {
    switch (block.kind) {
      case 'heading':
        parts.push(spansToPlainText(block.spans));
        break;
      case 'field':
        parts.push(
          [`${block.label}:`, spansToPlainText(block.spans), ...block.items.map(spansToPlainText)]
            .filter(Boolean)
            .join(' '),
        );
        break;
      case 'list':
        parts.push(block.items.map(spansToPlainText).join(' · '));
        break;
      case 'code':
        parts.push(block.text);
        break;
      case 'paragraph':
        parts.push(block.lines.map(spansToPlainText).join(' '));
        break;
    }
  }
  return parts.join(' · ').replace(/\s+/g, ' ').trim();
}
