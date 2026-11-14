import { useEffect, useState } from "react";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card";

type Prerequisite = { code: string; description: string; satisfied: boolean };
type Readiness = { scenario: string; eligible: boolean; blocking_prerequisite?: string; prerequisites?: Prerequisite[] };

const gates = ["declaration", "composition", "conformance", "attestation", "rehearsal", "distribution"];

export function ReadinessBoard() {
  const [items, setItems] = useState<Readiness[]>([]);
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");

  useEffect(() => {
    let cancelled = false;
    fetch("/api/v1/readiness")
      .then((response) => {
        if (!response.ok) throw new Error(`readiness request failed (${response.status})`);
        return response.json() as Promise<Readiness[]>;
      })
      .then((next) => { if (!cancelled) { setItems(next); setState("ready"); } })
      .catch(() => { if (!cancelled) setState("error"); });
    return () => { cancelled = true; };
  }, []);

  return (
    <div className="flex flex-col gap-4" data-testid="readiness-board">
      <Card>
        <CardHeader>
          <CardTitle>Fleet publish readiness</CardTitle>
          <CardDescription>Declarations and named blockers from the governed scenario manifests.</CardDescription>
        </CardHeader>
        <CardContent>
          {state === "loading" && <p role="status">Loading readiness…</p>}
          {state === "error" && <p role="alert">Readiness is unavailable. The API gate remains closed.</p>}
          {state === "ready" && (
            <div className="grid gap-3 md:grid-cols-2">
              {items.map((item) => <ReadinessRow key={item.scenario} item={item} />)}
              {items.length === 0 && <p className="text-app-muted-foreground">No governed scenarios were returned.</p>}
            </div>
          )}
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Gate ladder</CardTitle>
          <CardDescription>Grey means not run; a gate never appears green without a passing result.</CardDescription>
        </CardHeader>
        <CardContent>
          <ol className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
            {gates.map((gate, index) => (
              <li key={gate} className="flex items-center gap-3 rounded-control border border-app-border px-3 py-2">
                <span aria-label={`${gate} gate not run`} className="grid h-7 w-7 place-items-center rounded-full bg-app-surface-muted text-sm text-app-muted-foreground">{index + 1}</span>
                <span className="capitalize text-sm font-medium">{gate}</span>
                <span className="ml-auto text-xs text-app-muted-foreground">not run</span>
              </li>
            ))}
          </ol>
        </CardContent>
      </Card>
    </div>
  );
}

function ReadinessRow({ item }: { item: Readiness }) {
  const status = item.eligible ? "Eligible" : `Blocked: ${item.blocking_prerequisite ?? "unknown"}`;
  return (
    <article className="rounded-control border border-app-border p-3" aria-label={`${item.scenario}: ${status}`}>
      <div className="flex items-center justify-between gap-3">
        <h4 className="font-medium">{item.scenario}</h4>
        <span className={item.eligible ? "text-emerald-700" : "text-amber-700"}>
          <span aria-hidden="true">{item.eligible ? "●" : "○"}</span> {status}
        </span>
      </div>
      {item.prerequisites && item.prerequisites.length > 0 && (
        <ul className="mt-2 space-y-1 text-xs text-app-muted-foreground">
          {item.prerequisites.map((prerequisite) => <li key={prerequisite.code}>{prerequisite.satisfied ? "✓" : "!"} {prerequisite.code}: {prerequisite.description}</li>)}
        </ul>
      )}
    </article>
  );
}
