import { fireEvent } from "@testing-library/react";
import { renderWithProviders, screen, waitFor } from "../../test-utils";
import userEvent from "@testing-library/user-event";
import { vi } from "vitest";
import { StepWelcome } from "./StepWelcome";
import { evaluateProfile, fetchProfiles } from "../../api/profiles";
import { fetchTargets } from "../../api/host";
import type { EvaluateProfileResponse } from "@vrooli/proto-types/vrooli-onboarding/v1/profiles/profiles_pb";

vi.mock("../../api/host", () => ({ fetchHostFacts: vi.fn().mockRejectedValue(new Error("not needed")), fetchTargets: vi.fn().mockResolvedValue({ targets: [{ id: "remote-node", kind: "NODE_KIND_AGENT", available: true }] }) }));
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

    expect(await screen.findByTestId("profile-read-failed")).toHaveTextContent("Your saved answers are retained");
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

  it("keeps answers and delayed saves isolated when the selected target changes", async () => {
    const onProfileSessionChange = vi.fn();
    vi.mocked(evaluateProfile).mockImplementation(async (_profileId, answers, target) => ({
      profile: { id: "general-purpose", version: "1.0.0", titleKey: "profiles.generalPurpose.title", descriptionKey: "profiles.generalPurpose.description" },
      questions: [{ id: "workspace", type: "text", promptKey: "setup.label.prompt", required: false, visible: true, options: [] }],
      recommendations: [], scenarios: [], resources: [], issues: [], valid: true,
      digest: `${target}:${JSON.stringify(answers.workspace ?? "empty")}`,
    } as unknown as EvaluateProfileResponse));
    const view = renderWithProviders(<StepWelcome target="local" onAccept={vi.fn()} onAdjust={vi.fn()} onProfileSessionChange={onProfileSessionChange} />);

    const localInput = await screen.findByTestId("profile-question-workspace");
    fireEvent.change(localInput, { target: { value: "local-work" } });
    await waitFor(() => expect(vi.mocked(evaluateProfile)).toHaveBeenCalledWith("general-purpose", { workspace: "local-work" }, "local", {}));

    view.rerender(<StepWelcome target="remote-node" onAccept={vi.fn()} onAdjust={vi.fn()} onProfileSessionChange={onProfileSessionChange} />);
    const remoteInput = await screen.findByTestId("profile-question-workspace");
    expect(remoteInput).toHaveValue("");
    fireEvent.change(remoteInput, { target: { value: "remote-work" } });
    await waitFor(() => expect(vi.mocked(evaluateProfile)).toHaveBeenCalledWith("general-purpose", { workspace: "remote-work" }, "remote-node", {}));

    view.rerender(<StepWelcome target="local" onAccept={vi.fn()} onAdjust={vi.fn()} onProfileSessionChange={onProfileSessionChange} />);
    expect(await screen.findByTestId("profile-question-workspace")).toHaveValue("local-work");
    await waitFor(() => expect(onProfileSessionChange).toHaveBeenCalledWith(expect.objectContaining({ target: "local", answers: { workspace: "local-work" } })), { timeout: 1000 });
    expect(onProfileSessionChange.mock.calls.flatMap(([draft]) => draft.target === "remote-node" ? [draft.answers] : [])).not.toContainEqual({ workspace: "local-work" });
  });

  it("waits for durable target sessions before autosaving catalog defaults", async () => {
    const onProfileSessionChange = vi.fn();
    const view = renderWithProviders(<StepWelcome target="local" profileSession={null} onAccept={vi.fn()} onAdjust={vi.fn()} onProfileSessionChange={onProfileSessionChange} />);
    await screen.findByTestId("purpose-profile-select");
    await new Promise((resolve) => window.setTimeout(resolve, 320));
    expect(onProfileSessionChange).not.toHaveBeenCalled();

    const localSession = { target: "local", actor: "operator", mode: "guided" as const, profileId: "local-use", profileVersion: "1.0.0", answers: { purpose: "develop" }, manualDecisions: {}, targetContext: { host: "local" }, baseRevision: "local-r1", revision: "local-r2" };
    view.rerender(<StepWelcome target="local" profileSession={localSession} onAccept={vi.fn()} onAdjust={vi.fn()} onProfileSessionChange={onProfileSessionChange} />);
    await waitFor(() => expect(onProfileSessionChange).toHaveBeenCalledWith(expect.objectContaining({ target: "local", profileId: "local-use", answers: { purpose: "develop" } })));

    onProfileSessionChange.mockClear();
    view.rerender(<StepWelcome target="remote-node" profileSession={null} onAccept={vi.fn()} onAdjust={vi.fn()} onProfileSessionChange={onProfileSessionChange} />);
    await screen.findByTestId("purpose-profile-select");
    await new Promise((resolve) => window.setTimeout(resolve, 320));
    expect(onProfileSessionChange).not.toHaveBeenCalled();

    const remoteSession = { ...localSession, target: "remote-node", profileId: "general-purpose", answers: { purpose: "develop" }, targetContext: { host: "remote" }, baseRevision: "remote-r1", revision: "remote-r2" };
    view.rerender(<StepWelcome target="remote-node" profileSession={remoteSession} onAccept={vi.fn()} onAdjust={vi.fn()} onProfileSessionChange={onProfileSessionChange} />);
    await waitFor(() => expect(onProfileSessionChange).toHaveBeenCalledWith(expect.objectContaining({ target: "remote-node", targetContext: { host: "remote" }, answers: { purpose: "develop" } })));

    view.rerender(<StepWelcome target="local" profileSession={localSession} onAccept={vi.fn()} onAdjust={vi.fn()} onProfileSessionChange={onProfileSessionChange} />);
    await waitFor(() => expect(onProfileSessionChange).toHaveBeenLastCalledWith(expect.objectContaining({ target: "local", targetContext: { host: "local" }, profileId: "local-use", answers: { purpose: "develop" } })));
  });

  it("does not fetch or save a profile for an unsupported empty target", async () => {
    vi.mocked(fetchProfiles).mockClear();
    vi.mocked(evaluateProfile).mockClear();
    const onProfileSessionChange = vi.fn();
    renderWithProviders(<StepWelcome target="" onAccept={vi.fn()} onAdjust={vi.fn()} onProfileSessionChange={onProfileSessionChange} />);
    expect(await screen.findByTestId("unsupported-profile-target")).toHaveTextContent("Choose the local host or an available remote node");
    expect(fetchProfiles).not.toHaveBeenCalled();
    expect(evaluateProfile).not.toHaveBeenCalled();
    expect(onProfileSessionChange).not.toHaveBeenCalled();
  });

  it("does not read or save profile state for an unknown target ID", async () => {
    vi.mocked(fetchProfiles).mockClear();
    vi.mocked(evaluateProfile).mockClear();
    const onProfileSessionChange = vi.fn();
    renderWithProviders(<StepWelcome target="unregistered-host" onAccept={vi.fn()} onAdjust={vi.fn()} onProfileSessionChange={onProfileSessionChange} />);
    expect(await screen.findByTestId("unsupported-profile-target")).toHaveTextContent("not a registered host");
    expect(fetchProfiles).not.toHaveBeenCalled();
    expect(evaluateProfile).not.toHaveBeenCalled();
    expect(onProfileSessionChange).not.toHaveBeenCalled();
  });

  it("keeps a remote draft when registry admission disappears and restores it when the node returns", async () => {
    const supported = { targets: [{ id: "remote-node", kind: "NODE_KIND_AGENT", available: true }] };
    vi.mocked(fetchTargets).mockReset();
    vi.mocked(fetchTargets).mockResolvedValueOnce(supported).mockResolvedValueOnce(supported).mockResolvedValueOnce({ targets: [] }).mockResolvedValueOnce(supported);
    vi.mocked(fetchProfiles).mockClear();
    vi.mocked(evaluateProfile).mockResolvedValue({
      profile: { id: "general-purpose", version: "1.0.0", titleKey: "profiles.generalPurpose.title", descriptionKey: "profiles.generalPurpose.description" },
      questions: [{ id: "workspace", type: "text", promptKey: "setup.label.prompt", required: false, visible: true, options: [] }],
      recommendations: [], scenarios: [], resources: [], issues: [], valid: true,
    } as unknown as EvaluateProfileResponse);
    const onProfileSessionChange = vi.fn();
    const remoteSession = { target: "remote-node", actor: "operator", mode: "guided" as const, profileId: "general-purpose", profileVersion: "1.0.0", answers: { workspace: "saved-on-node" }, manualDecisions: {}, targetContext: { host: "remote" }, baseRevision: "remote-r1", revision: "remote-r2" };
    const view = renderWithProviders(<StepWelcome target="remote-node" profileSession={remoteSession} onAccept={vi.fn()} onAdjust={vi.fn()} onProfileSessionChange={onProfileSessionChange} />);
    expect(await screen.findByTestId("profile-question-workspace")).toHaveValue("saved-on-node");

    view.rerender(<StepWelcome target="local" onAccept={vi.fn()} onAdjust={vi.fn()} onProfileSessionChange={onProfileSessionChange} />);
    await screen.findByTestId("profile-question-workspace");
    vi.mocked(fetchProfiles).mockRejectedValueOnce(new Error("remote profile read failed"));
    onProfileSessionChange.mockClear();
    view.rerender(<StepWelcome target="remote-node" profileSession={remoteSession} onAccept={vi.fn()} onAdjust={vi.fn()} onProfileSessionChange={onProfileSessionChange} />);

    expect(await screen.findByTestId("unsupported-profile-target")).toHaveTextContent("not a registered host");
    expect(onProfileSessionChange).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Check target again" }));
    await waitFor(() => expect(screen.getByTestId("profile-question-workspace")).toBeInTheDocument());
    expect(screen.getByTestId("profile-question-workspace")).toHaveValue("saved-on-node");
  });

  it("retains a remote saved profile through a failed owner read while the node remains registered", async () => {
    const supported = { targets: [{ id: "remote-node", kind: "NODE_KIND_AGENT", available: true }] };
    vi.mocked(fetchTargets).mockReset();
    vi.mocked(fetchTargets).mockResolvedValueOnce(supported).mockResolvedValueOnce(supported);
    vi.mocked(fetchProfiles).mockRejectedValueOnce(new Error("remote profile read failed"));
    vi.mocked(evaluateProfile).mockResolvedValue({
      profile: { id: "general-purpose", version: "1.0.0", titleKey: "profiles.generalPurpose.title", descriptionKey: "profiles.generalPurpose.description" },
      questions: [{ id: "workspace", type: "text", promptKey: "setup.label.prompt", required: false, visible: true, options: [] }],
      recommendations: [], scenarios: [], resources: [], issues: [], valid: true,
    } as unknown as EvaluateProfileResponse);
    const onProfileSessionChange = vi.fn();
    const remoteSession = { target: "remote-node", actor: "operator", mode: "guided" as const, profileId: "general-purpose", profileVersion: "1.0.0", answers: { workspace: "saved-on-node" }, manualDecisions: {}, targetContext: { host: "remote" }, baseRevision: "remote-r1", revision: "remote-r2" };
    renderWithProviders(<StepWelcome target="remote-node" profileSession={remoteSession} onAccept={vi.fn()} onAdjust={vi.fn()} onProfileSessionChange={onProfileSessionChange} />);

    expect(await screen.findByTestId("profile-read-failed")).toHaveTextContent("Your saved answers are retained");
    expect(onProfileSessionChange).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Retry profile load" }));
    expect(await screen.findByTestId("profile-question-workspace")).toHaveValue("saved-on-node");
  });

  it("reports profile evaluation failures without hiding manual setup", async () => {
    vi.mocked(evaluateProfile).mockRejectedValueOnce(new Error("evaluation unavailable"));
    renderWithProviders(<StepWelcome onAccept={vi.fn()} onAdjust={vi.fn()} />);

    expect(await screen.findByRole("alert")).toHaveTextContent("This profile could not be evaluated");
    expect(screen.getByTestId("purpose-profile-manual")).toBeInTheDocument();
  });
});
