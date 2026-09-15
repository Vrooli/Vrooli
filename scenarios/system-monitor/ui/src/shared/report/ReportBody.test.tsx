import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders as render } from '../../test-utils/renderWithProviders';
import { ReportBody, ReportInline } from './ReportBody';

describe('ReportBody', () => {
  it('renders nothing for empty or absent text', () => {
    const { container, rerender } = render(<ReportBody text="" />);
    expect(container).toBeEmptyDOMElement();
    rerender(<ReportBody text={null} />);
    expect(container).toBeEmptyDOMElement();
    rerender(<ReportBody text={undefined} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('renders a **Key**: value line as a labelled pair, not as a bold run in a sentence', () => {
    const { container } = render(<ReportBody text="**Scripts Executed**: three probes" />);
    expect(container.querySelector('.report-field-label')?.textContent).toBe('Scripts Executed');
    expect(container.querySelector('.report-field-value')?.textContent).toBe('three probes');
    expect(container.textContent).not.toContain('*');
  });

  it('renders a Status value as a state chip toned by its value', () => {
    const tones: Array<[string, string]> = [
      ['Normal', 'report-chip-ok'],
      ['Warning', 'report-chip-warn'],
      ['Critical', 'report-chip-bad'],
      ['Inconclusive', 'report-chip-neutral'],
    ];
    for (const [value, expected] of tones) {
      const { container, unmount } = render(<ReportBody text={`**Status**: ${value}`} />);
      const chip = container.querySelector('.report-chip');
      expect(chip?.textContent).toBe(value);
      expect(chip?.className).toContain(expected);
      unmount();
    }
  });

  it('does not chip a label that carries prose rather than a state', () => {
    const { container } = render(<ReportBody text="**Recommendations**: restart the collector" />);
    expect(container.querySelector('.report-chip')).toBeNull();
    expect(screen.getByText('restart the collector')).toBeInTheDocument();
  });

  it('groups adjacent fields into a single definition list', () => {
    const { container } = render(<ReportBody text={'**Status**: Normal\n**Host**: bench-01'} />);
    expect(container.querySelectorAll('dl.report-fields')).toHaveLength(1);
    expect(container.querySelectorAll('dt.report-field-label')).toHaveLength(2);
  });

  it('renders headings, bullets and code spans as structure', () => {
    const report = [
      '### **Investigation Summary**',
      '',
      '**Key Findings**:',
      '- node held `91.4%` CPU',
      '- disk queue was **deep**',
    ].join('\n');
    const { container } = render(<ReportBody text={report} />);

    expect(screen.getByRole('heading', { name: 'Investigation Summary' })).toBeInTheDocument();
    expect(container.querySelectorAll('.report-list-item')).toHaveLength(2);
    expect(container.querySelector('code.report-code')?.textContent).toBe('91.4%');
    expect(container.querySelector('strong.report-strong')?.textContent).toBe('deep');
    expect(container.textContent).not.toContain('`');
    expect(container.textContent).not.toContain('**');
  });

  it('renders a fenced block as preformatted mono text', () => {
    const { container } = render(<ReportBody text={'```bash\nps aux | head\nuptime\n```'} />);
    const pre = container.querySelector('pre.report-pre');
    expect(pre?.textContent).toBe('ps aux | head\nuptime');
    expect(container.textContent).not.toContain('```');
  });

  it('keeps multiple prose lines with a line break between them', () => {
    const { container } = render(<ReportBody text={'first line\nsecond line'} />);
    const paragraph = container.querySelector('p.report-paragraph');
    expect(paragraph?.querySelectorAll('br')).toHaveLength(1);
    expect(paragraph?.textContent).toBe('first linesecond line');
  });

  it('renders unrecognised syntax as readable text rather than dropping it', () => {
    const { container } = render(<ReportBody text={'> quoted advice\n| a | table |\n~~struck~~'} />);
    const text = container.textContent ?? '';
    expect(text).toContain('> quoted advice');
    expect(text).toContain('| a | table |');
    expect(text).toContain('~~struck~~');
  });

  it('keeps an unterminated bold marker visible as text instead of eating the line', () => {
    const { container } = render(<ReportBody text="**Status: the marker never closed" />);
    expect(container.textContent).toBe('**Status: the marker never closed');
  });

  // The report text is machine-generated from host state. It is untrusted:
  // these two cases are the reason the renderer builds elements from tokens
  // and must never use dangerouslySetInnerHTML.
  it('renders a <script> payload as literal text and never as markup', () => {
    const payload = '**Status**: <script>window.__owned = true;</script>';
    const { container } = render(<ReportBody text={payload} />);
    expect(container.querySelector('script')).toBeNull();
    expect(container.textContent).toContain('<script>window.__owned = true;</script>');
    expect((window as unknown as Record<string, unknown>).__owned).toBeUndefined();
  });

  it('renders an <img onerror=...> payload as literal text and never as an element', () => {
    const payload = '- found `<img src=x onerror="window.__owned = true">` in the log';
    const { container } = render(<ReportBody text={payload} />);
    expect(container.querySelector('img')).toBeNull();
    // The angle brackets survive only in escaped form, so no element is built.
    expect(container.innerHTML).not.toContain('<img');
    expect(container.innerHTML).toContain('&lt;img src=x onerror=');
    expect(container.textContent).toContain('<img src=x onerror="window.__owned = true">');
    expect((window as unknown as Record<string, unknown>).__owned).toBeUndefined();
  });

  it('escapes markup that arrives inside a fenced block', () => {
    const { container } = render(<ReportBody text={'```\n<iframe src="evil"></iframe>\n```'} />);
    expect(container.querySelector('iframe')).toBeNull();
    expect(container.querySelector('pre')?.textContent).toBe('<iframe src="evil"></iframe>');
  });

  it('applies caller classes alongside its own', () => {
    const { container } = render(<ReportBody text="a line" className="report-body-inset" />);
    expect(container.firstElementChild?.className).toBe('report-body report-body-inset');
  });
});

describe('ReportInline', () => {
  it('renders bold and code spans without block structure', () => {
    const { container } = render(<ReportInline text="raise **swap** or tune `vm.swappiness`" />);
    expect(container.querySelector('strong.report-strong')?.textContent).toBe('swap');
    expect(container.querySelector('code.report-code')?.textContent).toBe('vm.swappiness');
    expect(container.textContent).toBe('raise swap or tune vm.swappiness');
  });

  it('renders nothing for empty text', () => {
    const { container, rerender } = render(<ReportInline text="" />);
    expect(container).toBeEmptyDOMElement();
    rerender(<ReportInline text={undefined} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('renders a markup payload as literal text', () => {
    const { container } = render(<ReportInline text='<img src=x onerror="alert(1)">' />);
    expect(container.querySelector('img')).toBeNull();
    expect(container.textContent).toBe('<img src=x onerror="alert(1)">');
  });
});
