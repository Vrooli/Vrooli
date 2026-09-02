import { useEffect, useState } from "react";
import { proseApi } from "../api/prose";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { selectors } from "../consts/selectors";
import { SurfaceFrame } from "./VariationPage";

export function StylesPage() {
  const [registry, setRegistry] = useState<Record<string, unknown>>(); const [error, setError] = useState<string>();
  useEffect(() => { proseApi.registry().then(setRegistry).catch((err: unknown) => setError(err instanceof Error ? err.message : "Registry unavailable")); }, []);
  const transforms = Array.isArray(registry?.transforms) ? registry.transforms : [];
  return <SurfaceFrame eyebrow="Voice" title="Style Library" description="Compose versioned voices from directives, exemplars, anti-patterns, lexicons, and measurable targets." testId={selectors.pages.styles} headingId="styles-heading"><div className="grid gap-4 lg:grid-cols-2"><Card><CardHeader><CardTitle>Declared voices</CardTitle><CardDescription>Files from consuming scenarios appear here with their content hash and source path.</CardDescription></CardHeader><CardContent><p className="text-sm font-medium">No declared styles registered</p>{error ? <p role="alert">{error}</p> : <p className="text-sm text-app-muted-foreground">The registry is live. Resolve a profile from the generation workspace to inspect its full versioned declaration.</p>}</CardContent></Card><Card><CardHeader><CardTitle>Transform operations</CardTitle><CardDescription>Typed refinement operations exposed by the service.</CardDescription></CardHeader><CardContent>{transforms.length === 0 ? <EmptyState title="Registry not loaded" description="The operator surface will show transform schemas when the API responds." action={<Button disabled title="Registry must load first">Refresh</Button>} /> : <ul className="flex flex-col gap-2 text-sm">{transforms.map((transform, index) => <li key={index} className="rounded-control border border-app-border p-3">{JSON.stringify(transform)}</li>)}</ul>}</CardContent></Card></div></SurfaceFrame>;
}
