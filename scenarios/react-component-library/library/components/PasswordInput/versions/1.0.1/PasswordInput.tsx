/**
 * @libraryId react-component-library:PasswordInput
 * @displayName Password Input
 * @description Secret input with reveal and paste-safe handling
 * @version 1.0.1
 * @tags ["forms","secret","setup"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import * as React from "react";

export interface PasswordInputProps {
  value?: string;
  onChange?: (value: string) => void;
  label?: string;
  name?: string;
  autoComplete?: string;
  disabled?: boolean;
  className?: string;
}
export default function PasswordInput({
  value = "",
  onChange,
  label = "Password",
  name = "password",
  autoComplete = "current-password",
  disabled,
  className = "",
}: PasswordInputProps) {
  const [revealed, setRevealed] = React.useState(false);
  return (
    <label
      className={`rcl-component password-input ${className}`.trim()}
      style={{ display: "grid", gap: 4 }}
    >
      <span>{label}</span>
      <span style={{ display: "flex", gap: 4 }}>
        <input
          name={name}
          type={revealed ? "text" : "password"}
          autoComplete={autoComplete}
          value={value}
          disabled={disabled}
          onChange={(e) => onChange?.(e.target.value)}
        />
        <button
          type="button"
          aria-label={revealed ? "Hide password" : "Reveal password"}
          onClick={() => setRevealed((current) => !current)}
        >
          {revealed ? "Hide" : "Show"}
        </button>
      </span>
    </label>
  );
}
