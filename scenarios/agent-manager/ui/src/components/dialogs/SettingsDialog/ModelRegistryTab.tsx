// Model registry configuration tab

import { Badge } from "../../ui/badge";
import { Button } from "../../ui/button";
import { Card, CardContent } from "../../ui/card";
import { Input } from "../../ui/input";
import { Label } from "../../ui/label";
import type { ModelRegistry } from "../../../types";
import { normalizeModelOptions } from "../../../lib/modelRegistry";
import { runnerTypeFromSlug, runnerTypeLabel } from "../../../lib/utils";

interface ModelRegistryTabProps {
  draft: ModelRegistry | null;
  loading: boolean;
  loadError: string | null;
  error: string | null;
  newRunnerKey: string;
  onNewRunnerKeyChange: (key: string) => void;
  knownRunners: Record<string, unknown> | undefined;
  onAddRunner: () => void;
  onRemoveRunner: (runnerKey: string) => void;
  onAddFallbackRunner: () => void;
  onUpdateFallbackRunner: (index: number, value: string) => void;
  onRemoveFallbackRunner: (index: number) => void;
  onAddModel: (runnerKey: string) => void;
  onRemoveModel: (runnerKey: string, index: number) => void;
  onUpdateModel: (runnerKey: string, index: number, field: "id" | "description", value: string) => void;
  onUpdatePreset: (runnerKey: string, presetKey: string, value: string) => void;
}

export function ModelRegistryTab({
  draft,
  loading,
  loadError,
  error,
  newRunnerKey,
  onNewRunnerKeyChange,
  knownRunners,
  onAddRunner,
  onRemoveRunner,
  onAddFallbackRunner,
  onUpdateFallbackRunner,
  onRemoveFallbackRunner,
  onAddModel,
  onRemoveModel,
  onUpdateModel,
  onUpdatePreset,
}: ModelRegistryTabProps) {
  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="space-y-1">
          <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">Model Registry</h3>
          <p className="text-sm text-muted-foreground">
            Configure per-runner model lists and map Fast/Cheap/Smart presets to model IDs. Updates apply across all scenarios that use agent-manager.
          </p>
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Badge variant="secondary">FAST</Badge>
          <Badge variant="secondary">CHEAP</Badge>
          <Badge variant="secondary">SMART</Badge>
        </div>
      </div>

      {!draft && loading && (
        <p className="text-sm text-muted-foreground">Loading model registry...</p>
      )}
      {!draft && loadError && (
        <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {loadError}
        </div>
      )}

      {draft && (
        <div className="space-y-5">
          <FallbackRunnersCard
            draft={draft}
            onAdd={onAddFallbackRunner}
            onUpdate={onUpdateFallbackRunner}
            onRemove={onRemoveFallbackRunner}
          />

          {Object.entries(draft.runners).length === 0 && (
            <Card className="border-border bg-card/40">
              <CardContent className="py-6 text-sm text-muted-foreground">
                No runners are configured yet. Add your first runner to build a model list.
              </CardContent>
            </Card>
          )}

          {Object.entries(draft.runners)
            .sort(([a], [b]) => a.localeCompare(b))
            .map(([runnerKey, runnerConfig]) => (
              <RunnerCard
                key={runnerKey}
                runnerKey={runnerKey}
                runnerConfig={runnerConfig}
                onRemoveRunner={onRemoveRunner}
                onAddModel={onAddModel}
                onRemoveModel={onRemoveModel}
                onUpdateModel={onUpdateModel}
                onUpdatePreset={onUpdatePreset}
              />
            ))}

          <AddRunnerCard
            newRunnerKey={newRunnerKey}
            onNewRunnerKeyChange={onNewRunnerKeyChange}
            onAddRunner={onAddRunner}
            knownRunners={knownRunners}
          />

          {error && (
            <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              {error}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function FallbackRunnersCard({
  draft,
  onAdd,
  onUpdate,
  onRemove,
}: {
  draft: ModelRegistry;
  onAdd: () => void;
  onUpdate: (index: number, value: string) => void;
  onRemove: (index: number) => void;
}) {
  const fallbackRunnerTypes = draft.fallbackRunnerTypes ?? [];
  const runnerOptions = Array.from(
    new Set([
      ...Object.keys(draft.runners),
      "claude-code",
      "codex",
      "opencode",
    ])
  ).sort((a, b) => a.localeCompare(b));

  return (
    <Card className="border-border bg-card/40">
      <CardContent className="space-y-4 py-5">
        <div className="space-y-1">
          <h3 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">Global Fallbacks</h3>
          <p className="text-xs text-muted-foreground">
            Ordered runner list to try when the primary runner is unavailable. Empty disables fallback.
          </p>
        </div>
        <div className="space-y-2">
          {fallbackRunnerTypes.length === 0 ? (
            <p className="text-xs text-muted-foreground">No global fallbacks configured.</p>
          ) : (
            fallbackRunnerTypes.map((runnerKey, index) => {
              const resolvedRunner = runnerTypeFromSlug(runnerKey);
              const label = resolvedRunner ? runnerTypeLabel(resolvedRunner) : runnerKey;
              return (
                <div key={`fallback-${index}`} className="flex flex-wrap items-center gap-2">
                  <select
                    value={runnerKey}
                    onChange={(event) => onUpdate(index, event.target.value)}
                    className="flex h-10 min-w-[200px] flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    <option value="">Select runner...</option>
                    {runnerOptions.map((option) => {
                      const resolvedOption = runnerTypeFromSlug(option);
                      const optionLabel = resolvedOption ? runnerTypeLabel(resolvedOption) : option;
                      return (
                        <option key={option} value={option}>
                          {optionLabel}
                        </option>
                      );
                    })}
                  </select>
                  <Badge variant="secondary">{label}</Badge>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onRemove(index)}
                  >
                    Remove
                  </Button>
                </div>
              );
            })
          )}
          <div>
            <Button variant="outline" size="sm" onClick={onAdd}>
              Add Fallback
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

interface RunnerConfig {
  models: Array<string | { id: string; description?: string }>;
  presets?: Record<string, string>;
}

function RunnerCard({
  runnerKey,
  runnerConfig,
  onRemoveRunner,
  onAddModel,
  onRemoveModel,
  onUpdateModel,
  onUpdatePreset,
}: {
  runnerKey: string;
  runnerConfig: RunnerConfig;
  onRemoveRunner: (runnerKey: string) => void;
  onAddModel: (runnerKey: string) => void;
  onRemoveModel: (runnerKey: string, index: number) => void;
  onUpdateModel: (runnerKey: string, index: number, field: "id" | "description", value: string) => void;
  onUpdatePreset: (runnerKey: string, presetKey: string, value: string) => void;
}) {
  const models = normalizeModelOptions(runnerConfig.models);
  const modelIds = models.map((model) => model.id).filter((id) => id);
  const presets = runnerConfig.presets ?? {};

  return (
    <Card className="border-border bg-card/40">
      <CardContent className="space-y-4 py-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <Badge variant="secondary">{runnerKey}</Badge>
              <span className="text-xs text-muted-foreground">
                {models.length} model{models.length === 1 ? "" : "s"}
              </span>
            </div>
            <p className="text-xs text-muted-foreground">Configure models and presets for this runner.</p>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onRemoveRunner(runnerKey)}
          >
            Remove Runner
          </Button>
        </div>

        <div className="space-y-3 rounded-md border border-border bg-background/60 p-3">
          <div className="grid gap-2 text-[11px] font-semibold uppercase text-muted-foreground sm:grid-cols-[1fr_1.5fr_auto]">
            <span>Model ID</span>
            <span>Description</span>
            <span className="sr-only">Actions</span>
          </div>
          {models.length === 0 ? (
            <p className="text-xs text-muted-foreground">No models yet. Add one to populate the list.</p>
          ) : (
            models.map((model, index) => (
              <div
                key={`${runnerKey}-${index}`}
                className="grid gap-2 sm:grid-cols-[1fr_1.5fr_auto] sm:items-end"
              >
                <Input
                  value={model.id}
                  onChange={(event) => onUpdateModel(runnerKey, index, "id", event.target.value)}
                  placeholder="gpt-5.1-codex"
                />
                <Input
                  value={model.description}
                  onChange={(event) => onUpdateModel(runnerKey, index, "description", event.target.value)}
                  placeholder="Short description"
                />
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onRemoveModel(runnerKey, index)}
                >
                  Remove
                </Button>
              </div>
            ))
          )}
          <div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => onAddModel(runnerKey)}
            >
              Add Model
            </Button>
          </div>
        </div>

        <div className="space-y-3 rounded-md border border-border bg-background/60 p-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <Label>Preset Mappings</Label>
            <p className="text-xs text-muted-foreground">Map presets to models in this list.</p>
          </div>
          <div className="grid gap-2 sm:grid-cols-3">
            {[
              { key: "FAST", label: "Fast" },
              { key: "CHEAP", label: "Cheap" },
              { key: "SMART", label: "Smart" },
            ].map((preset) => (
              <div key={`${runnerKey}-${preset.key}`} className="space-y-1">
                <Label>{preset.label}</Label>
                <select
                  value={presets[preset.key] ?? ""}
                  onChange={(event) => onUpdatePreset(runnerKey, preset.key, event.target.value)}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  <option value="">Unassigned</option>
                  {modelIds.map((modelId) => (
                    <option key={modelId} value={modelId}>
                      {modelId}
                    </option>
                  ))}
                </select>
              </div>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function AddRunnerCard({
  newRunnerKey,
  onNewRunnerKeyChange,
  onAddRunner,
  knownRunners,
}: {
  newRunnerKey: string;
  onNewRunnerKeyChange: (key: string) => void;
  onAddRunner: () => void;
  knownRunners: Record<string, unknown> | undefined;
}) {
  return (
    <Card className="border-dashed border-border bg-card/30">
      <CardContent className="space-y-3 py-5">
        <Label>Add Runner</Label>
        <div className="flex flex-wrap gap-2">
          <Input
            value={newRunnerKey}
            onChange={(event) => onNewRunnerKeyChange(event.target.value)}
            placeholder="runner-key"
            className="flex-1 min-w-[200px]"
          />
          <Button variant="outline" onClick={onAddRunner}>
            Add Runner
          </Button>
        </div>
        {knownRunners && Object.keys(knownRunners).length > 0 && (
          <p className="text-xs text-muted-foreground">
            Known runners: {Object.keys(knownRunners).join(", ")}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
