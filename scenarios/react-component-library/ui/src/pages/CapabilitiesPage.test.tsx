import { cleanup, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { describeCapabilities } = vi.hoisted(() => ({ describeCapabilities: vi.fn() }));
vi.mock("../api/catalog", () => ({ describeCapabilities }));

import { CapabilitiesPage } from "./CapabilitiesPage";
import { renderWithProviders } from "@vrooli/api-base/testing";

describe("CapabilitiesPage", () => {
  afterEach(() => cleanup());
  beforeEach(() => describeCapabilities.mockReset());

  it("makes availability and recovery guidance readable without the CLI", async () => {
    describeCapabilities.mockResolvedValue({
      states: [
        {
          id: "agent-manager",
          name: "Agent Manager",
          description: "Workflow execution",
          dependencyKind: "scenario",
          dependencySlug: "agent-manager",
          status: "unavailable",
          message: "Scenario is stopped",
          actionLabel: "Start Agent Manager",
          operatorCommand: "vrooli scenario start agent-manager --json",
        },
      ],
    });

    renderWithProviders(<CapabilitiesPage />);

    expect(
      await screen.findByRole("heading", { name: "Capability readiness" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Agent Manager")).toBeInTheDocument();
    expect(screen.getByText("Start Agent Manager")).toBeInTheDocument();
    expect(screen.getByText("vrooli scenario start agent-manager --json")).toBeInTheDocument();
  });

  it("explains when no capabilities are declared", async () => {
    describeCapabilities.mockResolvedValue({ states: [] });
    renderWithProviders(<CapabilitiesPage />);
    expect(await screen.findByText("No capabilities declared")).toBeInTheDocument();
  });

  it("distinguishes available and unknown states with dependency fallback copy", async () => {
    describeCapabilities.mockResolvedValue({
      states: [
        {
          id: "ready",
          name: "Ready",
          description: "Available",
          dependencyKind: "resource",
          dependencySlug: "postgres",
          status: "available",
        },
        {
          id: "unknown",
          name: "Unknown",
          description: "Needs inspection",
          dependencyKind: "scenario",
          dependencySlug: "worker",
          status: "unknown",
        },
      ],
    });
    renderWithProviders(<CapabilitiesPage />);
    expect(await screen.findByText("Dependency: postgres")).toBeInTheDocument();
    expect(screen.getByText("Dependency: worker")).toBeInTheDocument();
  });
});
