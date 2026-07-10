import { useId } from "react";
import type { ModelOption } from "../types";
import type { PolicyOption } from "../lib/modelPolicyCatalog";
import { Label } from "./ui/label";
import { ModelSelector } from "./ModelSelector";

export type ModelSelectionMode = "default" | "policy" | "model";

export interface ModelSelectionState {
  mode: ModelSelectionMode;
  model: string;
  policyRef: string;
}

interface ModelConfigSelectorProps {
  value: ModelSelectionState;
  onChange: (value: ModelSelectionState) => void;
  models: ModelOption[];
  policies?: PolicyOption[];
  label?: string;
}

export function ModelConfigSelector({
  value,
  onChange,
  models,
  policies = [],
  label = "Model Selection",
}: ModelConfigSelectorProps) {
  const groupId = useId();
  const selectedPolicy = policies.find((policy) => policy.ref === value.policyRef);

  return (
    <div className="space-y-3">
      <Label>{label}</Label>
      <div className="grid gap-2">
        <label className="flex items-center gap-2 text-sm">
          <input
            type="radio"
            name={groupId}
            checked={value.mode === "default"}
            onChange={() =>
              onChange({
                mode: "default",
                model: "",
                policyRef: "",
              })
            }
          />
          Use runner default
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="radio"
            name={groupId}
            checked={value.mode === "policy"}
            onChange={() =>
              onChange({
                ...value,
                mode: "policy",
                model: "",
                policyRef: value.policyRef || policies[0]?.ref || "",
              })
            }
          />
          Named policy
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="radio"
            name={groupId}
            checked={value.mode === "model"}
            onChange={() =>
              onChange({
                ...value,
                mode: "model",
                policyRef: "",
              })
            }
          />
          Specific model
        </label>
      </div>

      {value.mode === "policy" && (
        <div className="space-y-2">
          <Label htmlFor="policyRef">Policy</Label>
          <select
            id="policyRef"
            value={value.policyRef}
            onChange={(event) =>
              onChange({
                ...value,
                policyRef: event.target.value,
              })
            }
            className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            {policies.map((option) => (
              <option key={option.ref} value={option.ref}>
                {option.label} ({option.ref})
              </option>
            ))}
          </select>
          {selectedPolicy?.primaryModel && (
            <p className="text-xs text-muted-foreground">
              Primary candidate: <span className="font-medium text-foreground">{selectedPolicy.primaryModel}</span>.
            </p>
          )}
        </div>
      )}

      {value.mode === "model" && (
        <ModelSelector
          value={value.model}
          onChange={(model) =>
            onChange({
              ...value,
              model,
            })
          }
          models={models}
          label="Model"
          placeholder="Enter custom model..."
        />
      )}
    </div>
  );
}
