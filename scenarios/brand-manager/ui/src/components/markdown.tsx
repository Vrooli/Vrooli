import { useState, useMemo, type ReactNode } from "react";
import { Check, Copy } from "lucide-react";

/**
 * Minimal markdown renderer for author-controlled rule documentation.
 * Supports: paragraphs, ordered/unordered lists, fenced code blocks, inline code,
 * **bold**, and `code`. Intentionally narrower than a generic markdown library —
 * the inputs come from the brand-manager Go source, not user content.
 */

interface CodeBlockProps {
  code: string;
  language?: string;
}

function CodeBlock({ code, language }: CodeBlockProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      // clipboard unavailable; fail silently
    }
  };

  return (
    <div className="relative my-3 rounded-lg overflow-hidden border border-white/10">
      <div className="flex items-center justify-between px-3 py-1.5 bg-slate-800/70 border-b border-white/10">
        <span className="text-[11px] font-mono text-slate-400">{language || ""}</span>
        <button
          type="button"
          onClick={handleCopy}
          className="flex items-center gap-1 text-[11px] text-slate-400 hover:text-slate-100 transition-colors"
          aria-label={copied ? "Copied" : "Copy code"}
        >
          {copied ? (
            <>
              <Check className="h-3 w-3 text-emerald-400" />
              <span className="text-emerald-400">Copied</span>
            </>
          ) : (
            <>
              <Copy className="h-3 w-3" />
              <span>Copy</span>
            </>
          )}
        </button>
      </div>
      <pre className="p-3 text-xs text-slate-100 font-mono whitespace-pre overflow-x-auto bg-slate-900/60">
        {code}
      </pre>
    </div>
  );
}

function renderInline(text: string, keyPrefix: string): ReactNode[] {
  // Tokens: `code`, **bold**
  const out: ReactNode[] = [];
  const re = /(`[^`]+`|\*\*[^*]+\*\*)/g;
  let lastIdx = 0;
  let m: RegExpExecArray | null;
  let i = 0;
  while ((m = re.exec(text)) !== null) {
    if (m.index > lastIdx) out.push(text.slice(lastIdx, m.index));
    const tok = m[0];
    const k = `${keyPrefix}-${i++}`;
    if (tok.startsWith("`")) {
      out.push(
        <code key={k} className="px-1 py-0.5 rounded bg-white/10 text-amber-200 text-[0.85em] font-mono">
          {tok.slice(1, -1)}
        </code>,
      );
    } else {
      out.push(
        <strong key={k} className="font-semibold text-slate-100">
          {tok.slice(2, -2)}
        </strong>,
      );
    }
    lastIdx = m.index + tok.length;
  }
  if (lastIdx < text.length) out.push(text.slice(lastIdx));
  return out;
}

interface MarkdownProps {
  content: string;
  className?: string;
}

interface Block {
  kind: "p" | "ul" | "ol" | "code";
  lines: string[];
  language?: string;
}

function parseBlocks(content: string): Block[] {
  const lines = content.replace(/\r\n/g, "\n").split("\n");
  const blocks: Block[] = [];
  let i = 0;
  while (i < lines.length) {
    const line = lines[i] ?? "";
    // Fenced code block
    const fence = /^```(\w+)?\s*$/.exec(line);
    if (fence) {
      const lang = fence[1];
      const codeLines: string[] = [];
      i++;
      while (i < lines.length && !/^```\s*$/.test(lines[i] ?? "")) {
        codeLines.push(lines[i] ?? "");
        i++;
      }
      i++; // skip closing fence
      blocks.push({ kind: "code", lines: codeLines, language: lang });
      continue;
    }
    if (line.trim() === "") {
      i++;
      continue;
    }
    // Ordered list
    if (/^\s*\d+\.\s+/.test(line)) {
      const items: string[] = [];
      while (i < lines.length && /^\s*\d+\.\s+/.test(lines[i] ?? "")) {
        items.push((lines[i] ?? "").replace(/^\s*\d+\.\s+/, ""));
        i++;
      }
      blocks.push({ kind: "ol", lines: items });
      continue;
    }
    // Unordered list
    if (/^\s*[-*]\s+/.test(line)) {
      const items: string[] = [];
      while (i < lines.length && /^\s*[-*]\s+/.test(lines[i] ?? "")) {
        items.push((lines[i] ?? "").replace(/^\s*[-*]\s+/, ""));
        i++;
      }
      blocks.push({ kind: "ul", lines: items });
      continue;
    }
    // Paragraph: gather until blank or block boundary
    const para: string[] = [];
    while (i < lines.length) {
      const cur = lines[i] ?? "";
      if (
        cur.trim() === "" ||
        /^```/.test(cur) ||
        /^\s*\d+\.\s+/.test(cur) ||
        /^\s*[-*]\s+/.test(cur)
      ) {
        break;
      }
      para.push(cur);
      i++;
    }
    blocks.push({ kind: "p", lines: para });
  }
  return blocks;
}

export function Markdown({ content, className }: MarkdownProps) {
  const blocks = useMemo(() => parseBlocks(content), [content]);
  return (
    <div className={`markdown text-sm text-slate-300 leading-relaxed space-y-2 ${className || ""}`}>
      {blocks.map((b, idx) => {
        if (b.kind === "code") {
          return <CodeBlock key={idx} code={b.lines.join("\n")} language={b.language} />;
        }
        if (b.kind === "ul") {
          return (
            <ul key={idx} className="list-disc list-inside space-y-1">
              {b.lines.map((li, j) => (
                <li key={j}>{renderInline(li, `${idx}-${j}`)}</li>
              ))}
            </ul>
          );
        }
        if (b.kind === "ol") {
          return (
            <ol key={idx} className="list-decimal list-inside space-y-1">
              {b.lines.map((li, j) => (
                <li key={j}>{renderInline(li, `${idx}-${j}`)}</li>
              ))}
            </ol>
          );
        }
        return (
          <p key={idx}>{renderInline(b.lines.join(" "), `${idx}`)}</p>
        );
      })}
    </div>
  );
}
