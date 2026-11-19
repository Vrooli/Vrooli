import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Field, TextInput } from "./primitives";

describe("field association", () => {
  const label = "Layer name";

  it("labels a control that has no id of its own", () => {
    render(
      <Field label={label}>
        <TextInput />
      </Field>,
    );
    expect(screen.getByLabelText(label)).toBeInstanceOf(HTMLInputElement);
  });

  it("keeps an id the control already declares", () => {
    render(
      <Field label={label}>
        <TextInput id="custom-id" />
      </Field>,
    );
    expect(screen.getByLabelText(label).id).toBe("custom-id");
  });
});
