import { Fragment, type ReactNode } from 'react';
import { safeHref } from '../presentation/links';
import { parseMarkdown, slugify } from './markdownParse';

/**
 * A deliberately small Markdown renderer for operator-authored documents such
 * as the privacy policy. It builds React elements (never HTML strings), so
 * markup in the source is shown as text, not executed.
 *
 * Supported: # and ## section headings, ### subsections, paragraphs (single newlines become line
 * breaks), - and 1. lists, **bold**, *italic*, `code`, [links](url), and bare
 * email addresses.
 */

const INLINE = /(\*\*[^*]+\*\*|\*[^*\s][^*]*\*|`[^`]+`|\[[^\]]+\]\([^)\s]+\)|[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,})/g;

function renderInline(text: string): ReactNode[] {
  return text.split(INLINE).filter(Boolean).map((part, index) => {
    if (part.startsWith('**') && part.endsWith('**')) return <strong key={index}>{part.slice(2, -2)}</strong>;
    if (part.startsWith('`') && part.endsWith('`')) return <code key={index}>{part.slice(1, -1)}</code>;
    if (part.startsWith('*') && part.endsWith('*') && part.length > 2) return <em key={index}>{part.slice(1, -1)}</em>;
    const link = /^\[([^\]]+)\]\(([^)\s]+)\)$/.exec(part);
    if (link) {
      const target = link[2] ?? '';
      // Legal documents routinely link to an email address; safeHref only admits web paths.
      const href = /^mailto:[^\s@]+@[^\s@]+$/i.test(target) ? target : safeHref(target);
      return href ? <a key={index} href={href} {...(/^https?:/i.test(href) ? { target: '_blank', rel: 'noopener noreferrer' } : {})}>{link[1]}</a> : <Fragment key={index}>{link[1]}</Fragment>;
    }
    if (/^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$/.test(part)) return <a key={index} href={`mailto:${part}`}>{part}</a>;
    return <Fragment key={index}>{part}</Fragment>;
  });
}

export function Markdown({ source }: { source: string }) {
  return <>{parseMarkdown(source).map((block, index) => {
    if (block.kind === 'heading') {
      const Tag = block.level === 2 ? 'h2' : 'h3';
      return <Tag key={index} id={slugify(block.text)}>{renderInline(block.text)}</Tag>;
    }
    if (block.kind === 'list') {
      const Tag = block.ordered ? 'ol' : 'ul';
      return <Tag key={index}>{block.items.map((item, itemIndex) => <li key={itemIndex}>{renderInline(item)}</li>)}</Tag>;
    }
    return <p key={index}>{block.lines.map((line, lineIndex) => <Fragment key={lineIndex}>{lineIndex > 0 && <br />}{renderInline(line)}</Fragment>)}</p>;
  })}</>;
}
