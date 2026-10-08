import "@testing-library/jest-dom";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";

const fetchHealth = vi.hoisted(() => vi.fn());
vi.mock("./lib/api", () => ({ fetchHealth }));

import { Dashboard } from "./components/Dashboard";
import { renderWithProviders } from "./test-utils/renderWithProviders";

const savedKey = "scenario-to-cloud:deployment";

function saveDeployment(manifestJson: string) {
  localStorage.setItem(savedKey, JSON.stringify({
    manifestJson,
    currentStep: 2,
    timestamp: Date.now(),
  }));
}

describe("dashboard state surfaces", () => {
  beforeEach(() => {
    localStorage.clear();
    fetchHealth.mockReset();
  });

  it("renders the loading and healthy API states with an empty resume card", async () => {
    let resolveHealth!: (value: unknown) => void;
    fetchHealth.mockImplementation(() => new Promise((resolve) => { resolveHealth = resolve; }));
    renderWithProviders(<Dashboard onStartNew={vi.fn()} onResume={vi.fn()} />);
    expect(screen.getByText("Checking API status...")).toBeInTheDocument();
    resolveHealth({ service: "cloud-api", timestamp: "2026-01-01T00:00:00Z" });
    expect(await screen.findByText("API Online")).toBeInTheDocument();
    expect(screen.getByText("No Saved Progress")).toBeInTheDocument();
  });

  it("resumes and discards a valid saved deployment", async () => {
    saveDeployment(JSON.stringify({ scenario: { id: "demo" }, edge: { domain: "demo.example" } }));
    fetchHealth.mockResolvedValue({ service: "cloud-api", timestamp: "2026-01-01T00:00:00Z" });
    const onResume = vi.fn();
    renderWithProviders(<Dashboard onStartNew={vi.fn()} onResume={onResume} />);
    expect(await screen.findByText("Resume Deployment")).toBeInTheDocument();
    expect(screen.getByText("demo")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("dashboard-resume-button"));
    expect(onResume).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByTestId("dashboard-discard-button"));
    await waitFor(() => expect(screen.getByText("No Saved Progress")).toBeInTheDocument());
  });

  it("handles malformed saved state and an unavailable API", async () => {
    saveDeployment("not-json");
    fetchHealth.mockRejectedValue(new Error("offline"));
    renderWithProviders(<Dashboard onStartNew={vi.fn()} onResume={vi.fn()} />);
    expect(await screen.findByText("API Offline")).toBeInTheDocument();
    expect(screen.getByText("API Not Available")).toBeInTheDocument();
    expect(screen.getByText("No Saved Progress")).toBeInTheDocument();
  });
});
