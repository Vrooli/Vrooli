/** @vrooliComponentSource feedback.status-badge */
import { useQuery } from "@tanstack/react-query";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/Card";
import { EmptyState } from "../components/EmptyState";
import { StatusBadge } from "../components/StatusBadge";
import { describeCapabilities } from "../api/catalog";

export function CapabilitiesPage() {
  const capabilities = useQuery({
    queryKey: ["capabilities", "describe"],
    queryFn: describeCapabilities,
    staleTime: 30_000,
    retry: false,
  });
  const states = capabilities.data?.states ?? [];

  if (capabilities.isLoading)
    return (
      <div
        data-testid="capabilities-page"
        role="status"
        className="text-body text-app-muted-foreground"
      >
        Checking capability readiness…
      </div>
    );
  if (capabilities.isError)
    return (
      <EmptyState
        title="Capabilities unavailable"
        description="The capability registry could not be reached. Check the API health and retry."
      />
    );

  return (
    <div data-testid="capabilities-page" className="grid gap-space-lg">
      <header className="grid gap-space-3xs">
        <p className="text-label uppercase text-app-muted-foreground">Operator view</p>
        <h1 className="text-title">Capability readiness</h1>
        <p className="text-body text-app-muted-foreground">
          Understand which scenario integrations are available, what they enable, and how to recover
          blocked capabilities.
        </p>
      </header>
      {states.length ? (
        <div className="grid gap-space-sm md:grid-cols-2">
          {states.map((state) => (
            <Card key={state.id}>
              <CardHeader className="flex-row items-start justify-between gap-space-sm">
                <div className="grid gap-space-3xs">
                  <CardTitle>{state.name}</CardTitle>
                  <CardDescription>{state.description}</CardDescription>
                </div>
                <StatusBadge
                  tone={
                    state.status === "available"
                      ? "success"
                      : state.status === "unknown"
                        ? "warning"
                        : "danger"
                  }
                >
                  {state.status}
                </StatusBadge>
              </CardHeader>
              <CardContent className="grid gap-space-xs text-body">
                <p className="text-app-muted-foreground">
                  {state.message || `Dependency: ${state.dependencySlug}`}
                </p>
                {state.actionLabel && (
                  <p>
                    <span className="font-medium">Recovery:</span> {state.actionLabel}
                  </p>
                )}
                {state.operatorCommand && (
                  <code className="overflow-x-auto rounded-control bg-app-surface-muted p-space-xs text-body-sm">
                    {state.operatorCommand}
                  </code>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <EmptyState
          title="No capabilities declared"
          description="This scenario has no registered integrations yet."
        />
      )}
    </div>
  );
}
