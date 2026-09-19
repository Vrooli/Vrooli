import { lazy, Suspense } from 'react';

// react-syntax-highlighter (+ its Prism language/theme payload) is ~500 KB+.
// It is only needed when a script is actually rendered in "view" mode, which is
// never on the initial dashboard paint. Isolating the highlighter behind
// React.lazy keeps that payload out of the main entry chunk AND out of the
// modal/page chunks until the highlighter is genuinely shown — the chunk is
// fetched on first view, with a plain <pre> fallback in the meantime so the
// script content is always legible even before the highlighter loads.

interface InnerHighlighterProps {
  content: string;
  padding: string;
}

const PrismHighlighter = lazy<React.ComponentType<InnerHighlighterProps>>(async () => {
  const [{ Prism }, { tomorrow }] = await Promise.all([
    import('react-syntax-highlighter'),
    import('react-syntax-highlighter/dist/esm/styles/prism'),
  ]);

  // Preserve the exact theme used previously: the full `tomorrow` Prism theme
  // with the `pre` block overridden for the operational-console background/spacing.
  const Highlighter = ({ content, padding }: InnerHighlighterProps) => (
    <Prism
      language="bash"
      style={{
        ...tomorrow,
        'pre[class*="language-"]': {
          ...tomorrow['pre[class*="language-"]'],
          background: 'var(--overlay-backdrop)',
          margin: 0,
          padding,
          fontSize: 'var(--text-sm)',
          fontFamily: 'var(--font-mono)',
          lineHeight: '1.5',
        },
      }}
      customStyle={{
        background: 'var(--overlay-backdrop)',
        margin: 0,
        fontSize: 'var(--text-sm)',
        fontFamily: 'var(--font-mono)',
      }}
    >
      {content}
    </Prism>
  );

  return { default: Highlighter };
});

export interface ScriptHighlighterProps {
  content: string;
  /** Padding applied to the highlighted block + the plain-text fallback. */
  padding?: string;
}

/**
 * Renders bash script content with Prism syntax highlighting, lazy-loading the
 * highlighter on demand. While the highlighter chunk is in flight, the raw
 * script is shown in a monospace <pre> so content is never blank.
 */
export const ScriptHighlighter = ({ content, padding = 'var(--spacing-md)' }: ScriptHighlighterProps) => {
  const fallback = (
    <pre
      style={{
        background: 'var(--overlay-backdrop)',
        margin: 0,
        padding,
        fontSize: 'var(--text-sm)',
        fontFamily: 'var(--font-mono)',
        lineHeight: '1.5',
        whiteSpace: 'pre',
        overflow: 'auto',
        color: 'var(--color-text)',
      }}
    >
      {content}
    </pre>
  );

  return (
    <Suspense fallback={fallback}>
      <PrismHighlighter content={content} padding={padding} />
    </Suspense>
  );
};
