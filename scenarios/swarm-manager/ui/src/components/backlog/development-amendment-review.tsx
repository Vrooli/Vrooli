import { toJsonString } from "@bufbuild/protobuf";
import { PreviewDevelopmentRequestSchema } from "@vrooli/proto-types/swarm-manager/v1/api/transition_pb";
import type { PreviewDevelopmentRequest, DevelopmentArtifact } from "@vrooli/proto-types/swarm-manager/v1/api/transition_pb";

type Change = { field: string; before: string; after: string };

/** Compare owner-provided values; never re-derive target contents in the browser. */
function developmentAmendmentChanges(previous: PreviewDevelopmentRequest, proposed: PreviewDevelopmentRequest, retained: DevelopmentArtifact[], resolved: DevelopmentArtifact[]): Change[] {
  const before = JSON.parse(toJsonString(PreviewDevelopmentRequestSchema, previous, { useProtoFieldName: true })) as Record<string, unknown>;
  const after = JSON.parse(toJsonString(PreviewDevelopmentRequestSchema, proposed, { useProtoFieldName: true })) as Record<string, unknown>;
  const show = (value: unknown) => value === undefined ? "Not set" : JSON.stringify(value, null, 2);
  const changes: Change[] = [];
  for (const field of [...new Set([...Object.keys(before), ...Object.keys(after)])].sort()) {
    if (show(before[field]) !== show(after[field])) changes.push({ field, before: show(before[field]), after: show(after[field]) });
  }
  const oldArtifacts = new Map(retained.map((artifact) => [artifact.path, artifact.sha256]));
  const newArtifacts = new Map(resolved.map((artifact) => [artifact.path, artifact.sha256]));
  for (const path of [...new Set([...oldArtifacts.keys(), ...newArtifacts.keys()])].sort()) {
    if (oldArtifacts.get(path) !== newArtifacts.get(path)) changes.push({ field: `Artifact: ${path}`, before: oldArtifacts.get(path) ?? "Not included", after: newArtifacts.get(path) ?? "Removed" });
  }
  return changes;
}

export function DevelopmentAmendmentReview({ previous, proposed, retained, resolved }: {
  previous: PreviewDevelopmentRequest; proposed: PreviewDevelopmentRequest; retained: DevelopmentArtifact[]; resolved: DevelopmentArtifact[];
}) {
  const changes = developmentAmendmentChanges(previous, proposed, retained, resolved);
  return <section aria-label="Amendment changes" className="space-y-2">
    <h3 className="font-semibold">Changes from the retained approval</h3>
    <p>Review outcome, scope, effects, guidance, budget and artifact changes. The current grant remains active until an amendment is approved. Existing usage is not reset.</p>
    {changes.length === 0 ? <p>No proposal fields or artifact hashes changed. The generated goal may still differ; review it below.</p> : <div className="overflow-x-auto"><table className="w-full text-left text-xs">
      <thead><tr><th className="p-2">Field</th><th className="p-2">Retained</th><th className="p-2">Proposed</th></tr></thead>
      <tbody>{changes.map((change) => <tr key={change.field} className="border-t border-slate-700">
        <th className="max-w-48 break-words p-2 align-top">{change.field}</th>
        <td className="max-w-64 p-2 align-top"><pre className="whitespace-pre-wrap break-words">{change.before}</pre></td>
        <td className="max-w-64 p-2 align-top"><pre className="whitespace-pre-wrap break-words">{change.after}</pre></td>
      </tr>)}</tbody>
    </table></div>}
  </section>;
}
