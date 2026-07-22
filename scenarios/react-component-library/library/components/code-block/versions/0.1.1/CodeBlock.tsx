import { useState } from "react";

export interface CodeBlockProps { code: string; language?: string; className?: string; }
/**
 * @libraryId react-component-library:code-block
 * @displayName Code Block
 * @description Code block with language label and copy feedback.
 * @version 0.1.1
 * @tags ["markdown","code"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */

export function CodeBlock({ code, language, className }: CodeBlockProps) {
  const [copied, setCopied] = useState(false);
  const copy = async () => { await navigator.clipboard.writeText(code); setCopied(true); window.setTimeout(() => setCopied(false), 1500); };
  return <div className={className}><div className="flex items-center justify-between rounded-t border border-slate-700 bg-slate-900 px-2 py-1 text-xs text-slate-400"><span>{language ?? "text"}</span><button type="button" onClick={() => void copy()}>{copied ? "Copied" : "Copy"}</button></div><pre className="overflow-x-auto rounded-b border border-t-0 border-slate-700 bg-slate-950 p-3 text-sm text-slate-100"><code>{code}</code></pre></div>;
}
