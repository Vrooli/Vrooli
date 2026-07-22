import type { ReactNode } from "react";
export interface InlineCodeProps { children: ReactNode; onClick?: (text: string) => void; }
/** @libraryId react-component-library:inline-code @version 0.1.0 
 * @libraryId react-component-library:inline-code
 * @displayName Inline Code
 * @description Inline code token renderer with copy affordance.
 * @version 0.1.0
 * @tags ["markdown","code"]
*/
export function InlineCode({ children, onClick }: InlineCodeProps) { const text = String(children ?? ""); return <button type="button" onClick={() => onClick?.(text)} className="rounded bg-slate-800 px-1 py-0.5 font-mono text-cyan-200 disabled:cursor-text" disabled={!onClick}>{children}</button>; }