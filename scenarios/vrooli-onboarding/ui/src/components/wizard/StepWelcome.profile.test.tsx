import { fireEvent } from "@testing-library/react";
import { renderWithProviders, screen, waitFor } from "../../test-utils";
import userEvent from "@testing-library/user-event";
import { vi } from "vitest";
import { StepWelcome } from "./StepWelcome";
import { evaluateProfile, fetchProfiles } from "../../api/profiles";
import type { EvaluateProfileResponse } from "@vrooli/proto-types/vrooli-onboarding/v1/profiles/profiles_pb";

vi.mock("../../api/host", () => ({ fetchHostFacts: vi.fn().mockRejectedValue(new Error("not needed")) }));
vi.mock("../../api/profiles", () => ({
  fetchProfiles: vi.fn().mockResolvedValue({ profiles: [
    { id: "local-use", version: "1.0.0", titleKey: "profiles.localUse.title", descriptionKey: "profiles.localUse.description" },
    { id: "general-purpose", version: "1.0.0", default: true, titleKey: "profiles.generalPurpose.title", descriptionKey: "profiles.generalPurpose.description" },
  ] }),
  evaluateProfile: vi.fn().mockResolvedValue({
    profile: { id: "general-purpose", version: "1.0.0", default: true, titleKey: "profiles.generalPurpose.title", descriptionKey: "profiles.generalPurpose.description" },
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
    expect(screen.getByTestId("profile-recommendation-summary")).toBeInTheDocument();
    expect(screen.queryByTestId("plan-summary")).not.toBeInTheDocument();
    expect(screen.getByText("Local development tools support this purpose.")).toBeInTheDocument();
    expect(screen.getByTestId("purpose-profile-select")).toHaveAttribute("data-rcl-select", "true");
    expect(screen.getByTestId("profile-question-purpose")).toHaveAttribute("data-rcl-select", "true");
    await user.click(screen.getByRole("button", { name: "Use this profile" }));
    await waitFor(() => expect(onAccept).toHaveBeenCalledWith("general-purpose", ["vrooli-onboarding"]));
  });

  it("keeps manual setup available when the profile catalog cannot be loaded", async () => {
    vi.mocked(fetchProfiles).mockRejectedValueOnce(new Error("catalog unavailable"));
    const onAdjust = vi.fn();
    renderWithProviders(<StepWelcome onAccept={vi.fn()} onAdjust={onAdjust} />);

    expect(await screen.findByRole("alert")).toHaveTextContent("Purpose profiles could not be loaded");
    expect((await screen.findAllByText("starter")).length).toBeGreaterThan(0);
  });

  it("retains reconciliation warnings and exposes a retry action", async () => {
    const retry = vi.fn();
    renderWithProviders(<StepWelcome
      onAccept={vi.fn()}
      onAdjust={vi.fn()}
      profileSession={{ target: "local", actor: "test", mode: "guided", profileId: "general-purpose", profileVersion: "0.9.0", answers: {}, manualDecisions: {}, targetContext: {}, baseRevision: "base-r1", revision: "r1", reconciliationState: "review_required", reconciliationReasons: ["The profile changed."] }}
      profileSessionSaveState="conflict"
      profileSessionError="Your answers are retained."
      onRetryProfileSessionSave={retry}
    />);

    expect(await screen.findByTestId("profile-session-reconciliation")).toHaveTextContent("The profile changed.");
    expect(screen.getByTestId("profile-session-status")).toHaveTextContent("Your answers are retained.");
    screen.getByRole("button", { name: "Retry save" }).click();
    expect(retry).toHaveBeenCalled();
  });

  it("supports manual mode and the remaining profile question types", async () => {
    const onAdjust = vi.fn();
    const onProfileSessionChange = vi.fn();
    vi.mocked(evaluateProfile).mockResolvedValueOnce({
      profile: { id: "general-purpose", version: "1.0.0", titleKey: "profiles.generalPurpose.title", descriptionKey: "profiles.generalPurpose.description", provenanceSource: "fixture", provenanceRevision: "r2" },
      questions: [
        { id: "purposes", type: "multi-select", promptKey: "setup.purposes.prompt", required: true, visible: true, options: [{ id: "develop", labelKey: "setup.purpose.develop" }] },
        { id: "remote", type: "boolean", promptKey: "setup.remote.prompt", required: false, visible: true, options: [] },
        { id: "label", type: "text", promptKey: "setup.label.prompt", required: false, visible: true, options: [] },
      ],
      recommendations: [{ capabilityRef: "fixture.capability", scenarioRefs: [], reasonKey: "setup.reason.localDevelopment" }],
      scenarios: [], resources: ["postgres"], issues: [], valid: true,
    } as unknown as EvaluateProfileResponse);
    renderWithProviders(<StepWelcome onAccept={vi.fn()} onAdjust={onAdjust} onProfileSessionChange={onProfileSessionChange} />);

    expect(await screen.findByText("fixture.capability")).toBeInTheDocument();
    screen.getByTestId("profile-question-purposes-develop").click();
    screen.getByTestId("profile-question-remote").click();
    const textboxes = screen.getAllByRole("textbox");
    const labelInput = textboxes[textboxes.length - 1];
    expect(labelInput).toBeDefined();
    fireEvent.change(labelInput!, { target: { value: "workspace" } });
    screen.getByTestId("purpose-profile-manual").click();
    expect(onAdjust).toHaveBeenCalled();
    expect(onProfileSessionChange).toHaveBeenCalledWith(expect.objectContaining({ mode: "manual" }));
  });

  it("reports profile evaluation failures without hiding manual setup", async () => {
    vi.mocked(evaluateProfile).mockRejectedValueOnce(new Error("evaluation unavailable"));
    renderWithProviders(<StepWelcome onAccept={vi.fn()} onAdjust={vi.fn()} />);

    expect(await screen.findByRole("alert")).toHaveTextContent("This profile could not be evaluated");
    expect(screen.getByTestId("purpose-profile-manual")).toBeInTheDocument();
  });
});
