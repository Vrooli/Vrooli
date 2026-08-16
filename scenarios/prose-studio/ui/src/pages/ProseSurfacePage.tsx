import { useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { selectors } from "../consts/selectors";

type Surface = "variation" | "styles" | "document" | "declarations";

const surfaceCopy: Record<Surface, { eyebrow: string; title: string; description: string }> = {
  variation: { eyebrow: "Generation", title: "Variation Board", description: "Review a spread of measured candidates. The board never orders prose by a taste score." },
  styles: { eyebrow: "Voice", title: "Style Library", description: "Compose versioned voices from directives, exemplars, anti-patterns, lexicons, and measurable targets." },
  document: { eyebrow: "Composition", title: "Document Workspace", description: "Converge an outline and each section independently, with bounded feed-forward context." },
  declarations: { eyebrow: "Governance", title: "Declaration Registry", description: "Inspect consumer-owned files. A registered file is authoritative and cannot be overwritten through the API." },
};

export function ProseSurfacePage({ surface }: { surface: Surface }) {
  const copy = surfaceCopy[surface];
  return <section data-testid={selectors.pages[surface]} className="flex flex-col gap-6" aria-labelledby={`${surface}-heading`}>
    <div className="flex flex-col gap-2">
      <p className="text-xs font-semibold uppercase tracking-[0.18em] text-app-primary">{copy.eyebrow}</p>
      <h2 id={`${surface}-heading`} className="text-2xl font-semibold">{copy.title}</h2>
      <p className="max-w-2xl text-app-muted-foreground">{copy.description}</p>
    </div>
    {surface === "variation" && <VariationBoard />}
    {surface === "styles" && <StyleLibrary />}
    {surface === "document" && <DocumentWorkspace />}
    {surface === "declarations" && <DeclarationRegistry />}
  </section>;
}

export function VariationBoard({ candidates = [] }: { candidates?: readonly string[] }) {
  return <div className="flex flex-col gap-4">
    <Card><CardHeader><CardTitle>Candidate spread</CardTitle><CardDescription>Set diversity is reported once for the set; individual cards show measurements, never a verdict.</CardDescription></CardHeader><CardContent>
      <div className="flex items-center justify-between gap-4 rounded-control bg-app-surface-muted p-4"><div><p className="text-sm text-app-muted-foreground">Diversity basis</p><p className="font-medium">Lexical 1–3 gram similarity; deterministic</p></div><p className="text-2xl font-semibold" aria-label="Set diversity is not available yet">—</p></div>
    </CardContent></Card>
    {candidates.length === 0 ? <EmptyState title="No candidate set yet" body="Choose a profile and query to create the first measured round. Candidates remain visible when they fail a hard constraint." action="Generate candidates" /> : candidates.map((candidate, index) => <Card key={candidate} tabIndex={0} aria-label={`Candidate ${index + 1}`}><CardContent><p>{candidate}</p><p className="mt-3 text-xs text-app-muted-foreground">Measurement vector is available after generation.</p></CardContent></Card>)}
    <div className="flex flex-wrap gap-3"><Button type="button" disabled title="Generate a round before rerolling">None of these — reroll</Button><span className="self-center text-sm text-app-muted-foreground">Disabled until a candidate set exists.</span></div>
  </div>;
}

function StyleLibrary() { return <div className="grid gap-4 lg:grid-cols-2"><Card><CardHeader><CardTitle>Declared voices</CardTitle><CardDescription>Files from consuming scenarios appear here with their content hash and source path.</CardDescription></CardHeader><CardContent><EmptyState title="No declared styles registered" body="Register a consumer declaration to make its voice available without integration code." action="Open declarations" href="/declarations" /></CardContent></Card><Card><CardHeader><CardTitle>Operator-authored</CardTitle><CardDescription>Local records can be composed and versioned here. Once exported, the file becomes authoritative.</CardDescription></CardHeader><CardContent><EmptyState title="No local style versions" body="Create a style from the API or CLI to start a versioned voice." action="View registry" href="/declarations" /></CardContent></Card></div>; }

function DocumentWorkspace() { return <div className="grid gap-4 lg:grid-cols-[18rem_1fr]"><Card><CardHeader><CardTitle>Outline rail</CardTitle><CardDescription>Sections own their own passage session.</CardDescription></CardHeader><CardContent><ol className="flex flex-col gap-3 text-sm"><li className="rounded-control border border-app-primary bg-app-surface-muted p-3"><span className="font-medium">1. Opening</span><span className="block text-xs text-app-muted-foreground">Not started</span></li><li className="rounded-control border border-app-border p-3"><span className="font-medium">2. Evidence</span><span className="block text-xs text-app-muted-foreground">Waiting for outline</span></li><li className="rounded-control border border-app-border p-3"><span className="font-medium">3. Close</span><span className="block text-xs text-app-muted-foreground">Waiting for outline</span></li></ol></CardContent></Card><Card><CardHeader><CardTitle>Active section</CardTitle><CardDescription>Context includes the outline, prior committed sections, following intents, and the resolved profile.</CardDescription></CardHeader><CardContent><EmptyState title="No document selected" body="Create a document to begin outline variation and section-level convergence." action="Open variation board" href="/variation" /></CardContent></Card></div>; }

function DeclarationRegistry() {
  const [state, setState] = useState<"idle" | "loading" | "ready" | "error">("idle");
  const [message, setMessage] = useState("");
  const validate = () => {
    setState("loading");
    fetch("/api/v1/prose/declarations/validate", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ root: "." }) })
      .then(async response => { if (!response.ok) throw new Error(await response.text()); return response.json(); })
      .then(() => setState("ready"))
      .catch(error => { setMessage(error instanceof Error ? error.message : "Unable to validate declarations"); setState("error"); });
  };
  useEffect(validate, []);
  return <Card><CardHeader><CardTitle>File authority</CardTitle><CardDescription>Validation is callable directly and does not invoke test-genie. Malformed files remain visible as invalid records.</CardDescription></CardHeader><CardContent>{state === "loading" && <p role="status">Checking declaration files…</p>}{state === "error" && <div role="alert" className="rounded-control border border-app-danger p-4"><p className="font-medium">Declaration validation failed</p><p className="mt-1 text-sm text-app-muted-foreground">{message}</p><Button type="button" variant="secondary" onClick={validate}>Retry validation</Button></div>}{state === "ready" && <div className="flex flex-col gap-3"><p role="status" className="font-medium">Declaration scan complete</p><p className="text-sm text-app-muted-foreground">No files are hidden: registered, invalid, collision, and unregistered states are retained for audit.</p><Button type="button" variant="secondary" onClick={validate}>Reindex files</Button></div>}</CardContent></Card>;
}

function EmptyState({ title, body, action, href }: { title: string; body: string; action: string; href?: string }) { return <div className="flex flex-col items-start gap-3 rounded-control border border-dashed border-app-border p-6"><h3 className="font-medium">{title}</h3><p className="max-w-xl text-sm text-app-muted-foreground">{body}</p>{href ? <Link className="inline-flex min-h-11 items-center justify-center rounded-control bg-app-primary px-3 text-sm font-medium text-app-primary-foreground focus-visible:outline focus-visible:outline-2" to={href}>{action}</Link> : <Button type="button" disabled title="This action requires a generated round">{action}</Button>}</div>; }
