// [REQ:REQ-P0-003] Welcome Step Component
import { cleanup, renderWithProviders, screen } from "../../test-utils";
import { afterEach, beforeEach, vi } from "vitest";
// provider-free-exception: StepWelcome is static wizard content with no provider dependency.
import { StepWelcome } from "./StepWelcome";

const hostApi = vi.hoisted(() => ({ fetchHostFacts: vi.fn() }));
vi.mock("../../api/host", () => hostApi);

beforeEach(() => {
  hostApi.fetchHostFacts.mockResolvedValue({ available: false });
});

afterEach(cleanup);

describe("StepWelcome", () => {
  it("renders welcome heading", () => {
    renderWithProviders(<StepWelcome />);
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("This machine is about to become a Vrooli node");
  });

  it("renders step-welcome test id", () => {
    renderWithProviders(<StepWelcome />);
    expect(screen.getByTestId("step-welcome")).toBeInTheDocument();
  });

  it("shows description text", () => {
    renderWithProviders(<StepWelcome />);
    expect(screen.getByText(/nothing is written until you approve it/i)).toBeInTheDocument();
  });

  it("states the three setup commitments", () => {
    renderWithProviders(<StepWelcome />);
    expect(screen.getByText("Installs local services")).toBeInTheDocument();
    expect(screen.getByText("Asks before touching the host")).toBeInTheDocument();
    expect(screen.getByText("Stays reversible")).toBeInTheDocument();
  });

  it("keeps decorative icons out of the accessibility tree", () => {
    renderWithProviders(<StepWelcome />);
    const container = screen.getByTestId("step-welcome");
    const svg = container.querySelector("svg");
    expect(svg).toHaveAttribute("aria-hidden", "true");
  });

  it("uses the semantic welcome lede treatment", () => {
    renderWithProviders(<StepWelcome />);
    const description = screen.getByText(/nothing is written until you approve it/i);
    expect(description.className).toContain("welcome-screen__lede");
  });

  it("renders available host facts with safe fallbacks", async () => {
    hostApi.fetchHostFacts.mockResolvedValueOnce({
      available: true,
      memory_total_bytes: 1024,
      disk_free_bytes: 1024 * 1024 * 12.5,
      gpus: [],
    });
    renderWithProviders(<StepWelcome />);

    const facts = await screen.findByTestId("host-facts");
    expect(facts).toHaveTextContent("1.0 KiB");
    expect(facts).toHaveTextContent("13 MiB");
    expect(facts).toHaveTextContent("None detected");
    expect(facts).toHaveTextContent("Unknown");
  });

  it("renders host facts returned with a platform and GPU", async () => {
    hostApi.fetchHostFacts.mockResolvedValueOnce({
      available: true,
      memory_total_bytes: 16 * 1024 * 1024 * 1024,
      disk_free_bytes: 0,
      gpus: ["NVIDIA RTX"],
      platform: "linux",
    });
    renderWithProviders(<StepWelcome />);

    const facts = await screen.findByTestId("host-facts");
    expect(facts).toHaveTextContent("16 GiB");
    expect(facts).toHaveTextContent("—");
    expect(facts).toHaveTextContent("NVIDIA RTX");
    expect(facts).toHaveTextContent("linux");
  });
});
