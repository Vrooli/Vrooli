import type { MessageShape } from "@bufbuild/protobuf";
import type { GetModelPolicyCatalogResponseSchema } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";
import { Badge } from "../../ui/badge";
import { Card, CardContent } from "../../ui/card";

type CatalogResponse = MessageShape<typeof GetModelPolicyCatalogResponseSchema>;

export function ModelPolicyTab({
  data,
  loading,
  error,
}: {
  data: CatalogResponse | null;
  loading: boolean;
  error: string | null;
}) {
  if (loading && !data) return <p className="text-sm text-muted-foreground">Loading model policy…</p>;
  if (error && !data) return <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</div>;
  if (!data?.status) return <p className="text-sm text-muted-foreground">Model policy status is unavailable.</p>;

  const { status, catalog } = data;
  return (
    <div className="space-y-5">
      <Card className="border-border bg-card/40">
        <CardContent className="space-y-2 py-5 text-sm">
          <div className="flex items-center gap-2">
            <Badge variant={status.ready ? "default" : "destructive"}>{status.ready ? "Ready" : "Not ready"}</Badge>
            <span className="font-mono text-xs break-all">{status.activeDigest || "no active revision"}</span>
          </div>
          <p><span className="text-muted-foreground">Path:</span> <span className="font-mono text-xs">{status.path}</span></p>
          {status.requirement?.required && <p><span className="text-muted-foreground">Required because:</span> {status.requirement.reason}</p>}
          {status.lastReloadAttempt?.diagnostic && (
            <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-destructive">
              {status.lastReloadAttempt.diagnostic.code}: {status.lastReloadAttempt.diagnostic.message}
            </div>
          )}
          <p className="text-xs text-muted-foreground">This view is read-only. Edit the declared catalog in Git, then use <code>agent-manager policy validate</code> and <code>agent-manager policy reload</code>.</p>
        </CardContent>
      </Card>

      {catalog && (
        <>
          <Card className="border-border bg-card/40">
            <CardContent className="space-y-2 py-5 text-sm">
              <h3 className="font-semibold">{catalog.metadata?.catalogId}</h3>
              <p className="text-muted-foreground">Default policy: {catalog.defaultPolicy}</p>
              {catalog.runners.map((runner) => (
                <div key={runner.runnerType} className="border-t border-border pt-2">
                  <div className="font-medium">{String(runner.runnerType)} · {runner.models.length} declared models</div>
                  <div className="text-xs text-muted-foreground">{runner.models.map((model) => model.id).join(", ")}</div>
                </div>
              ))}
            </CardContent>
          </Card>
          <Card className="border-border bg-card/40">
            <CardContent className="space-y-3 py-5 text-sm">
              <h3 className="font-semibold">Named policies</h3>
              {catalog.policies.map((policy) => (
                <div key={policy.name} className="border-t border-border pt-2">
                  <div className="font-medium">{policy.name} <span className="text-muted-foreground">({policy.intent})</span></div>
                  <ol className="mt-1 list-decimal pl-5 text-xs text-muted-foreground">
                    {policy.candidates.map((candidate, index) => (
                      <li key={`${policy.name}-${index}`}>{String(candidate.runnerType)} / {String(candidate.selectionType)}{candidate.model ? ` / ${candidate.model}` : ""}</li>
                    ))}
                  </ol>
                </div>
              ))}
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}
