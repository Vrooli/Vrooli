/**
 * @libraryId react-component-library:PinInput
 * @displayName PIN Input
 * @description Grouped numeric code input with paste support
 * @version 1.0.1
 * @tags ["forms","secret","setup"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import * as React from "react";

export interface PinInputProps {
  length?: number;
  value?: string;
  onChange?: (value: string) => void;
  label?: string;
  className?: string;
}
export default function PINInput({
  length = 6,
  value = "",
  onChange,
  label = "PIN",
  className = "",
}: PinInputProps) {
  const chars = Array.from({ length }, (_, i) => value[i] ?? "");
  const update = (index: number, next: string) => {
    const digit = next.replace(/\D/g, "").slice(-1);
    const result = chars.map((char, i) => (i === index ? digit : char)).join("");
    onChange?.(result);
  };
  return (
    <fieldset className={`rcl-component pin-input ${className}`.trim()}>
      <legend>{label}</legend>
      <div
        role="group"
        aria-label={label}
        onPaste={(e) => {
          e.preventDefault();
          onChange?.(e.clipboardData.getData("text").replace(/\D/g, "").slice(0, length));
        }}
      >
        {chars.map((char, i) => (
          <input
            key={i}
            aria-label={`${label} digit ${i + 1}`}
            inputMode="numeric"
            maxLength={1}
            value={char}
            onChange={(e) => update(i, e.target.value)}
          />
        ))}
      </div>
    </fieldset>
  );
}
