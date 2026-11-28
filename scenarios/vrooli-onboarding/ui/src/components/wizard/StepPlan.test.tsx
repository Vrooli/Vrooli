import userEvent from "@testing-library/user-event";
import { renderWithProviders, screen } from "../../test-utils";
import { vi } from "vitest";
import { StepPlan } from "./StepPlan";
import { fetchClosure, fetchRecommendation } from "../../api/selection";
import type { GetClosureResponse, GetRecommendationResponse } from "@vrooli/proto-types/vrooli-onboarding/v1/selection/selection_pb";

vi.mock("../../api/selection", () => ({
  fetchRecommendation: vi.fn(),
  fetchClosure: vi.fn(),
}));

describe("StepPlan", () => {
  beforeEach(() => {
    vi.mocked(fetchRecommendation).mockResolvedValue({ profile: "starter", scenarios: ["alpha"], resources: ["redis"], explanation: "A compact setup" } as unknown as GetRecommendationResponse);
    vi.mocked(fetchClosure).mockResolvedValue({ resources: [{ name: "postgres" }] } as unknown as GetClosureResponse);
  });

  it("shows an error when either selection response is malformed", async () => {
    vi.mocked(fetchRecommendation).mockResolvedValueOnce({ profile: "starter", scenarios: undefined, resources: [], explanation: "" } as unknown as GetRecommendationResponse);
    renderWithProviders(<StepPlan onAccept={vi.fn()} onAdjust={vi.fn()} />);

    expect(await screen.findByRole("alert")).toHaveTextContent("recommended setup is not available yet");
  });

  it("shows an error when selection loading fails", async () => {
    vi.mocked(fetchClosure).mockRejectedValueOnce(new Error("closure unavailable"));
    renderWithProviders(<StepPlan onAccept={vi.fn()} onAdjust={vi.fn()} />);

    expect(await screen.findByRole("alert")).toHaveTextContent("recommended setup could not be loaded");
  });

  it("renders without closure resources when the closure response is incomplete", async () => {
    vi.mocked(fetchClosure).mockResolvedValueOnce({ resources: undefined } as unknown as GetClosureResponse);
    renderWithProviders(<StepPlan onAccept={vi.fn()} onAdjust={vi.fn()} />);

    expect(await screen.findByTestId("plan-summary")).toBeInTheDocument();
    expect(screen.getByText("alpha")).toBeInTheDocument();
    expect(screen.queryByText("postgres")).not.toBeInTheDocument();
  });

  it("surfaces a failed apply without losing the plan", async () => {
    const onAccept = vi.fn().mockRejectedValueOnce(new Error("apply failed"));
    const user = userEvent.setup();
    renderWithProviders(<StepPlan onAccept={onAccept} onAdjust={vi.fn()} />);

    await user.click(await screen.findByRole("button", { name: "Use recommended setup" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("recommendation could not be accepted");
    expect(screen.getByTestId("plan-summary")).toBeInTheDocument();
  });

  it("does not show the previous target recommendation while the next target loads", async () => {
    let finishRemote!: (value: GetRecommendationResponse) => void;
    vi.mocked(fetchRecommendation).mockImplementation((target) => target === "node-1"
      ? new Promise<GetRecommendationResponse>((resolve) => { finishRemote = resolve; })
      : Promise.resolve({ profile: "local profile", scenarios: ["local-only"], resources: [], explanation: "local plan" } as unknown as GetRecommendationResponse));
    const view = renderWithProviders(<StepPlan target="local" onAccept={vi.fn()} onAdjust={vi.fn()} />);
    expect(await screen.findByText("local-only")).toBeInTheDocument();

    view.rerender(<StepPlan target="node-1" onAccept={vi.fn()} onAdjust={vi.fn()} />);
    expect(screen.getByTestId("plan-loading")).toBeInTheDocument();
    expect(screen.queryByText("local-only")).not.toBeInTheDocument();

    finishRemote({ profile: "remote profile", scenarios: ["remote-only"], resources: [], explanation: "remote plan" } as unknown as GetRecommendationResponse);
    expect(await screen.findByText("remote-only")).toBeInTheDocument();
  });
});
