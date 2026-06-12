import { memo, useEffect, useState } from 'react';
import { Check, Copy } from 'lucide-react';
import { useCodeCopy } from '../hooks/useCodeCopy';
import { detectLanguage, isSupportedLanguage, normalizeLanguage } from '../utils/languageDetection';
import { useTheme } from '@/contexts/useTheme';

interface CodeBlockProps {
  code: string;
  language?: string;
  className?: string;
}

const highlighterPromises = new Map<string, Promise<import('shiki').Highlighter>>();

async function getHighlighter(theme: 'github-dark' | 'github-light') {
  let promise = highlighterPromises.get(theme);
  if (!promise) {
    promise = import('shiki').then((shiki) =>
      shiki.createHighlighter({
        themes: [theme],
        langs: [
          'typescript',
          'javascript',
          'python',
          'go',
          'json',
          'bash',
          'sql',
          'html',
          'css',
          'yaml',
          'markdown',
          'jsx',
          'tsx',
          'rust',
          'java',
          'c',
          'cpp',
          'ruby',
          'php',
          'swift',
          'kotlin',
        ],
      })
    );
    highlighterPromises.set(theme, promise);
  }
  return promise;
}

export const CodeBlock = memo(function CodeBlock({
  code,
  language,
  className,
}: CodeBlockProps) {
  const [highlightedHtml, setHighlightedHtml] = useState<string | null>(null);
  const { copied, copyCode } = useCodeCopy(code);
  const { resolvedTheme } = useTheme();
  const shikiTheme = resolvedTheme === 'dark' ? 'github-dark' : 'github-light';

  const extractedLanguage = className?.replace(/^language-/, '') || language;
  const normalizedLanguage = extractedLanguage
    ? normalizeLanguage(extractedLanguage)
    : detectLanguage(code);
  const displayLanguage = normalizedLanguage === 'text' ? '' : normalizedLanguage;

  useEffect(() => {
    let cancelled = false;

    async function highlightCode() {
      try {
        const highlighter = await getHighlighter(shikiTheme);
        if (cancelled) return;

        const languageToUse = isSupportedLanguage(normalizedLanguage) ? normalizedLanguage : 'text';
        const html = highlighter.codeToHtml(code, {
          lang: languageToUse,
          theme: shikiTheme,
        });

        setHighlightedHtml(html);
      } catch (error) {
        console.warn('Syntax highlighting failed:', error);
        setHighlightedHtml(null);
      }
    }

    void highlightCode();

    return () => {
      cancelled = true;
    };
  }, [code, normalizedLanguage, shikiTheme]);

  return (
    <div className="relative my-3 overflow-hidden rounded-lg border border-white/10">
      <div className="flex items-center justify-between border-b border-white/10 bg-slate-900/80 px-3 py-2">
        <span className="font-mono text-xs text-slate-400">{displayLanguage}</span>
        <button
          onClick={() => void copyCode()}
          className="flex items-center gap-1.5 text-xs text-slate-300 transition-colors hover:text-slate-100"
          aria-label={copied ? 'Copied' : 'Copy code'}
          type="button"
        >
          {copied ? (
            <>
              <Check className="h-3.5 w-3.5 text-emerald-400" />
              <span className="text-emerald-400">Copied</span>
            </>
          ) : (
            <>
              <Copy className="h-3.5 w-3.5" />
              <span>Copy</span>
            </>
          )}
        </button>
      </div>

      <div className="overflow-x-auto bg-slate-950/80">
        {highlightedHtml ? (
          <div
            className="p-4 text-sm [&>pre]:!m-0 [&>pre]:!bg-transparent [&>pre]:!p-0"
            dangerouslySetInnerHTML={{ __html: highlightedHtml }}
          />
        ) : (
          <pre className="overflow-x-auto whitespace-pre p-4 font-mono text-sm text-slate-100">
            {code}
          </pre>
        )}
      </div>
    </div>
  );
});
