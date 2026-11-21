import { RunLadder } from "./RunLadder";

/** A setup run part-way through: one phase done, one running, one item failed. */
export function InProgress() {
  return (
    <main data-rcl-story-background>
      <RunLadder
        runId="run-2026-10-30"
        resumeNote="Resumed after reconnect"
        phases={[
          { id: "resources", label: "Resources", state: "complete", durationMs: 4200 },
          { id: "scenarios", label: "Scenarios", state: "active", detail: "3 of 5 started" },
          { id: "verify", label: "Verify", state: "pending" },
        ]}
        items={[
          { name: "postgres", outcome: "succeeded" },
          { name: "web-console", outcome: "running" },
          { name: "audio-tools", outcome: "failed", error: "Port 8020 is already in use." },
        ]}
      />
    </main>
  );
}
