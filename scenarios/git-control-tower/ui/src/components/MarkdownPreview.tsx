import { Check, Copy } from "lucide-react";
import {
  memo,
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ComponentPropsWithoutRef,
  type ReactNode,
} from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { BundledLanguage, Highlighter } from "shiki";
import { MermaidDiagram } from "./MermaidDiagram";

interface MarkdownPreviewProps {
  content: string;
}

interface MarkdownCodeBlockProps {
  code: string;
  language?: string;
  className?: string;
}

let highlighterPromise: Promise<Highlighter> | null = null;

async function getHighlighter(): Promise<Highlighter> {
  if (!highlighterPromise) {
    highlighterPromise = import("shiki").then((shiki) =>
      shiki.createHighlighter({
        themes: ["github-dark"],
        langs: [
          "typescript",
          "javascript",
          "python",
          "go",
          "json",
          "bash",
          "sql",
          "html",
          "css",
          "yaml",
          "markdown",
          "jsx",
          "tsx",
          "rust",
          "java",
          "c",
          "cpp",
          "ruby",
          "php",
          "swift",
          "kotlin",
          "text",
        ],
      })
    );
  }
  return highlighterPromise;
}

function normalizeLanguage(lang?: string): string {
  if (!lang) return "";
  const aliases: Record<string, string> = {
    js: "javascript",
    ts: "typescript",
    py: "python",
    sh: "bash",
    shell: "bash",
    yml: "yaml",
    md: "markdown",
    rs: "rust",
    kt: "kotlin",
    "c++": "cpp",
  };
  const normalized = lang.toLowerCase().trim();
  return aliases[normalized] || normalized;
}

const MarkdownCodeBlock = memo(function MarkdownCodeBlock({
  code,
  language,
  className,
}: MarkdownCodeBlockProps) {
  const [highlightedHtml, setHighlightedHtml] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const extractedLang = className?.replace(/^language-/, "") || language || "";
  const normalizedLang = normalizeLanguage(extractedLang);
  const displayLang = normalizedLang || "text";

  const copyCode = useCallback(() => {
    const copyFallback = () => {
      const textArea = document.createElement("textarea");
      textArea.value = code;
      textArea.setAttribute("readonly", "");
      textArea.style.position = "fixed";
      textArea.style.left = "-9999px";
      document.body.appendChild(textArea);
      textArea.select();
      document.execCommand("copy");
      document.body.removeChild(textArea);
    };

    void (navigator.clipboard?.writeText(code).catch(() => {
      copyFallback();
    }) ?? Promise.resolve(copyFallback()))
      .then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      })
      .catch(() => {
        // Ignore copy failures.
      });
  }, [code]);

  useEffect(() => {
    let cancelled = false;

    async function highlight(): Promise<void> {
      try {
        const highlighter = await getHighlighter();
        if (cancelled) return;

        const loadedLangs = highlighter.getLoadedLanguages();
        let langToUse: string = normalizedLang || "text";

        if (langToUse !== "text" && !loadedLangs.includes(langToUse as BundledLanguage)) {
          try {
            await highlighter.loadLanguage(langToUse as BundledLanguage);
          } catch {
            langToUse = "text";
          }
        }

        const html = highlighter.codeToHtml(code, {
          lang: langToUse as BundledLanguage | "text",
          theme: "github-dark",
        });
        if (!cancelled) {
          setHighlightedHtml(html);
        }
      } catch {
        if (!cancelled) {
          setHighlightedHtml(null);
        }
      }
    }

    void highlight();

    return () => {
      cancelled = true;
    };
  }, [code, normalizedLang]);

  return (
    <div className="relative group rounded-lg overflow-hidden my-3">
      <div className="flex items-center justify-between px-4 py-2 bg-slate-900 border-b border-slate-700">
        <span className="text-xs text-slate-400 font-mono">{displayLang}</span>
        <button
          onClick={copyCode}
          className="flex items-center gap-1.5 text-xs text-slate-400 hover:text-slate-200 transition-colors"
          aria-label={copied ? "Copied" : "Copy code"}
          type="button"
        >
          {copied ? (
            <>
              <Check className="h-3.5 w-3.5 text-green-400" />
              <span className="text-green-400">Copied</span>
            </>
          ) : (
            <>
              <Copy className="h-3.5 w-3.5" />
              <span>Copy</span>
            </>
          )}
        </button>
      </div>

      <div className="bg-slate-800 overflow-x-auto">
        {highlightedHtml ? (
          <div
            className="p-4 text-sm [&>pre]:!bg-transparent [&>pre]:!m-0 [&>pre]:!p-0"
            dangerouslySetInnerHTML={{ __html: highlightedHtml }}
          />
        ) : (
          <pre className="p-4 text-sm text-slate-200 font-mono whitespace-pre overflow-x-auto">
            {code}
          </pre>
        )}
      </div>
    </div>
  );
});

function extractTextContent(children: ReactNode): string {
  if (typeof children === "string") {
    return children;
  }
  if (Array.isArray(children)) {
    return children.map(extractTextContent).join("");
  }
  if (children && typeof children === "object" && "props" in children) {
    return extractTextContent(
      (children as { props: { children?: ReactNode } }).props.children
    );
  }
  return "";
}

export const MarkdownPreview = memo(function MarkdownPreview({
  content,
}: MarkdownPreviewProps) {
  const components = useMemo(
    () => ({
      // Headers
      h1: ({ children }: { children?: ReactNode }) => (
        <h1 className="text-2xl font-bold text-white mb-4 mt-6 first:mt-0 border-b border-slate-700 pb-2">
          {children}
        </h1>
      ),
      h2: ({ children }: { children?: ReactNode }) => (
        <h2 className="text-xl font-semibold text-white mb-3 mt-5 first:mt-0 border-b border-slate-800 pb-2">
          {children}
        </h2>
      ),
      h3: ({ children }: { children?: ReactNode }) => (
        <h3 className="text-lg font-semibold text-white mb-2 mt-4 first:mt-0">
          {children}
        </h3>
      ),
      h4: ({ children }: { children?: ReactNode }) => (
        <h4 className="text-base font-semibold text-white mb-2 mt-3 first:mt-0">
          {children}
        </h4>
      ),
      // Paragraphs and text
      p: ({ children }: { children?: ReactNode }) => (
        <p className="mb-4 leading-relaxed">{children}</p>
      ),
      // Links
      a: ({ href, children }: { href?: string; children?: ReactNode }) => (
        <a
          href={href}
          target="_blank"
          rel="noopener noreferrer"
          className="text-blue-400 hover:text-blue-300 underline"
        >
          {children}
        </a>
      ),
      // Lists
      ul: ({ children }: { children?: ReactNode }) => (
        <ul className="list-disc list-inside mb-4 space-y-1">{children}</ul>
      ),
      ol: ({ children }: { children?: ReactNode }) => (
        <ol className="list-decimal list-inside mb-4 space-y-1">{children}</ol>
      ),
      li: ({ children }: { children?: ReactNode }) => (
        <li className="text-slate-300">{children}</li>
      ),
      // Code
      code: ({
        inline,
        className,
        children,
        ...props
      }: ComponentPropsWithoutRef<"code"> & {
        inline?: boolean;
        className?: string;
      }) => {
        const codeContent = extractTextContent(children);
        const isInline = inline ?? (!className && !codeContent.includes("\n"));

        if (isInline) {
          return (
            <code className="bg-slate-800 text-emerald-300 px-1.5 py-0.5 rounded text-sm font-mono">
              {children}
            </code>
          );
        }

        if (className === "language-mermaid") {
          return <MermaidDiagram code={codeContent} />;
        }

        return <MarkdownCodeBlock code={codeContent} className={className} {...props} />;
      },
      pre: ({ children }: { children?: ReactNode }) => <>{children}</>,
      // Blockquotes
      blockquote: ({ children }: { children?: ReactNode }) => (
        <blockquote className="border-l-4 border-slate-600 pl-4 italic text-slate-400 my-4">
          {children}
        </blockquote>
      ),
      // Tables
      table: ({ children }: { children?: ReactNode }) => (
        <div className="overflow-x-auto mb-4">
          <table className="min-w-full border border-slate-700 rounded">
            {children}
          </table>
        </div>
      ),
      thead: ({ children }: { children?: ReactNode }) => (
        <thead className="bg-slate-800">{children}</thead>
      ),
      tbody: ({ children }: { children?: ReactNode }) => (
        <tbody className="divide-y divide-slate-700">{children}</tbody>
      ),
      tr: ({ children }: { children?: ReactNode }) => <tr>{children}</tr>,
      th: ({ children }: { children?: ReactNode }) => (
        <th className="px-4 py-2 text-left text-sm font-semibold text-slate-200">
          {children}
        </th>
      ),
      td: ({ children }: { children?: ReactNode }) => (
        <td className="px-4 py-2 text-sm text-slate-300">{children}</td>
      ),
      // Horizontal rule
      hr: () => <hr className="border-slate-700 my-6" />,
      // Images
      img: ({ src, alt }: { src?: string; alt?: string }) => (
        <img
          src={src}
          alt={alt || ""}
          className="max-w-full h-auto rounded my-4"
        />
      ),
      // Strong and emphasis
      strong: ({ children }: { children?: ReactNode }) => (
        <strong className="font-semibold text-white">{children}</strong>
      ),
      em: ({ children }: { children?: ReactNode }) => (
        <em className="italic text-slate-300">{children}</em>
      ),
    }),
    []
  );

  return (
    <div
      className="p-6 max-w-none text-slate-200 markdown-preview"
      data-testid="markdown-preview"
    >
      <ReactMarkdown components={components} remarkPlugins={[remarkGfm]}>
        {content}
      </ReactMarkdown>
    </div>
  );
});
