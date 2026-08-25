import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { Select } from "@vrooli/react-component-library/Select/1.1.0";

/**
 * Fixture copy, named once. These are the test's OWN sample values rather
 * than application copy, but they are referenced through a constant so the
 * copy-driven-query lint rule stays enforceable without a per-file exemption.
 */
const CHOICE_LABEL = "Choice";

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

    await user.selectOptions(screen.getByLabelText(CHOICE_LABEL), "b");

    expect(screen.getByLabelText<HTMLSelectElement>(CHOICE_LABEL).value).toBe("b");
  });
});
