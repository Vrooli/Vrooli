import { screen } from "@testing-library/react";
import { renderWithProviders } from "../test-utils";
import { describe, expect, it } from "vitest";

import { Field, TextInput } from "./primitives";

describe("field association", () => {
  const label = "Layer name";

  it("labels a control that has no id of its own", () => {
    renderWithProviders(
      <Field label={label}>
        <TextInput />
      </Field>,
    );
    expect(screen.getByLabelText(label)).toBeInstanceOf(HTMLInputElement);
  });

  it("keeps an id the control already declares", () => {
    renderWithProviders(
      <Field label={label}>
        <TextInput id="custom-id" />
      </Field>,
    );
    expect(screen.getByLabelText(label).id).toBe("custom-id");
  });
});
