import { useState, useEffect } from "react";
import { Input } from "./ui/input";
import { Label } from "./ui/label";

interface ModelSelectorProps {
  value: string | undefined;
  onChange: (value: string) => void;
  models: string[];
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
  const actualValue = value ?? "";
  // Determine if current value is a custom one (not in the models list)
  const isCustomValue = actualValue !== "" && !models.includes(actualValue);
  const [isCustomMode, setIsCustomMode] = useState(isCustomValue);
  const [customValue, setCustomValue] = useState(isCustomValue ? actualValue : "");

  // Update custom mode when value changes externally (e.g., runner type switch)
  // Only reset custom mode if a known model is selected externally
  useEffect(() => {
    if (actualValue && models.includes(actualValue)) {
      // External change to a known model - exit custom mode
      setIsCustomMode(false);
      setCustomValue("");
    } else if (actualValue && !models.includes(actualValue)) {
      // External change to a custom value - enter custom mode
      setIsCustomMode(true);
      setCustomValue(actualValue);
    }
    // Don't change anything if value is empty - let user stay in custom mode
  }, [actualValue, models]);

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
      {label && <Label>{label}</Label>}
      <div className="flex gap-2">
        <select
          value={isCustomMode ? CUSTOM_OPTION : actualValue}
          onChange={handleSelectChange}
          className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          {models.length === 0 && !isCustomMode && (
            <option value="" disabled>
              Loading models...
            </option>
          )}
          {models.map((model) => (
            <option key={model} value={model}>
              {model}
            </option>
          ))}
          <option value={CUSTOM_OPTION}>Other (custom)...</option>
        </select>
      </div>
      {isCustomMode && (
        <Input
          value={customValue}
          onChange={handleCustomChange}
          placeholder={placeholder}
          className="mt-2"
        />
      )}
    </div>
  );
}
