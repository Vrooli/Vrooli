// Orchestration settings tab - configure run lifecycle, safety, health detection, and termination

import * as React from "react";
import { useCallback, useEffect, useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../ui/card";
import { Input } from "../../ui/input";
import { Label } from "../../ui/label";
import { Checkbox } from "../../ui/checkbox";
import type { OrchestrationSettings } from "../../../hooks/useOrchestrationSettings";

export interface OrchestrationTabHandle {
  hasChanges: boolean;
  saving: boolean;
  save: () => Promise<void>;
  reset: () => Promise<void>;
}

interface OrchestrationTabProps {
  settings: OrchestrationSettings | null;
  loading: boolean;
  error: string | null;
  onSave: (settings: OrchestrationSettings) => Promise<OrchestrationSettings | void>;
  onReset: () => Promise<OrchestrationSettings | void>;
  onDirtyChange?: (dirty: boolean) => void;
}

function cloneSettings(s: OrchestrationSettings): OrchestrationSettings {
  return JSON.parse(JSON.stringify(s)) as OrchestrationSettings;
}

export const OrchestrationTab = React.forwardRef<OrchestrationTabHandle, OrchestrationTabProps>(
  function OrchestrationTab({ settings, loading, error, onSave, onReset, onDirtyChange }, ref) {
    const [draft, setDraft] = useState<OrchestrationSettings | null>(null);
    const [saving, setSaving] = useState(false);
    const [localError, setLocalError] = useState<string | null>(null);
    const [validationErrors, setValidationErrors] = useState<string[]>([]);

    // Sync draft from settings when they load
    useEffect(() => {
      if (settings) {
        setDraft(cloneSettings(settings));
      }
    }, [settings]);

    const hasChanges = !!(
      settings &&
      draft &&
      JSON.stringify(draft) !== JSON.stringify(settings)
    );

    // Notify parent of dirty state changes
    useEffect(() => {
      onDirtyChange?.(hasChanges);
    }, [hasChanges, onDirtyChange]);

    const validate = useCallback((d: OrchestrationSettings): string[] => {
      const errors: string[] = [];
      const { healthDetection, processTermination } = d;

      if (healthDetection.heartbeatIntervalSeconds >= healthDetection.staleThresholdSeconds) {
        errors.push(
          "Heartbeat interval must be less than stale threshold.",
        );
      }
      if (healthDetection.staleThresholdSeconds >= healthDetection.maxRecoveryAgeSeconds) {
        errors.push(
          "Stale threshold must be less than max recovery age.",
        );
      }
      const terminationWindow =
        processTermination.gracePeriodSeconds * processTermination.terminationMaxRetries;
      if (terminationWindow >= healthDetection.staleThresholdSeconds) {
        errors.push(
          `Grace period (${processTermination.gracePeriodSeconds}s) x max retries (${processTermination.terminationMaxRetries}) = ${terminationWindow}s must be less than stale threshold (${healthDetection.staleThresholdSeconds}s).`,
        );
      }
      return errors;
    }, []);

    const handleSave = useCallback(async () => {
      if (!draft) return;
      const errors = validate(draft);
      setValidationErrors(errors);
      if (errors.length > 0) return;

      setSaving(true);
      setLocalError(null);
      try {
        await onSave(draft);
      } catch (err) {
        setLocalError((err as Error).message);
      } finally {
        setSaving(false);
      }
    }, [draft, onSave, validate]);

    const handleReset = useCallback(async () => {
      setLocalError(null);
      setValidationErrors([]);
      try {
        await onReset();
      } catch (err) {
        setLocalError((err as Error).message);
      }
    }, [onReset]);

    // Expose imperative handle for unified footer
    React.useImperativeHandle(
      ref,
      () => ({
        hasChanges,
        saving,
        save: handleSave,
        reset: handleReset,
      }),
      [hasChanges, saving, handleSave, handleReset],
    );

    // Helper to update a nested group field
    const updateDraft = useCallback(
      <G extends keyof OrchestrationSettings>(
        group: G,
        field: keyof OrchestrationSettings[G],
        value: OrchestrationSettings[G][keyof OrchestrationSettings[G]],
      ) => {
        setDraft((prev) => {
          if (!prev) return prev;
          return {
            ...prev,
            [group]: {
              ...prev[group],
              [field]: value,
            },
          };
        });
      },
      [],
    );

    if (loading && !settings) {
      return (
        <div className="flex items-center justify-center py-8 text-muted-foreground">
          Loading orchestration settings...
        </div>
      );
    }

    if (!draft) {
      return (
        <div className="flex items-center justify-center py-8 text-muted-foreground">
          No orchestration settings available.
        </div>
      );
    }

    const displayError = localError || error;

    return (
      <div className="space-y-6">
        {/* Card 1: Run Execution Limits */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Run Execution Limits</CardTitle>
            <CardDescription className="text-xs">
              Control how long and how many agent runs can execute simultaneously
            </CardDescription>
          </CardHeader>
          <CardContent className="pt-0">
            <div className="grid gap-4 sm:grid-cols-3">
              <div className="space-y-2">
                <Label htmlFor="runTimeoutMinutes">Run Timeout</Label>
                <div className="flex items-center gap-2">
                  <Input
                    id="runTimeoutMinutes"
                    type="number"
                    min={1}
                    max={9999}
                    step={1}
                    value={draft.runExecution.runTimeoutMinutes}
                    onChange={(e) =>
                      updateDraft("runExecution", "runTimeoutMinutes", Number(e.target.value))
                    }
                    className="h-8"
                  />
                  <span className="text-xs text-muted-foreground shrink-0">minutes</span>
                </div>
                <p className="text-xs text-muted-foreground">
                  Maximum wall-clock time per run. Longer values allow complex tasks but tie up
                  resources.
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="maxConcurrentRuns">Max Concurrent Runs</Label>
                <div className="flex items-center gap-2">
                  <Input
                    id="maxConcurrentRuns"
                    type="number"
                    min={1}
                    max={9999}
                    step={1}
                    value={draft.runExecution.maxConcurrentRuns}
                    onChange={(e) =>
                      updateDraft("runExecution", "maxConcurrentRuns", Number(e.target.value))
                    }
                    className="h-8"
                  />
                  <span className="text-xs text-muted-foreground shrink-0">runs</span>
                </div>
                <p className="text-xs text-muted-foreground">
                  Maximum simultaneous runs across all scopes. Higher values increase throughput but
                  use more resources.
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="maxTurns">Max Turns</Label>
                <div className="flex items-center gap-2">
                  <Input
                    id="maxTurns"
                    type="number"
                    min={1}
                    max={9999}
                    step={1}
                    value={draft.runExecution.maxTurns}
                    onChange={(e) =>
                      updateDraft("runExecution", "maxTurns", Number(e.target.value))
                    }
                    className="h-8"
                  />
                  <span className="text-xs text-muted-foreground shrink-0">turns</span>
                </div>
                <p className="text-xs text-muted-foreground">
                  Maximum conversation turns per agent. Higher values allow deeper reasoning but
                  increase cost and time.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Card 2: Safety & Isolation */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Safety &amp; Isolation</CardTitle>
            <CardDescription className="text-xs">
              Control how locked down agent runs are
            </CardDescription>
          </CardHeader>
          <CardContent className="pt-0">
            <div className="grid gap-4 sm:grid-cols-3">
              <label className="flex cursor-pointer items-start gap-3 rounded-lg border border-border p-3 transition-colors hover:bg-accent/50">
                <Checkbox
                  checked={draft.safetyIsolation.requireSandbox}
                  onCheckedChange={(checked) =>
                    updateDraft("safetyIsolation", "requireSandbox", checked === true)
                  }
                />
                <div>
                  <div className="text-sm font-medium">Require Sandbox</div>
                  <p className="text-xs text-muted-foreground">
                    Run agents in bwrap filesystem isolation. Disabling is faster but removes
                    filesystem protection.
                  </p>
                </div>
              </label>

              <label className="flex cursor-pointer items-start gap-3 rounded-lg border border-border p-3 transition-colors hover:bg-accent/50">
                <Checkbox
                  checked={draft.safetyIsolation.requireApproval}
                  onCheckedChange={(checked) =>
                    updateDraft("safetyIsolation", "requireApproval", checked === true)
                  }
                />
                <div>
                  <div className="text-sm font-medium">Require Approval</div>
                  <p className="text-xs text-muted-foreground">
                    Require human review before applying changes. Disabling enables fully autonomous
                    operation.
                  </p>
                </div>
              </label>

              <div className="space-y-2">
                <Label htmlFor="networkAccess">Network Access</Label>
                <select
                  id="networkAccess"
                  value={draft.safetyIsolation.networkAccess}
                  onChange={(e) =>
                    updateDraft(
                      "safetyIsolation",
                      "networkAccess",
                      e.target.value as "none" | "localhost" | "full",
                    )
                  }
                  className="flex h-8 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                >
                  <option value="none">None</option>
                  <option value="localhost">Localhost only</option>
                  <option value="full">Full</option>
                </select>
                <p className="text-xs text-muted-foreground">
                  Controls what network the agent can reach. &apos;Full&apos; enables fetching docs
                  and APIs but increases exfiltration risk.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Card 3: Health Detection */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Health Detection</CardTitle>
            <CardDescription className="text-xs">
              Control how quickly the system detects stuck or dead runs
            </CardDescription>
          </CardHeader>
          <CardContent className="pt-0">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="heartbeatIntervalSeconds">Heartbeat Interval</Label>
                <div className="flex items-center gap-2">
                  <Input
                    id="heartbeatIntervalSeconds"
                    type="number"
                    min={1}
                    max={9999}
                    step={1}
                    value={draft.healthDetection.heartbeatIntervalSeconds}
                    onChange={(e) =>
                      updateDraft(
                        "healthDetection",
                        "heartbeatIntervalSeconds",
                        Number(e.target.value),
                      )
                    }
                    className="h-8"
                  />
                  <span className="text-xs text-muted-foreground shrink-0">seconds</span>
                </div>
                <p className="text-xs text-muted-foreground">
                  How often the executor pings the database. Shorter means faster stale detection but
                  more database writes.
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="staleThresholdSeconds">Stale Threshold</Label>
                <div className="flex items-center gap-2">
                  <Input
                    id="staleThresholdSeconds"
                    type="number"
                    min={1}
                    max={99999}
                    step={1}
                    value={draft.healthDetection.staleThresholdSeconds}
                    onChange={(e) =>
                      updateDraft(
                        "healthDetection",
                        "staleThresholdSeconds",
                        Number(e.target.value),
                      )
                    }
                    className="h-8"
                  />
                  <span className="text-xs text-muted-foreground shrink-0">seconds</span>
                </div>
                <p className="text-xs text-muted-foreground">
                  Time without heartbeat before a run is considered stale. Must be greater than
                  heartbeat interval.
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="maxRecoveryAgeSeconds">Max Recovery Age</Label>
                <div className="flex items-center gap-2">
                  <Input
                    id="maxRecoveryAgeSeconds"
                    type="number"
                    min={1}
                    max={99999}
                    step={1}
                    value={draft.healthDetection.maxRecoveryAgeSeconds}
                    onChange={(e) =>
                      updateDraft(
                        "healthDetection",
                        "maxRecoveryAgeSeconds",
                        Number(e.target.value),
                      )
                    }
                    className="h-8"
                  />
                  <span className="text-xs text-muted-foreground shrink-0">seconds</span>
                </div>
                <p className="text-xs text-muted-foreground">
                  Maximum time a stale run with a live process can persist before being killed. Must
                  be greater than stale threshold.
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="reconcilerIntervalSeconds">Reconciler Interval</Label>
                <div className="flex items-center gap-2">
                  <Input
                    id="reconcilerIntervalSeconds"
                    type="number"
                    min={1}
                    max={99999}
                    step={1}
                    value={draft.healthDetection.reconcilerIntervalSeconds}
                    onChange={(e) =>
                      updateDraft(
                        "healthDetection",
                        "reconcilerIntervalSeconds",
                        Number(e.target.value),
                      )
                    }
                    className="h-8"
                  />
                  <span className="text-xs text-muted-foreground shrink-0">seconds</span>
                </div>
                <p className="text-xs text-muted-foreground">
                  How often the background sweep checks for stuck runs and orphan processes.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Card 4: Process Termination */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Process Termination</CardTitle>
            <CardDescription className="text-xs">
              Control how aggressively the system kills stuck processes
            </CardDescription>
          </CardHeader>
          <CardContent className="pt-0 space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="gracePeriodSeconds">Grace Period</Label>
                <div className="flex items-center gap-2">
                  <Input
                    id="gracePeriodSeconds"
                    type="number"
                    min={1}
                    max={9999}
                    step={1}
                    value={draft.processTermination.gracePeriodSeconds}
                    onChange={(e) =>
                      updateDraft(
                        "processTermination",
                        "gracePeriodSeconds",
                        Number(e.target.value),
                      )
                    }
                    className="h-8"
                  />
                  <span className="text-xs text-muted-foreground shrink-0">seconds</span>
                </div>
                <p className="text-xs text-muted-foreground">
                  Wait time between SIGTERM and SIGKILL. Longer allows cleaner shutdown but delays
                  cleanup.
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="orphanGracePeriodSeconds">Orphan Grace Period</Label>
                <div className="flex items-center gap-2">
                  <Input
                    id="orphanGracePeriodSeconds"
                    type="number"
                    min={1}
                    max={9999}
                    step={1}
                    value={draft.processTermination.orphanGracePeriodSeconds}
                    onChange={(e) =>
                      updateDraft(
                        "processTermination",
                        "orphanGracePeriodSeconds",
                        Number(e.target.value),
                      )
                    }
                    className="h-8"
                  />
                  <span className="text-xs text-muted-foreground shrink-0">seconds</span>
                </div>
                <p className="text-xs text-muted-foreground">
                  How long an untracked process can live before being killed.
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="terminationMaxRetries">Termination Max Retries</Label>
                <div className="flex items-center gap-2">
                  <Input
                    id="terminationMaxRetries"
                    type="number"
                    min={1}
                    max={99}
                    step={1}
                    value={draft.processTermination.terminationMaxRetries}
                    onChange={(e) =>
                      updateDraft(
                        "processTermination",
                        "terminationMaxRetries",
                        Number(e.target.value),
                      )
                    }
                    className="h-8"
                  />
                  <span className="text-xs text-muted-foreground shrink-0">attempts</span>
                </div>
                <p className="text-xs text-muted-foreground">
                  Number of escalation attempts before giving up on killing a process.
                </p>
              </div>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <label className="flex cursor-pointer items-start gap-3 rounded-lg border border-border p-3 transition-colors hover:bg-accent/50">
                <Checkbox
                  checked={draft.processTermination.killProcessGroup}
                  onCheckedChange={(checked) =>
                    updateDraft("processTermination", "killProcessGroup", checked === true)
                  }
                />
                <div>
                  <div className="text-sm font-medium">Kill Process Group</div>
                  <p className="text-xs text-muted-foreground">
                    Kill the entire process tree, not just the main process. Prevents orphan child
                    processes.
                  </p>
                </div>
              </label>

              <label className="flex cursor-pointer items-start gap-3 rounded-lg border border-border p-3 transition-colors hover:bg-accent/50">
                <Checkbox
                  checked={draft.processTermination.killOrphans}
                  onCheckedChange={(checked) =>
                    updateDraft("processTermination", "killOrphans", checked === true)
                  }
                />
                <div>
                  <div className="text-sm font-medium">Kill Orphans</div>
                  <p className="text-xs text-muted-foreground">
                    Automatically kill agent processes that have no corresponding database record.
                  </p>
                </div>
              </label>
            </div>
          </CardContent>
        </Card>

        {/* Validation errors */}
        {validationErrors.length > 0 && (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 space-y-1">
            {validationErrors.map((err) => (
              <p key={err} className="text-sm text-destructive">
                {err}
              </p>
            ))}
          </div>
        )}

        {/* Error display */}
        {displayError && (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            {displayError}
          </div>
        )}
      </div>
    );
  },
);
