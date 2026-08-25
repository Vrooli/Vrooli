/* eslint-disable no-restricted-syntax */
import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { Select } from "@vrooli/react-component-library/Select/1.1.0";

describe("Select", () => {
  it("renders options and forwards native selection", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <Select
        aria-label="Choice"
        options={[
          { value: "a", label: "Alpha" },
          { value: "b", label: "Beta" },
        ]}
      />,
    );

    await user.selectOptions(screen.getByLabelText("Choice"), "b");

    expect(screen.getByLabelText<HTMLSelectElement>("Choice").value).toBe("b");
  });

  it("renders placeholders and disabled options", () => {
    renderWithProviders(
      <Select aria-label="Choice" placeholder="Choose one" options={[{ value: "a", label: "Alpha", disabled: true }]} />,
    );
    expect(screen.getByRole("option", { name: "Choose one" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Alpha" })).toBeDisabled();
  });
});
