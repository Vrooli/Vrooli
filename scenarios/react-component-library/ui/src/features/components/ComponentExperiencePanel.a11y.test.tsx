import { render } from "@testing-library/react";
import { describe, it } from "vitest";

import { expectNoA11yViolations } from "../../test-utils/a11y";
import { ComponentExperiencePanel } from "./ComponentExperiencePanel";

describe("ComponentExperiencePanel accessibility", () => {
  it("labels the contract summary and claim list", async () => {
    const { container } = render(
      <ComponentExperiencePanel
        isLoading={false}
        experience={{
          componentId: "button",
          libraryId: "react-component-library:Button",
          version: "1.2.0",
          contractId: "button",
          title: "Button",
          purpose: "Provide an accessible action.",
          evidenceStatus: "available",
          evidenceMessage: "",
          states: [{ id: "primary", exampleName: "primary", description: "Primary action." }],
          claims: [
            {
              id: "action-present",
              type: "element-present",
              statement: "A named action is present.",
              tier: "machine",
              states: ["primary"],
            },
          ],
          evidence: [],
        }}
      />,
    );
    await expectNoA11yViolations(container);
  });

  it("announces a live-evidence loading failure without presenting it as no contract", async () => {
    const { container, getByRole } = render(<ComponentExperiencePanel isLoading={false} isError />);
    expect(getByRole("alert")).toBeInTheDocument();
    await expectNoA11yViolations(container);
  });
});
