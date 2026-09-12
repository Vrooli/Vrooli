import { useParams } from "react-router-dom";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";

type PipelinePageProps = { kind: "package" | "evidence" | "publication" | "attribution" };

const copy = {
  package: {
    title: "Package detail",
    description: "The package identity is digest-bound; downstream gates remain closed until they run.",
    items: ["Composition: not run", "Conformance: not run", "Attestation: not run", "Rehearsal: not run"],
  },
  evidence: {
    title: "Evidence review",
    description: "Only redacted, reference-based evidence belongs in this surface.",
    items: ["Protocol capture: not run", "Target verdict: unavailable", "Source revision: not supplied"],
  },
  publication: {
    title: "Publication history",
    description: "A publication is recorded only after an external gate and retrieval confirmation.",
    items: ["Deployment-manager gate: not run", "OCI retrieval: not run", "Revocation: no publication recorded"],
  },
  attribution: {
    title: "Attribution",
    description: "This ramp records package and evidence references, never credentials or opaque artifact bytes.",
    items: ["Artifact digest: unavailable", "Scanner authority: deployment-managed", "Workspace data: isolated per rehearsal"],
  },
} as const;

export function PipelinePage({ kind }: PipelinePageProps) {
  const { packageId } = useParams();
  const view = copy[kind];
  return (
    <section className="flex flex-col gap-5" data-testid={`page-${kind}`} aria-labelledby={`${kind}-heading`}>
      <div>
        <h2 id={`${kind}-heading`} className="text-2xl font-semibold">{view.title}</h2>
        <p className="mt-1 text-app-muted-foreground">{view.description}</p>
      </div>
      {packageId && <p className="font-mono text-xs text-app-muted-foreground">Package: {packageId}</p>}
      <Card>
        <CardHeader>
          <CardTitle>Gate state</CardTitle>
          <CardDescription>Not run is intentionally distinct from passed, failed, and unavailable.</CardDescription>
        </CardHeader>
        <CardContent>
          <ul className="space-y-2">
            {view.items.map((item) => <li key={item} className="rounded-control border border-app-border px-3 py-2 text-sm">{item}</li>)}
          </ul>
        </CardContent>
      </Card>
    </section>
  );
}
