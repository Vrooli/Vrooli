import { useEffect, useState } from "react";
import type { AskAttachment } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { AskAttachment_Kind } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { fetchAttachment } from "../../api/attachments";

type ObservedPacket = {
  schemaVersion: "experience-packet-observed/v1";
  scenario: string;
  generatedAt: string;
  build: string | null;
  sessions: Array<{ id: string; build: string | null }>;
  window: { from: string | null; to: string | null; sessions: number };
  headline: string;
  topFlow: null | { steps: string[]; count: number; medianDurationMs: number };
  topFriction: null | { kind: string; count: number };
  timeCostMs: number;
  reportDigest: string;
  caveat: string;
};

function isObservedPacket(value: unknown): value is ObservedPacket {
  if (!value || typeof value !== "object") return false;
  const block = value as Partial<ObservedPacket>;
  return block.schemaVersion === "experience-packet-observed/v1"
    && typeof block.scenario === "string" && Boolean(block.scenario)
    && typeof block.generatedAt === "string"
    && (typeof block.build === "string" || block.build === null)
    && Array.isArray(block.sessions)
    && block.sessions.every((session) => typeof session?.id === "string" && Boolean(session.id) && (typeof session.build === "string" || session.build === null))
    && Boolean(block.window && Number.isInteger(block.window.sessions) && block.window.sessions >= 0)
    && typeof block.headline === "string"
    && (block.topFlow === null || Boolean(block.topFlow && Array.isArray(block.topFlow.steps) && Number.isFinite(block.topFlow.count) && Number.isFinite(block.topFlow.medianDurationMs)))
    && (block.topFriction === null || Boolean(block.topFriction && typeof block.topFriction.kind === "string" && Number.isFinite(block.topFriction.count)))
    && Number.isFinite(block.timeCostMs) && block.timeCostMs! >= 0
    && typeof block.reportDigest === "string" && /^sha256:[a-f0-9]{64}$/.test(block.reportDigest)
    && typeof block.caveat === "string" && Boolean(block.caveat);
}

export function ObservedPacketBlock({ askId, attachments }: { askId: string; attachments: readonly AskAttachment[] }) {
  const attachment = attachments.find((item) => item.name === "packet-observed.json" && item.kind === AskAttachment_Kind.FILE);
  const [block, setBlock] = useState<ObservedPacket>();
  const [error, setError] = useState("");

  useEffect(() => {
    setBlock(undefined);
    setError("");
    if (!attachment) return;
    if (attachment.sizeBytes > 262144n) {
      setError("Observed packet block exceeds the 256 KB display limit.");
      return;
    }
    const abort = new AbortController();
    void fetchAttachment(askId, attachment.name, "", abort.signal)
      .then(async (response) => {
        const value: unknown = await response.json();
        if (!isObservedPacket(value)) throw new Error("Unsupported or incomplete observed packet block.");
        setBlock(value);
      })
      .catch((failure: unknown) => {
        if (!abort.signal.aborted) setError(failure instanceof Error ? failure.message : "Observed packet block could not be evaluated.");
      });
    return () => abort.abort();
  }, [askId, attachment]);

  if (!attachment) return null;
  if (error) return <section aria-label="Observed behavior packet" role="status" className="rounded-panel border border-app-border bg-app-surface-muted p-3 text-sm">Observed behavior not evaluated: {error}</section>;
  if (!block) return <p role="status" className="text-sm text-app-muted-foreground">Loading optional observed packet context…</p>;

  return (
    <aside role="note" aria-label="Observed behavior packet context" className="grid gap-2 rounded-panel border border-app-border bg-app-surface-muted p-3">
      <div className="flex flex-wrap items-center gap-2"><h2 className="font-semibold">Observed behavior · advisory context</h2><span className="rounded-full border border-app-border px-2 py-0.5 text-xs">not a fidelity label</span></div>
      <p className="text-sm">{block.headline}</p>
      <p className="text-sm">Top flow: {block.topFlow ? `${block.topFlow.steps.join(" → ")} · ${block.topFlow.count}x · median ${Math.round(block.topFlow.medianDurationMs / 1000)}s` : "none observed"}</p>
      <p className="text-sm">Scenario: <code>{block.scenario}</code> · time cost: {Math.round(block.timeCostMs / 1000)}s · friction: {block.topFriction ? `${block.topFriction.kind} (${block.topFriction.count})` : "none"}</p>
      <p className="text-sm">Build: <code>{block.build ?? "not available"}</code> · {block.sessions.length} session(s) · report digest: <code className="break-all">{block.reportDigest}</code></p>
      <details className="text-xs text-app-muted-foreground"><summary className="min-h-11 cursor-pointer py-2">Session references and caveat</summary><ul className="list-inside list-disc">{block.sessions.map((session) => <li key={session.id}><code>{session.id}</code> · {session.build ?? "build not available"}</li>)}</ul><p className="mt-2">{block.caveat}</p></details>
    </aside>
  );
}
