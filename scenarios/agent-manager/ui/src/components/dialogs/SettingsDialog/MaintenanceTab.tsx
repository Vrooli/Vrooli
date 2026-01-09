// Maintenance tab with purge controls

import { Button } from "../../ui/button";
import { Card, CardContent } from "../../ui/card";
import { Input } from "../../ui/input";
import { Label } from "../../ui/label";
import { PurgeTarget } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";

interface MaintenanceTabProps {
  purgePattern: string;
  onPurgePatternChange: (pattern: string) => void;
  loading: boolean;
  error: string | null;
  onPurgePreview: (targets: PurgeTarget[], label: string) => void;
}

export function MaintenanceTab({
  purgePattern,
  onPurgePatternChange,
  loading,
  error,
  onPurgePreview,
}: MaintenanceTabProps) {
  return (
    <div className="space-y-6">
      <div className="space-y-3">
        <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">Danger Zone</h3>
        <Card className="border-destructive/40 bg-destructive/5">
          <CardContent className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="purgePattern">Regex Pattern</Label>
              <Input
                id="purgePattern"
                value={purgePattern}
                onChange={(event) => onPurgePatternChange(event.target.value)}
                placeholder="^test-.*"
              />
              <p className="text-xs text-muted-foreground">
                Matches profile keys (profiles), titles (tasks), and tags (runs; falls back to run ID if empty). Use <code>.*</code> to match everything.
              </p>
            </div>

            <div className="grid gap-2 sm:grid-cols-2">
              <Button
                variant="destructive"
                onClick={() => onPurgePreview([PurgeTarget.PROFILES], "Delete Profiles")}
                disabled={loading}
              >
                Delete Profiles
              </Button>
              <Button
                variant="destructive"
                onClick={() => onPurgePreview([PurgeTarget.TASKS], "Delete Tasks")}
                disabled={loading}
              >
                Delete Tasks
              </Button>
              <Button
                variant="destructive"
                onClick={() => onPurgePreview([PurgeTarget.RUNS], "Delete Runs")}
                disabled={loading}
              >
                Delete Runs
              </Button>
              <Button
                variant="destructive"
                onClick={() =>
                  onPurgePreview(
                    [PurgeTarget.PROFILES, PurgeTarget.TASKS, PurgeTarget.RUNS],
                    "Delete All"
                  )
                }
                disabled={loading}
              >
                Delete All
              </Button>
            </div>
            {error && (
              <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {error}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <div className="space-y-2 text-sm text-muted-foreground">
        <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">Service Controls</h3>
        <p>Future settings like pausing new runs or offline mode will live here.</p>
      </div>
    </div>
  );
}
