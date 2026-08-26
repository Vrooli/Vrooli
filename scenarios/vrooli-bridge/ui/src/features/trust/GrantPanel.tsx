import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { create } from "@bufbuild/protobuf";

import {
  CreateGrantRequestSchema,
  grantsClient,
  ListGrantsRequestSchema,
  RevokeGrantRequestSchema,
  type CredentialGrant,
} from "../../api/grants";

const GRANTS_QUERY_KEY = ["trust", "credential-grants"] as const;

export function GrantPanel() {
  const queryClient = useQueryClient();
  const [nodeId, setNodeId] = useState("");
  const [logicalId, setLogicalId] = useState("");
  const [field, setField] = useState("");
  const [grantClass, setGrantClass] = useState("user_prompt");
  const [retention, setRetention] = useState("durable");
  const grants = useQuery({
    queryKey: GRANTS_QUERY_KEY,
    queryFn: async (): Promise<CredentialGrant[]> => (await grantsClient.listGrants(create(ListGrantsRequestSchema))).grants,
  });
  const createGrant = useMutation({
    mutationFn: () => grantsClient.createGrant(create(CreateGrantRequestSchema, { nodeId, logicalId, field, class: grantClass, retention, generation: 1n })),
    onSuccess: async () => { setLogicalId(""); setField(""); await queryClient.invalidateQueries({ queryKey: GRANTS_QUERY_KEY }); },
  });
  const revokeGrant = useMutation({
    mutationFn: (id: string) => grantsClient.revokeGrant(create(RevokeGrantRequestSchema, { id })),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: GRANTS_QUERY_KEY }),
  });

  return (
    <div className="flex flex-col gap-4" data-testid="credential-grants-panel">
      <form className="grid gap-2 rounded border border-app-border p-4 md:grid-cols-2" onSubmit={(event) => { event.preventDefault(); createGrant.mutate(); }}>
        <label>Node<input required value={nodeId} onChange={(event) => setNodeId(event.target.value)} /></label>
        <label>Logical identity<input required value={logicalId} onChange={(event) => setLogicalId(event.target.value)} placeholder="namespace/name" /></label>
        <label>Field<input required value={field} onChange={(event) => setField(event.target.value)} /></label>
        <label>Class<select value={grantClass} onChange={(event) => setGrantClass(event.target.value)}><option value="user_prompt">user_prompt</option><option value="remote_fetch">remote_fetch</option><option value="infrastructure">infrastructure</option></select></label>
        <label>Retention<select value={retention} onChange={(event) => setRetention(event.target.value)}><option value="durable">durable</option><option value="ephemeral">ephemeral</option></select></label>
        <button type="submit" disabled={createGrant.isPending}>Create metadata-only grant</button>
      </form>
      {grants.isError && <p role="alert">Unable to load credential grants.</p>}
      <div className="flex flex-col gap-2" aria-label="Credential grants">
        {(grants.data ?? []).map((grant) => (
          <div className="flex items-center justify-between rounded border border-app-border p-3" key={grant.id}>
            <span>{grant.nodeId} · {grant.logicalId}:{grant.field} · {grant.class}/{grant.retention} · generation {grant.generation.toString()} · acked {grant.ackedGeneration.toString()}</span>
            <button type="button" onClick={() => revokeGrant.mutate(grant.id)} disabled={revokeGrant.isPending}>Revoke</button>
          </div>
        ))}
        {!grants.isLoading && (grants.data ?? []).length === 0 && <p className="text-app-muted-foreground">No active grants.</p>}
      </div>
    </div>
  );
}
