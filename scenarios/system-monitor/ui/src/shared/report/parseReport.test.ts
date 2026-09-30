import { describe, expect, it } from 'vitest';
import { parseInline, parseReport, reportToPlainText, type ReportSpan } from './parseReport';

/** Flatten spans to the characters a reader would see. */
function seen(spans: ReportSpan[]): string {
  return spans.map((span) => (span.type === 'strong' ? seen(span.spans) : span.text)).join('');
}

describe('parseInline', () => {
  it('reads a bold run', () => {
    expect(parseInline('a **b** c')).toEqual([
      { type: 'text', text: 'a ' },
      { type: 'strong', spans: [{ type: 'text', text: 'b' }] },
      { type: 'text', text: ' c' },
    ]);
  });

  it('reads adjacent bold runs without swallowing the gap', () => {
    expect(parseInline('**one****two**')).toEqual([
      { type: 'strong', spans: [{ type: 'text', text: 'one' }] },
      { type: 'strong', spans: [{ type: 'text', text: 'two' }] },
    ]);
  });

  it('reads a code span inside a bold run', () => {
    expect(parseInline('**use `ps aux`**')).toEqual([
      {
        type: 'strong',
        spans: [
          { type: 'text', text: 'use ' },
          { type: 'code', text: 'ps aux' },
        ],
      },
    ]);
  });

  it('keeps an unterminated ** as literal characters', () => {
    const spans = parseInline('**Status: unclosed');
    expect(spans).toEqual([{ type: 'text', text: '**Status: unclosed' }]);
    expect(seen(spans)).toBe('**Status: unclosed');
  });

  it('keeps an unterminated backtick as a literal character', () => {
    expect(parseInline('run `ps aux')).toEqual([{ type: 'text', text: 'run `ps aux' }]);
  });

  it('keeps an empty code span as literal backticks rather than dropping it', () => {
    expect(seen(parseInline('a `` b'))).toBe('a `` b');
  });

  it('returns nothing for an empty line', () => {
    expect(parseInline('')).toEqual([]);
  });

  it('never loses characters, whatever the delimiters do', () => {
    for (const input of ['**a*', '`a**b`', '***a***', '*', '**', '`', 'plain']) {
      expect(seen(parseInline(input)).replace(/\*|`/g, '')).toBe(input.replace(/\*|`/g, ''));
    }
  });
});

describe('parseReport', () => {
  it('returns no blocks for an empty string', () => {
    expect(parseReport('')).toEqual([]);
    expect(parseReport('   \n  \n')).toEqual([]);
  });

  it('reads the **Key**: value shape as a field', () => {
    expect(parseReport('**Status**: Normal')).toEqual([
      { kind: 'field', label: 'Status', spans: [{ type: 'text', text: 'Normal' }], items: [] },
    ]);
  });

  it('hangs a following bullet list under a bare field label', () => {
    const blocks = parseReport('**Scripts Executed**: \n- ran cpu-probe.sh\n- ran io-probe.sh');
    expect(blocks).toHaveLength(1);
    expect(blocks[0]).toMatchObject({ kind: 'field', label: 'Scripts Executed' });
    const field = blocks[0] as Extract<(typeof blocks)[number], { kind: 'field' }>;
    expect(field.items.map(seen)).toEqual(['ran cpu-probe.sh', 'ran io-probe.sh']);
  });

  it('does not absorb a list into a field that already carries a value', () => {
    const blocks = parseReport('**Status**: Normal\n- unrelated bullet');
    expect(blocks.map((block) => block.kind)).toEqual(['field', 'list']);
  });

  it('reads headings and strips the bold the agent wraps them in', () => {
    const blocks = parseReport('### **Investigation Summary**');
    expect(blocks[0]).toMatchObject({ kind: 'heading', level: 3 });
    const heading = blocks[0] as Extract<(typeof blocks)[number], { kind: 'heading' }>;
    expect(seen(heading.spans)).toBe('Investigation Summary');
  });

  it('reads ordered list items', () => {
    const blocks = parseReport('1. first\n2) second');
    expect(blocks).toHaveLength(1);
    const list = blocks[0] as Extract<(typeof blocks)[number], { kind: 'list' }>;
    expect(list.items.map(seen)).toEqual(['first', 'second']);
  });

  it('reads a fenced code block and drops only the fences', () => {
    const blocks = parseReport('```bash\nps aux | head\nuptime\n```');
    expect(blocks).toEqual([{ kind: 'code', text: 'ps aux | head\nuptime' }]);
  });

  it('emits the content of an unterminated fence rather than losing it', () => {
    expect(parseReport('```\nps aux')).toEqual([{ kind: 'code', text: 'ps aux' }]);
  });

  it('keeps consecutive prose lines together and splits on a blank line', () => {
    const blocks = parseReport('line one\nline two\n\nline three');
    expect(blocks).toHaveLength(2);
    const first = blocks[0] as Extract<(typeof blocks)[number], { kind: 'paragraph' }>;
    expect(first.lines.map(seen)).toEqual(['line one', 'line two']);
  });

  it('falls unrecognised syntax through as readable text instead of dropping it', () => {
    const blocks = parseReport('> a blockquote\n| a | table |\n[link](http://x)');
    const paragraph = blocks[0] as Extract<(typeof blocks)[number], { kind: 'paragraph' }>;
    expect(paragraph.lines.map(seen)).toEqual(['> a blockquote', '| a | table |', '[link](http://x)']);
  });

  it('normalises CRLF input', () => {
    expect(parseReport('**Status**: Normal\r\n')).toHaveLength(1);
  });

  it('parses a whole realistic agent report without losing a line', () => {
    const report = [
      '### **Investigation Summary**',
      '',
      '**Status**: Warning',
      '',
      '**Scripts Executed**:',
      '- `cpu-probe.sh` — captured top talkers',
      '',
      '**Key Findings**:',
      '- node consumed **91.4%** CPU',
    ].join('\n');
    const kinds = parseReport(report).map((block) => block.kind);
    expect(kinds).toEqual(['heading', 'field', 'field', 'field']);
  });
});

describe('reportToPlainText', () => {
  it('flattens a report to one line with no syntax characters left', () => {
    const flat = reportToPlainText('### **Summary**\n\n**Status**: Normal\n- ran `probe.sh`\n\ntail prose');
    expect(flat).toBe('Summary · Status: Normal · ran probe.sh · tail prose');
    expect(flat).not.toContain('*');
    expect(flat).not.toContain('`');
  });

  it('flattens a bare field label together with its bullets', () => {
    expect(reportToPlainText('**Key Findings**:\n- disk at 94%')).toBe('Key Findings: disk at 94%');
  });

  it('includes fenced code content', () => {
    expect(reportToPlainText('```\nuptime\n```')).toBe('uptime');
  });

  it('is empty for empty input', () => {
    expect(reportToPlainText('')).toBe('');
  });

  it('passes plain sentences through unchanged', () => {
    expect(reportToPlainText('Investigation activity recorded')).toBe('Investigation activity recorded');
  });
});
