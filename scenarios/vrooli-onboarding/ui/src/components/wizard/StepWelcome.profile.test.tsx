import { renderWithProviders, screen, waitFor } from "../../test-utils";
import userEvent from "@testing-library/user-event";
import { vi } from "vitest";
import { StepWelcome } from "./StepWelcome";

vi.mock("../../api/host", () => ({ fetchHostFacts: vi.fn().mockRejectedValue(new Error("not needed")) }));
vi.mock("../../api/profiles", () => ({
  fetchProfiles: vi.fn().mockResolvedValue({ profiles: [{ id: "local-use", version: "1.0.0", titleKey: "profiles.localUse.title", descriptionKey: "profiles.localUse.description" }] }),
  evaluateProfile: vi.fn().mockResolvedValue({
    profile: { id: "local-use", version: "1.0.0", titleKey: "profiles.localUse.title", descriptionKey: "profiles.localUse.description" },
    questions: [{ id: "purpose", type: "single-select", promptKey: "setup.purpose.prompt", required: true, visible: true, options: [{ id: "develop", labelKey: "setup.purpose.develop" }] }],
    recommendations: [{ capabilityRef: "development.local", scenarioRefs: ["vrooli-onboarding"], reasonKey: "setup.reason.localDevelopment" }],
    scenarios: ["vrooli-onboarding"],
    resources: [],
    issues: [],
    valid: true,
  }),
}));
vi.mock("../../api/selection", () => ({
  fetchRecommendation: vi.fn().mockResolvedValue({ profile: "starter", scenarios: [], resources: [], explanation: "starter" }),
  fetchClosure: vi.fn().mockResolvedValue({ resources: [] }),
}));

describe("StepWelcome purpose profiles", () => {
  it("evaluates a profile and commits its recommended scenarios", async () => {
    const user = userEvent.setup();
    const onAccept = vi.fn().mockResolvedValue(undefined);
    renderWithProviders(<StepWelcome onAccept={onAccept} onAdjust={vi.fn()} />);

    await waitFor(() => expect(screen.getByText("Recommended capabilities: vrooli-onboarding")).toBeInTheDocument());
    expect(screen.getByTestId("purpose-profile-select")).toHaveAttribute("data-rcl-select", "true");
    expect(screen.getByTestId("profile-question-purpose")).toHaveAttribute("data-rcl-select", "true");
    await user.click(screen.getByRole("button", { name: "Use this profile" }));
    await waitFor(() => expect(onAccept).toHaveBeenCalledWith("local-use", ["vrooli-onboarding"]));
  });
});
