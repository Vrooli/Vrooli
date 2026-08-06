import { useQuery } from "@tanstack/react-query";

import { fetchBindings, fetchUnbound } from "../../api/bindings";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card";
import { StatusBadge } from "../../components/ui/status-badge";
import { selectors } from "../../consts/selectors";

export function BindingRegistry() {
  const bound = useQuery({ queryKey: ["bindings"], queryFn: fetchBindings });
  const unbound = useQuery({ queryKey: ["bindings", "unbound"], queryFn: fetchUnbound });
  const error = bound.error ?? unbound.error;

  return (
    <section data-testid={selectors.bindings.registry} aria-labelledby="binding-registry-heading" className="grid gap-4 lg:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle id="binding-registry-heading">Governed callable registry</CardTitle>
          <CardDescription>Descriptor-backed operations available to programs.</CardDescription>
        </CardHeader>
        <CardContent>
          {bound.isLoading && <p role="status">Loading governed bindings…</p>}
          {error && <p role="alert">Unable to load bindings: {String(error)}</p>}
          {!bound.isLoading && !error && (
            bound.data?.length ? (
              <ul data-testid={selectors.bindings.namespace} className="space-y-2" aria-label="Bound operations">
                {bound.data.map((binding) => (
                  <li key={binding.id} className="rounded border border-app-border p-3">
                    <div className="flex items-center justify-between gap-2">
                      <span className="font-medium">{binding.id}</span>
                      <StatusBadge tone="success">{binding.effect || "governed"}</StatusBadge>
                    </div>
                    <p className="text-xs text-app-muted-foreground">{binding.signature}</p>
                  </li>
                ))}
              </ul>
            ) : <p className="text-sm text-app-muted-foreground">No governed bindings resolved.</p>
          )}
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Unbound capabilities</CardTitle>
          <CardDescription>Every gap keeps a closed-set reason.</CardDescription>
        </CardHeader>
        <CardContent>
          {unbound.isLoading && <p role="status">Loading unbound capabilities…</p>}
          {!unbound.isLoading && !unbound.error && (
            unbound.data?.length ? (
              <ul data-testid={selectors.bindings.unbound} className="space-y-2" aria-label="Unbound capabilities">
                {unbound.data.map((capability, index) => (
                  <li key={`${capability.scenario}/${capability.command}/${index}`} className="rounded border border-app-border p-3">
                    <div className="font-medium">{capability.scenario}{capability.command ? ` / ${capability.command}` : ""}</div>
                    <div className="text-sm">{String(capability.reason).replace("UNBOUND_REASON_", "").toLowerCase().replace(/_/g, " ")}</div>
                    {capability.detail && <div className="text-xs text-app-muted-foreground">{capability.detail}</div>}
                  </li>
                ))}
              </ul>
            ) : <p className="text-sm text-app-muted-foreground">No unbound capabilities reported.</p>
          )}
        </CardContent>
      </Card>
    </section>
  );
}
