import { useState } from "react";

import { Input } from "./input";
import { SegmentedControl } from "./segmented-control";

const KB = 1024;
const MB = 1024 * 1024;

type SizeUnit = "kb" | "mb";

export interface TargetSizeFieldProps {
  /** Already-translated label. */
  label: string;
  /** Target size in bytes; 0 means no limit. */
  valueBytes: number;
  onChange: (bytes: number) => void;
  /** Already-translated KB / MB unit labels. */
  kbLabel: string;
  mbLabel: string;
  /** Already-translated "no limit" helper text. */
  noLimitLabel: string;
  "data-testid"?: string;
}

/** Pick a friendly default unit + amount for an existing byte count. */
const deriveUnit = (bytes: number): SizeUnit => (bytes >= MB ? "mb" : "kb");
const toAmount = (bytes: number, unit: SizeUnit): number =>
  bytes <= 0 ? 0 : Math.round((bytes / (unit === "mb" ? MB : KB)) * 100) / 100;

/**
 * A numeric amount paired with a KB/MB unit toggle that emits a byte count on
 * the wire. A non-positive amount means "no size limit" (0 bytes) and shows a
 * helper. The chosen unit is local UI state; the byte value is the contract.
 */
export function TargetSizeField({
  label,
  valueBytes,
  onChange,
  kbLabel,
  mbLabel,
  noLimitLabel,
  ...rest
}: TargetSizeFieldProps) {
  const [unit, setUnit] = useState<SizeUnit>(() => deriveUnit(valueBytes));
  const amount = toAmount(valueBytes, unit);

  const emit = (nextAmount: number, nextUnit: SizeUnit) => {
    if (!Number.isFinite(nextAmount) || nextAmount <= 0) {
      onChange(0);
      return;
    }
    onChange(Math.round(nextAmount * (nextUnit === "mb" ? MB : KB)));
  };

  return (
    <div className="flex flex-col gap-1">
      <span className="text-xs text-app-muted-foreground">{label}</span>
      <div className="flex items-center gap-2">
        <Input
          type="number"
          min={0}
          aria-label={label}
          data-testid={rest["data-testid"]}
          value={amount === 0 ? "" : String(amount)}
          onChange={(e) => emit(Number(e.target.value), unit)}
          className="w-24"
        />
        <SegmentedControl<SizeUnit>
          label={label}
          value={unit}
          options={[
            { value: "kb", label: kbLabel },
            { value: "mb", label: mbLabel },
          ]}
          onChange={(nextUnit) => {
            setUnit(nextUnit);
            emit(amount, nextUnit);
          }}
        />
      </div>
      {valueBytes <= 0 ? (
        <span className="text-xs text-app-muted-foreground">{noLimitLabel}</span>
      ) : null}
    </div>
  );
}
