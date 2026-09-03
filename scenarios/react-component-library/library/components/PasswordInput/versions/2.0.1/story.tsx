// Preview contract specimens for PasswordInput 2.0.0.
// Each export is one claim the declarative contract then asserts against.
import { useState } from "react";
import { PasswordInput } from "./PasswordInput";

/** The ordinary case: a labelled secret with a reveal control inside the field. */
export function Default() {
  const [value, setValue] = useState("hunter2-correct-horse");
  return (
    <div style={{ inlineSize: "22rem", maxInlineSize: "100%" }}>
      <PasswordInput
        testId="story-password"
        label="Password"
        description="Used once to install the agent. Not stored."
        value={value}
        onValueChange={setValue}
        autoComplete="current-password"
      />
    </div>
  );
}

/**
 * A value the operator may replace but must never read back. The reveal
 * control is absent rather than disabled, because the field is still writable.
 */
export function WriteOnly() {
  const [value, setValue] = useState("");
  return (
    <div style={{ inlineSize: "22rem", maxInlineSize: "100%" }}>
      <PasswordInput
        testId="story-sealed"
        label="Registry token"
        description="Pushed sealed to the node. It is never displayed here."
        revealable={false}
        value={value}
        onValueChange={setValue}
        autoComplete="new-password"
      />
    </div>
  );
}

/** An error recolours the shared border and is announced by FormField. */
export function Invalid() {
  return (
    <div style={{ inlineSize: "22rem", maxInlineSize: "100%" }}>
      <PasswordInput
        testId="story-invalid"
        label="Password"
        required
        error="This machine rejected the password."
        defaultValue="wrong"
      />
    </div>
  );
}

/** No label means the bare control, for a caller supplying its own FormField. */
export function Bare() {
  return (
    <div style={{ inlineSize: "22rem", maxInlineSize: "100%" }}>
      <PasswordInput testId="story-bare" defaultValue="secret" />
    </div>
  );
}
