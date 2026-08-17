/* eslint-disable no-restricted-syntax -- synthetic select copy is a component-contract fixture. */
import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { Select } from "./select";

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
});
