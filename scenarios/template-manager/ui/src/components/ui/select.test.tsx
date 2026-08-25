import { describe, expect, it } from "vitest";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { Select } from "@vrooli/react-component-library/Select/1.1.0";

describe("Select", () => {
  it("renders options and forwards native selection", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(
      <Select
        aria-label="Choice"
        options={[
          { value: "a", label: "Alpha" },
          { value: "b", label: "Beta" },
        ]}
      />,
    );

    const select = container.querySelector<HTMLSelectElement>("select");
    expect(select).not.toBeNull();
    await user.selectOptions(select as HTMLSelectElement, "b");

    expect(select?.value).toBe("b");
  });
});
