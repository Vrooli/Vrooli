import { useEffect, useState } from "react";
import { Button } from "@vrooli/react-component-library/Button/1.2.0";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { selectors } from "../consts/selectors";
import { SurfaceFrame } from "./VariationPage";

export function DeclarationsPage() {
  const [state, setState] = useState<"idle" | "loading" | "ready" | "error">("idle");
  const [message, setMessage] = useState("");
  const validate = () => { setState("loading"); fetch("/api/v1/prose/declarations/validate", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ root: "." }) }).then(async response => { if (!response.ok) throw new Error(await response.text()); return response.json(); }).then(() => setState("ready")).catch(error => { setMessage(error instanceof Error ? error.message : "Unable to validate declarations"); setState("error"); }); };
  useEffect(() => { validate(); }, []);
  return <SurfaceFrame eyebrow="Governance" title="Declaration Registry" description="Inspect consumer-owned files. A registered file is authoritative and cannot be overwritten through the API." testId={selectors.pages.declarations} headingId="declarations-heading"><Card><CardHeader><CardTitle>File authority</CardTitle><CardDescription>Validation is callable directly and does not invoke test-genie. Malformed files remain visible as invalid records.</CardDescription></CardHeader><CardContent>{state === "idle" || state === "loading" ? <p role="status">Checking declaration files…</p> : state === "error" ? <div role="alert" className="rounded-control border border-app-danger p-4"><p className="font-medium">Declaration validation failed</p><p className="mt-1 text-sm text-app-muted-foreground">{message}</p><Button variant="secondary" onClick={validate}>Retry validation</Button></div> : <div className="flex flex-col gap-3"><p role="status" className="font-medium">Declaration scan complete</p><p className="text-sm text-app-muted-foreground">No files are hidden: registered, invalid, collision, and unregistered states are retained for audit.</p><Button variant="secondary" onClick={validate}>Reindex files</Button></div>}</CardContent></Card></SurfaceFrame>;
}
