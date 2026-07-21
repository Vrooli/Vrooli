/**
 * @vrooliComponentSource react-component-library:mermaid-diagram
 * @vrooliComponentVersion 0.1.1
 * @vrooliComponentAdoption 8e77b43c-d426-4f7f-9ade-e574901f93ea
 * @vrooliComponentAppliedAt 2026-07-21T13:00:17Z
 * @vrooliComponentSourceSha256 f0b37b2521ba2d15462396002cfe68436fccd1ab54f6492de48226876a4d079c
 * @vrooliComponentDriftHash f0b37b2521ba2d15462396002cfe68436fccd1ab54f6492de48226876a4d079c
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { useState } from "react";
export interface MermaidDiagramProps { code: string; onMermaidOpen?: (code: string) => void; sourceLabel?: string; diagramLabel?: string; }
/**
 * @libraryId react-component-library:mermaid-diagram
 * @displayName Mermaid Diagram
 * @description Mermaid diagram source preview with an expand seam.
 * @version 0.1.1
 * @tags ["markdown","mermaid"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */

export function MermaidDiagram({ code, onMermaidOpen, sourceLabel = "Source", diagramLabel = "Diagram" }: MermaidDiagramProps) { const [source, setSource] = useState(false); return <section className="rounded border border-slate-700 bg-slate-950 p-3"><div className="mb-2 flex gap-2 text-xs"><button type="button" onClick={() => setSource(false)}>{diagramLabel}</button><button type="button" onClick={() => setSource(true)}>{sourceLabel}</button><button type="button" onClick={() => onMermaidOpen?.(code)}>Open</button></div>{source ? <pre className="overflow-x-auto text-xs text-slate-200">{code}</pre> : <pre className="whitespace-pre-wrap text-xs text-slate-300">{code}</pre>}</section>; }