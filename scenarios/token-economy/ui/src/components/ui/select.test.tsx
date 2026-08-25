import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { Select } from "@vrooli/react-component-library/Select/1.1.0";

const choiceLabel = "Choice";

describe("Select", () => {
  it("renders options and forwards native selection", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <Select
        aria-label={choiceLabel}
        options={[
          { value: "a", label: "Alpha" },
          { value: "b", label: "Beta" },
        ]}
      />,
    );

    await user.selectOptions(screen.getByLabelText(choiceLabel), "b");

    expect(screen.getByLabelText<HTMLSelectElement>(choiceLabel).value).toBe("b");
  });
});
