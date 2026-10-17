import { Fragment, type ReactNode } from 'react';
import { parseReport, parseInline, type ReportBlock, type ReportSpan } from './parseReport';

/**
 * Renders an agent-authored report as structure.
 *
 * Every node is a React element built from parsed tokens, so the report text —
 * which is machine-generated from host state and therefore untrusted — can
 * never become markup. `dangerouslySetInnerHTML` must not be introduced here.
 */

/** Join defined class names; keeps the component free of a class-name dependency. */
function cx(...names: Array<string | undefined>): string {
  return names.filter(Boolean).join(' ');
}

/** Field labels that carry a state rather than prose, rendered as a chip. */
const CHIP_LABELS = new Set(['status', 'severity', 'risk', 'risk level', 'health', 'result', 'verdict', 'state']);

const CHIP_TONES: Array<[tone: string, values: string[]]> = [
  ['ok', ['normal', 'ok', 'okay', 'healthy', 'pass', 'passed', 'success', 'stable', 'clean', 'none', 'low']],
  ['warn', ['warning', 'warn', 'degraded', 'elevated', 'medium', 'partial', 'attention']],
  ['bad', ['critical', 'error', 'failed', 'failure', 'fatal', 'severe', 'high', 'urgent']],
];

function chipTone(value: string): string {
  const normalized = value.trim().toLowerCase();
  for (const [tone, values] of CHIP_TONES) {
    if (values.includes(normalized)) return tone;
  }
  return 'neutral';
}

function renderSpans(spans: ReportSpan[], keyPrefix: string, flattenStrong = false): ReactNode[] {
  return spans.map((span, index) => {
    const key = `${keyPrefix}-${String(index)}`;
    if (span.type === 'code') {
      return <code key={key} className="report-code">{span.text}</code>;
    }
    if (span.type === 'strong') {
      const children = renderSpans(span.spans, key, flattenStrong);
      return flattenStrong
        ? <Fragment key={key}>{children}</Fragment>
        : <strong key={key} className="report-strong">{children}</strong>;
    }
    return <Fragment key={key}>{span.text}</Fragment>;
  });
}

function renderItems(items: ReportSpan[][], keyPrefix: string): ReactNode {
  return (
    <ul className="report-list">
      {items.map((item, index) => (
        <li key={`${keyPrefix}-${String(index)}`} className="report-list-item">
          {renderSpans(item, `${keyPrefix}-${String(index)}`)}
        </li>
      ))}
    </ul>
  );
}

/** The value text when a field holds exactly one plain-text span, else ''. */
function soleTextValue(spans: ReportSpan[]): string {
  if (spans.length !== 1) return '';
  const [span] = spans;
  if (span === undefined || span.type !== 'text') return '';
  return span.text;
}

function renderField(block: Extract<ReportBlock, { kind: 'field' }>, key: string): ReactNode {
  const plainValue = soleTextValue(block.spans);
  const asChip = plainValue !== '' && CHIP_LABELS.has(block.label.toLowerCase());

  return (
    <div key={key} className="report-field">
      <dt className="report-field-label">{block.label}</dt>
      <dd className="report-field-value">
        {asChip ? (
          <span className={cx('report-chip', `report-chip-${chipTone(plainValue)}`)}>{plainValue}</span>
        ) : (
          block.spans.length > 0 && <p className="report-paragraph">{renderSpans(block.spans, key)}</p>
        )}
        {block.items.length > 0 && renderItems(block.items, key)}
      </dd>
    </div>
  );
}

function renderBlock(block: Exclude<ReportBlock, { kind: 'field' }>, index: number): ReactNode {
  const key = `block-${String(index)}`;
  switch (block.kind) {
    case 'heading': {
      // Levels are clamped so a report can never outrank the surface hosting it.
      const Tag = block.level <= 2 ? 'h4' : 'h5';
      return <Tag key={key} className="report-heading">{renderSpans(block.spans, key, true)}</Tag>;
    }
    case 'list':
      return <Fragment key={key}>{renderItems(block.items, key)}</Fragment>;
    case 'code':
      return <pre key={key} className="report-pre"><code>{block.text}</code></pre>;
    case 'paragraph':
      return (
        <p key={key} className="report-paragraph">
          {block.lines.map((line, lineIndex) => (
            <Fragment key={`${key}-${String(lineIndex)}`}>
              {lineIndex > 0 && <br />}
              {renderSpans(line, `${key}-${String(lineIndex)}`)}
            </Fragment>
          ))}
        </p>
      );
  }
}

export interface ReportBodyProps {
  /** Raw agent-authored report text. Treated as untrusted content. */
  text: string | undefined | null;
  /** Extra classes, e.g. `report-body-inset` for a bordered well. */
  className?: string;
}

export function ReportBody({ text, className }: ReportBodyProps) {
  const blocks = parseReport(text ?? '');
  if (blocks.length === 0) return null;

  // Runs of adjacent field lines become one definition list — the honest
  // element for the label/value pairs these reports are built from. Prose,
  // headings and code sit between those lists rather than inside them, which
  // a `dl` may not legally contain.
  const rendered: ReactNode[] = [];
  let index = 0;
  while (index < blocks.length) {
    // Bound by the loop condition, but an index expression is
    // `ReportBlock | undefined` under `noUncheckedIndexedAccess`. Skipping an
    // impossible hole is safer than asserting it away: a malformed report
    // loses one block rather than throwing away the entire render.
    const block = blocks[index];
    if (!block) {
      index += 1;
      continue;
    }
    if (block.kind === 'field') {
      const fields: ReactNode[] = [];
      const start = index;
      while (index < blocks.length) {
        const candidate = blocks[index];
        if (!candidate || candidate.kind !== 'field') break;
        fields.push(renderField(candidate, `block-${String(index)}`));
        index += 1;
      }
      rendered.push(<dl key={`fields-${String(start)}`} className="report-fields">{fields}</dl>);
      continue;
    }
    rendered.push(renderBlock(block, index));
    index += 1;
  }

  return <div className={cx('report-body', className)}>{rendered}</div>;
}

export interface ReportInlineProps {
  /** A single line of report text — inline syntax only, no block structure. */
  text: string | undefined | null;
}

/** Renders one line of report text for surfaces that cannot host block structure. */
export function ReportInline({ text }: ReportInlineProps) {
  if (!text) return null;
  return <>{renderSpans(parseInline(text), 'inline')}</>;
}
