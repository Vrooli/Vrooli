const escapeHtml = (value = '') =>
  value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');

const formatInline = (value = '') => {
  const escaped = escapeHtml(value);

  const withLinks = escaped.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_match, text, href) => {
    const safeHref = escapeHtml(href.trim());
    return `<a href="${safeHref}" target="_blank" rel="noopener noreferrer">${text.trim()}</a>`;
  });

  const withCode = withLinks.replace(/`([^`]+)`/g, '<code>$1</code>');
  const withStrong = withCode
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    .replace(/__([^_]+)__/g, '<strong>$1</strong>');
  const withEmphasis = withStrong
    .replace(/(^|[^*_])\*([^*]+)\*/g, '$1<em>$2</em>')
    .replace(/(^|[^*_])_([^_]+)_/g, '$1<em>$2</em>');

  return withEmphasis;
};

export const renderMarkdown = (markdown = '') => {
  const source = typeof markdown === 'string' ? markdown.trim() : '';
  if (!source) {
    return '<p>No documentation available.</p>';
  }

  const lines = source.replace(/\r\n/g, '\n').split('\n');
  let html = '';
  let paragraph = [];
  let listType = '';
  let inCode = false;
  let inBlockquote = false;

  const closeList = () => {
    if (listType) {
      html += `</${listType}>`;
      listType = '';
    }
  };

  const closeBlockquote = () => {
    if (inBlockquote) {
      html += '</blockquote>';
      inBlockquote = false;
    }
  };

  const flushParagraph = () => {
    if (paragraph.length === 0) {
      return;
    }
    const text = paragraph.join(' ');
    html += `<p>${formatInline(text)}</p>`;
    paragraph = [];
  };

  lines.forEach(rawLine => {
    const line = rawLine.replace(/\t/g, '    ');
    const trimmed = line.trim();

    if (trimmed.startsWith('```')) {
      flushParagraph();
      closeList();
      closeBlockquote();
      if (inCode) {
        html += '</code></pre>';
        inCode = false;
      } else {
        inCode = true;
        html += '<pre><code>';
      }
      return;
    }

    if (inCode) {
      html += `${escapeHtml(line)}\n`;
      return;
    }

    if (!trimmed) {
      flushParagraph();
      closeBlockquote();
      closeList();
      return;
    }

    if (/^(-{3,}|_{3,}|\*{3,})$/.test(trimmed)) {
      flushParagraph();
      closeList();
      closeBlockquote();
      html += '<hr />';
      return;
    }

    const headingMatch = trimmed.match(/^(#{1,6})\s+(.*)$/);
    if (headingMatch) {
      flushParagraph();
      closeList();
      closeBlockquote();
      const level = Math.min(headingMatch[1].length, 6);
      html += `<h${level}>${formatInline(headingMatch[2])}</h${level}>`;
      return;
    }

    if (/^>\s?/.test(trimmed)) {
      flushParagraph();
      if (!inBlockquote) {
        closeList();
        html += '<blockquote>';
        inBlockquote = true;
      }
      const content = trimmed.replace(/^>\s?/, '');
      html += `<p>${formatInline(content)}</p>`;
      return;
    }

    const unorderedMatch = trimmed.match(/^[-*+]\s+(.*)$/);
    if (unorderedMatch) {
      flushParagraph();
      closeBlockquote();
      if (listType !== 'ul') {
        closeList();
        listType = 'ul';
        html += '<ul>';
      }
      html += `<li>${formatInline(unorderedMatch[1])}</li>`;
      return;
    }

    const orderedMatch = trimmed.match(/^\d+\.\s+(.*)$/);
    if (orderedMatch) {
      flushParagraph();
      closeBlockquote();
      if (listType !== 'ol') {
        closeList();
        listType = 'ol';
        html += '<ol>';
      }
      html += `<li>${formatInline(orderedMatch[1])}</li>`;
      return;
    }

    closeBlockquote();
    closeList();
    paragraph.push(trimmed);
  });

  flushParagraph();
  closeList();
  closeBlockquote();

  if (inCode) {
    html += '</code></pre>';
  }

  return html;
};
