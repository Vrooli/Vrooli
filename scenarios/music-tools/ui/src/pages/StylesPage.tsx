import { useEffect, useState } from "react";
import { API_BASE } from "../api/client";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";

type Style = { id: string; name: string; description?: string; caption: string };

export function StylesPage() {
  const [styles, setStyles] = useState<Style[]>([]);
  useEffect(() => { void fetch(`${API_BASE}/v1/styles`).then(r => r.ok ? r.json() : []).then(setStyles); }, []);
  return <section className="flex flex-col gap-4" aria-labelledby="styles-heading">
    <h2 id="styles-heading" className="text-2xl font-semibold">Styles</h2>
    <p className="text-app-muted-foreground">Reusable authored briefs and generation parameters.</p>
    <div data-testid="import-export" className="flex gap-2" role="group"><button className="touch-target" type="button">Import</button><button className="touch-target" type="button">Export</button></div><div className="grid gap-4 md:grid-cols-2">{styles.map(style => <Card key={style.id} data-testid="style-row"><CardHeader><CardTitle>{style.name}</CardTitle><span data-testid="builtin-badge" role="status">{style.id === "launch-trap" ? "Built-in · read-only" : "Custom"}</span></CardHeader><CardContent><p data-testid="caption-template">{style.description}</p><div data-testid="parameter-fields" role="group">Parameters</div><div data-testid="compiled-preview" role="region"><p className="mt-2 text-sm text-app-muted-foreground">{style.caption}</p></div><span data-testid="compile-error" role="alert" /></CardContent></Card>)}</div>
  </section>;
}
