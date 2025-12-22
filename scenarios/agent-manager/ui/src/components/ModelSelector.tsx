import { useState, useEffect, useId } from "react";
import type { ModelOption } from "../types";
import { Input } from "./ui/input";
import { Label } from "./ui/label";

interface ModelSelectorProps {
  value: string | undefined;
  onChange: (value: string) => void;
  models: ModelOption[];
  label?: string;
  placeholder?: string;
}

const CUSTOM_OPTION = "__custom__";

export function ModelSelector({
  value,
  onChange,
  models,
  label = "Model",
  placeholder = "Enter custom model...",
}: ModelSelectorProps) {
  const selectId = useId();
  const inputId = useId();
  const labelText = label || "Model";
  const actualValue = value ?? "";
  const normalizedModels = models.map((model) =>
    typeof model === "string" ? { id: model, description: "" } : model
  );
  const modelIds = normalizedModels.map((model) => model.id);
  // Determine if current value is a custom one (not in the models list)
  const isCustomValue = actualValue !== "" && !modelIds.includes(actualValue);
  const [isCustomMode, setIsCustomMode] = useState(isCustomValue);
  const [customValue, setCustomValue] = useState(isCustomValue ? actualValue : "");

  // Update custom mode when value changes externally (e.g., runner type switch)
  // Only reset custom mode if a known model is selected externally
  useEffect(() => {
    if (actualValue && modelIds.includes(actualValue)) {
      // External change to a known model - exit custom mode
      setIsCustomMode(false);
      setCustomValue("");
    } else if (actualValue && !modelIds.includes(actualValue)) {
      // External change to a custom value - enter custom mode
      setIsCustomMode(true);
      setCustomValue(actualValue);
    }
    // Don't change anything if value is empty - let user stay in custom mode
  }, [actualValue, modelIds]);

  const handleSelectChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const selected = e.target.value;
    if (selected === CUSTOM_OPTION) {
      setIsCustomMode(true);
      setCustomValue("");
      // Don't call onChange yet - wait for user to type
    } else {
      setIsCustomMode(false);
      setCustomValue("");
      onChange(selected);
    }
  };

  const handleCustomChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    setCustomValue(newValue);
    onChange(newValue);
  };

  return (
    <div className="space-y-2">
      {labelText && <Label htmlFor={selectId}>{labelText}</Label>}
      <div className="flex gap-2">
        <select
          id={selectId}
          value={isCustomMode ? CUSTOM_OPTION : actualValue}
          onChange={handleSelectChange}
          aria-label={labelText}
          className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          {normalizedModels.length === 0 && !isCustomMode && (
            <option value="" disabled>
              Loading models...
            </option>
          )}
          {normalizedModels.map((model) => (
            <option key={model.id} value={model.id}>
              {model.description ? `${model.id} — ${model.description}` : model.id}
            </option>
          ))}
          <option value={CUSTOM_OPTION}>Other (custom)...</option>
        </select>
      </div>
      {isCustomMode && (
        <Input
          id={inputId}
          value={customValue}
          onChange={handleCustomChange}
          placeholder={placeholder}
          aria-label={placeholder ?? "Custom model"}
          className="mt-2"
        />
      )}
    </div>
  );
}
