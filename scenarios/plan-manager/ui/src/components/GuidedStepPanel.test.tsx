import { create } from "@bufbuild/protobuf";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { GuidedStepSchema, NextActionKind } from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";
import { strings } from "../consts/strings";
import { renderWithProviders } from "../test-utils";
import { GuidedStepPanel } from "./GuidedStepPanel";

describe("GuidedStepPanel", () => {
  it("renders the full guided action contract", () => {
    renderWithProviders(
      <GuidedStepPanel
        headingId="guided-step-heading"
        testId="guided-step"
        commandPrefix={["vrooli", "scenario", "plan-manager"]}
        step={create(GuidedStepSchema, {
          stepKind: "phase_context",
          title: "Phase Context",
          summary: "Use context before editing.",
          instructions: ["Run setup first."],
          requiredInputs: ["validation"],
          examples: ["plan-manager exec status e1"],
          commonMistakes: ["Skipping validation."],
          nextActions: [
            {
              id: "run-validation",
              kind: NextActionKind.RECOMMENDED,
              label: "Run validation",
              reason: "A recent pass is required.",
              argv: ["validate", "run", "plan-1", "--phase", "ph-1"],
              blockedBy: ["no stored validation result"],
            },
          ],
        })}
      />,
    );

    const panel = screen.getByTestId("guided-step");
    expect(panel).toHaveTextContent("Use context before editing.");
    expect(panel).toHaveTextContent("validation");
    expect(panel).toHaveTextContent(strings.guidedStep.recommended);
    expect(panel).toHaveTextContent("Run validation");
    expect(panel).toHaveTextContent("A recent pass is required.");
    expect(panel).toHaveTextContent("vrooli scenario plan-manager validate run plan-1 --phase ph-1");
    expect(panel).toHaveTextContent(`${strings.guidedStep.blocked}: no stored validation result`);
    expect(panel).toHaveTextContent(`${strings.guidedStep.example}: plan-manager exec status e1`);
    expect(panel).toHaveTextContent(`${strings.guidedStep.avoid}: Skipping validation.`);
  });
});
