import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { PhaseInternalsDisclosure } from "./phase-internals-disclosure";
import { selectors } from "../../../consts/selectors";
import type { OperatingModeCatalogPhase } from "../../../types/operating-mode";
import { createQueryWrapper } from "../../../test-utils/query";

vi.mock("../../../services/prompt-service", () => ({
  promptService: {
    getSkill: vi.fn(),
  },
}));

import { promptService } from "../../../services/prompt-service";

function basePhase(overrides: Partial<OperatingModeCatalogPhase> & { phase: string }): OperatingModeCatalogPhase {
  return {
    label: overrides.phase,
    phaseKind: "investigate",
    title: overrides.phase,
    purpose: "",
    trigger: "",
    profileKey: "swarm-manager/deep-work",
    writesRepo: false,
    catalogId: "holistic-loop-investigate",
    skillId: "swarm-manager/holistic-loop-investigate",
    activityPurpose: "",
    lockPurpose: "",
    outputContract: {
      requiresStructuredResult: true,
      requiresProgress: false,
      requiresVerdict: false,
      requiresHandoff: false,
      requiresBacklogSync: false,
      requiredArtifactCount: 0,
    },
    ...overrides,
  };
}

describe("PhaseInternalsDisclosure", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders the skill ID as a clickable button that opens the SkillViewerDialog", async () => {
    vi.mocked(promptService.getSkill).mockResolvedValue({
      id: "swarm-manager/holistic-loop-investigate",
      name: "Holistic Loop Investigate",
      description: "Surveys the initiative.",
      draft: false,
      usage_type: "direct_runtime",
      groups: ["initiative"],
      trigger_count: 1,
      impact_summary: "",
      current_content: "Body",
    });
    render(
      <PhaseInternalsDisclosure
        phase={basePhase({ phase: "investigate" })}
        defaultOpen
      />,
      { wrapper: createQueryWrapper() },
    );

    expect(screen.queryByTestId(selectors.initiativeDetails.skillViewerDialog)).toBeNull();

    const skillButton = screen.getByTestId(selectors.initiativeDetails.phaseSkillIdButton);
    expect(skillButton.tagName).toBe("BUTTON");
    expect(skillButton).toHaveTextContent("swarm-manager/holistic-loop-investigate");
    await userEvent.click(skillButton);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.initiativeDetails.skillViewerDialog)).toBeInTheDocument();
    });
    expect(promptService.getSkill).toHaveBeenCalledWith("swarm-manager/holistic-loop-investigate");
  });

  it("does not render the skill button when no skill ID is set", () => {
    render(
      <PhaseInternalsDisclosure
        phase={basePhase({ phase: "investigate", skillId: "" })}
        defaultOpen
      />,
      { wrapper: createQueryWrapper() },
    );
    expect(screen.queryByTestId(selectors.initiativeDetails.phaseSkillIdButton)).toBeNull();
  });
});
