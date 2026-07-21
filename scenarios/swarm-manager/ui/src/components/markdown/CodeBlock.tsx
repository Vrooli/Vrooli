/**
 * @vrooliComponentSource react-component-library:code-block
 * @vrooliComponentVersion 0.1.1
 * @vrooliComponentAdoption 2bfd9820-9f30-4621-b43f-39b98e1f7ba5
 * @vrooliComponentAppliedAt 2026-07-21T05:02:44Z
 * @vrooliComponentSourceSha256 24af7ea1b2731a61e8e5199f4c32b29dcad119517a42ad054d42c6f666ab7e7e
 * @vrooliComponentDriftHash 24af7ea1b2731a61e8e5199f4c32b29dcad119517a42ad054d42c6f666ab7e7e
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
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
  const copy = async () => { await navigator.clipboard?.writeText(code); setCopied(true); window.setTimeout(() => setCopied(false), 1500); };
  return <div className={className}><div className="flex items-center justify-between rounded-t border border-slate-700 bg-slate-900 px-2 py-1 text-xs text-slate-400"><span>{language ?? "text"}</span><button type="button" onClick={() => void copy()}>{copied ? "Copied" : "Copy"}</button></div><pre className="overflow-x-auto rounded-b border border-t-0 border-slate-700 bg-slate-950 p-3 text-sm text-slate-100"><code>{code}</code></pre></div>;
}