import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Workflow, ExternalLink, CheckCircle2 } from "lucide-react";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Label } from "../../components/ui/label";
import { Badge } from "../../components/ui/badge";
import { reportMigrationTask, type MigrationTaskFeedback } from "../../lib/api";
import { getErrorMessage } from "../../lib/utils";

interface MigrationTaskCardProps {
  // defaultScenario pre-fills the target scenario (e.g. the deployment profile).
  defaultScenario?: string;
}

// statusVariant maps a backlog item status to a badge variant.
function statusVariant(status: string): "default" | "success" | "warning" | "secondary" {
  switch (status.toLowerCase()) {
    case "done":
    case "completed":
      return "success";
    case "in_progress":
    case "running":
      return "warning";
    case "queued":
    case "pending":
      return "default";
    default:
      return "secondary";
  }
}

// MigrationTaskCard lets an operator file the source-code migration that a
// dependency swap requires. It posts to deployment-manager's /migration-tasks
// endpoint, which files a swarm-manager backlog `fix` item, and renders the live
// feedback (status, queue position, deep link).
export function MigrationTaskCard({ defaultScenario }: MigrationTaskCardProps) {
  const [scenario, setScenario] = useState(defaultScenario ?? "");
  const [fromDependency, setFromDependency] = useState("");
  const [toDependency, setToDependency] = useState("");
  const [notes, setNotes] = useState("");

  const mutation = useMutation<MigrationTaskFeedback, Error>({
    mutationFn: () =>
      reportMigrationTask({
        scenario: scenario.trim(),
        from_dependency: fromDependency.trim(),
        to_dependency: toDependency.trim(),
        profile_id: defaultScenario,
        notes: notes.trim() || undefined,
      }),
  });

  const feedback = mutation.data;
  const canSubmit = scenario.trim() && fromDependency.trim() && toDependency.trim();

  return (
    <div className="rounded-lg border border-white/10 bg-white/5 p-3 space-y-3">
      <div className="flex items-center gap-2 text-sm font-semibold">
        <Workflow className="h-4 w-4" />
        File a migration task in the backlog
      </div>
      <p className="text-sm text-slate-400">
        Swaps only touch deployment profiles. File the source-code migration as a swarm-manager backlog
        task so an agent can pick it up; track its live status here.
      </p>

      <div className="grid gap-2 sm:grid-cols-3">
        <div className="space-y-1">
          <Label htmlFor="mt-scenario" className="text-xs">Scenario</Label>
          <Input
            id="mt-scenario"
            value={scenario}
            onChange={(e) => setScenario(e.target.value)}
            placeholder="my-scenario"
          />
        </div>
        <div className="space-y-1">
          <Label htmlFor="mt-from" className="text-xs">From dependency</Label>
          <Input
            id="mt-from"
            value={fromDependency}
            onChange={(e) => setFromDependency(e.target.value)}
            placeholder="redis"
          />
        </div>
        <div className="space-y-1">
          <Label htmlFor="mt-to" className="text-xs">To dependency</Label>
          <Input
            id="mt-to"
            value={toDependency}
            onChange={(e) => setToDependency(e.target.value)}
            placeholder="valkey"
          />
        </div>
      </div>
      <div className="space-y-1">
        <Label htmlFor="mt-notes" className="text-xs">Notes (optional)</Label>
        <Input
          id="mt-notes"
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
          placeholder="Migration rationale or context"
        />
      </div>

      <Button
        size="sm"
        onClick={() => mutation.mutate()}
        disabled={!canSubmit || mutation.isPending}
        className="gap-2"
      >
        <Workflow className="h-4 w-4" />
        {mutation.isPending ? "Filing..." : "File migration task"}
      </Button>

      {mutation.isError && (
        <div className="rounded border border-red-500/30 bg-red-500/10 p-2 text-xs text-red-100">
          {getErrorMessage(mutation.error)}
        </div>
      )}

      {feedback && (
        <div className="rounded-lg border border-white/10 bg-slate-900/60 p-3 space-y-2">
          <div className="flex items-center gap-2 text-sm font-semibold text-slate-100">
            <CheckCircle2 className="h-4 w-4 text-green-400" />
            {feedback.deduped ? "Linked to existing backlog item" : "Backlog item created"}
          </div>
          <div className="flex flex-wrap items-center gap-2 text-xs">
            <Badge variant={statusVariant(feedback.status)}>{feedback.status || "unknown"}</Badge>
            <Badge variant="outline">priority {feedback.priority}</Badge>
            {typeof feedback.queue_position === "number" && (
              <Badge variant="secondary">{feedback.queue_position} ahead in queue</Badge>
            )}
          </div>
          <div className="text-xs text-slate-400 break-all">{feedback.item_id}</div>
          <a
            href={feedback.deep_link}
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-1 text-xs text-blue-300 hover:underline"
          >
            Open in swarm-manager backlog
            <ExternalLink className="h-3 w-3" />
          </a>
        </div>
      )}
    </div>
  );
}
