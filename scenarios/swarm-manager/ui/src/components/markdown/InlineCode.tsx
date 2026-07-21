/**
 * @vrooliComponentSource react-component-library:inline-code
 * @vrooliComponentVersion 0.1.1
 * @vrooliComponentAdoption 0f8b663a-3b47-4fdd-848a-6b73e5167dac
 * @vrooliComponentAppliedAt 2026-07-21T05:02:44Z
 * @vrooliComponentSourceSha256 3c50a4a77f0750b598fd3c68d9bf81e0bbe24ef5684019530087b834d900252e
 * @vrooliComponentDriftHash 3c50a4a77f0750b598fd3c68d9bf81e0bbe24ef5684019530087b834d900252e
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ReactNode } from "react";
export interface InlineCodeProps { children: ReactNode; onClick?: (text: string) => void; }
/**
 * @libraryId react-component-library:inline-code
 * @displayName Inline Code
 * @description Inline code token renderer with copy affordance.
 * @version 0.1.1
 * @tags ["markdown","code"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */

export function InlineCode({ children, onClick }: InlineCodeProps) { const text = String(children ?? ""); return <button type="button" onClick={() => onClick?.(text)} className="rounded bg-slate-800 px-1 py-0.5 font-mono text-cyan-200 disabled:cursor-text" disabled={!onClick}>{children}</button>; }