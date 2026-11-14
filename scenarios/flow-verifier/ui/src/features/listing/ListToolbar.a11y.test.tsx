import { afterEach, describe, it } from "vitest";
import { cleanup } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { ListToolbar } from "./ListToolbar";

describe("ListToolbar accessibility", () => {
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(
      <ListToolbar
        testId="probe"
        searchValue=""
        onSearchChange={() => {}}
        sort={{
          options: [{ value: "name", label: "Name" }],
          value: { key: "name", dir: "asc" },
          onChange: () => {},
          testIdPrefix: "probe",
        }}
      />,
    );
    await expectNoA11yViolations(container);
  });
});
