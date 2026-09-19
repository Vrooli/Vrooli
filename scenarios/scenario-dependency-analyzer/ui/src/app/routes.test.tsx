import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders, expectNoA11yViolations } from "../test-utils";
import { routes, TestAppRouter } from "./routes";

const graphPayload = {
  id: "test-graph",
  graph_type: "combined",
  nodes: [
    {
      id: "alpha",
      label: "Alpha Scenario",
      type: "scenario",
      group: "business"
    },
    {
      id: "postgres",
      label: "Postgres",
      type: "resource",
      group: "storage"
    }
  ],
  edges: [
    {
      source: "alpha",
      target: "postgres",
      label: "stores data in",
      type: "resource",
      required: true,
      weight: 2
    }
  ],
  metadata: { complexity_score: 0 }
};

const scenarioSummary = {
  name: "alpha",
  display_name: "Alpha Scenario",
  description: "A scenario with incomplete deployment metadata.",
  last_scanned: "2026-06-14T12:00:00Z"
};

const scenarioDetail = {
  scenario: "alpha",
  display_name: "Alpha Scenario",
  description: "A scenario with incomplete deployment metadata.",
  last_scanned: "2026-06-14T12:00:00Z",
  declared_resources: {},
  declared_scenarios: {},
  stored_dependencies: {
    resources: [],
    scenarios: [],
    shared_workflows: []
  },
  resource_diff: {
    missing: [{ name: "postgres" }],
    extra: []
  },
  scenario_diff: {
    missing: [],
    extra: [{ name: "deployment-manager" }]
  },
  optimization_recommendations: []
};

const deploymentReport = {
  scenario: "alpha",
  report_version: 1,
  generated_at: "2026-06-14T12:00:00Z",
  dependencies: [],
  aggregates: {
    desktop: {
      fitness_score: 0.42,
      dependency_count: 1,
      blocking_dependencies: ["postgres"],
      estimated_requirements: {
        ram_mb: 512,
        disk_mb: 1024,
        cpu_cores: 1
      }
    }
  },
  metadata_gaps: {
    total_gaps: 3,
    scenarios_missing_all: 1,
    gaps_by_scenario: {
      alpha: {
        scenario_name: "alpha",
        scenario_path: "scenarios/alpha",
        has_deployment_block: false,
        missing_dependency_catalog: true,
        missing_tier_definitions: ["desktop"],
        suggested_actions: ["Add deployment metadata."]
      }
    },
    missing_tiers: ["desktop"],
    recommendations: ["Add deployment metadata."]
  }
};

let failGraph = false;
let failScan = false;

function jsonResponse(payload: unknown, status = 200): Response {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { "Content-Type": "application/json" }
  });
}

beforeEach(() => {
  failGraph = false;
  failScan = false;
  vi.restoreAllMocks();
  vi.spyOn(globalThis, "fetch").mockImplementation((input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    if (url.includes("/health/analysis")) return Promise.resolve(jsonResponse({ status: "healthy" }));
    if (url.includes("/graph/")) {
      return Promise.resolve(failGraph ? jsonResponse({ error: "boom" }, 500) : jsonResponse(graphPayload));
    }
    if (url.endsWith("/scenarios")) return Promise.resolve(jsonResponse([scenarioSummary]));
    if (url.endsWith("/scenarios/alpha")) return Promise.resolve(jsonResponse(scenarioDetail));
    if (url.endsWith("/scenarios/alpha/deployment")) return Promise.resolve(jsonResponse(deploymentReport));
    if (url.endsWith("/scenarios/alpha/scan") && init?.method === "POST") {
      return Promise.resolve(failScan ? jsonResponse({ error: "scan failed" }, 500) : jsonResponse({ ok: true }));
    }
    if (url.endsWith("/optimize") && init?.method === "POST") return Promise.resolve(jsonResponse({ ok: true }));
    return Promise.resolve(jsonResponse({}));
  });
});

describe("App routes", () => {
  it("registers the Governance route in the primary navigation model", () => {
    expect(routes).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ key: "governance", path: "/governance", label: "Governance" })
      ])
    );
  });

  it("renders a deep-linked graph route", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/graph?layout=grid&graph_type=scenario"]} />, { withoutRouter: true });

    await waitFor(() => expect(screen.getByText("System Telemetry")).toBeInTheDocument());
    expect(screen.getAllByTestId("sda-nav-graph").some((item) => item.getAttribute("aria-current") === "page")).toBe(true);
  });

  it("renders a keyboard-friendly graph data view and selects nodes from it", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/graph"]} />, { withoutRouter: true });

    await waitFor(() => expect(screen.getByRole("table", { name: "Dependency graph nodes" })).toBeInTheDocument());
    expect(screen.getByRole("table", { name: "Dependency graph edges" })).toBeInTheDocument();
    await waitFor(() => expect(screen.getAllByText("Alpha Scenario").length).toBeGreaterThan(0));

    const selectButton = screen.getAllByRole("button", { name: "Select" })[0];
    if (!selectButton) {
      throw new Error("Expected a graph node selection button");
    }
    fireEvent.click(selectButton);

    await waitFor(() => expect(screen.getAllByText("Connections").length).toBeGreaterThan(1));
  });

  it("keeps graph table empty states visible when filters hide all rows", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/graph"]} />, { withoutRouter: true });

    await waitFor(() => expect(screen.getByRole("table", { name: "Dependency graph nodes" })).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("Focus Search"), { target: { value: "no-such-dependency" } });

    expect(screen.getByText("No nodes match the current filters.")).toBeInTheDocument();
    expect(screen.getByText("No edges match the current filters.")).toBeInTheDocument();
  });

  it("navigates between primary surfaces", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });

    const catalogNav = screen
      .getAllByTestId("sda-nav-catalog")
      .find((element) => element.tagName === "BUTTON");
    if (!catalogNav) {
      throw new Error("Expected catalog navigation button");
    }
    fireEvent.click(catalogNav);

    await waitFor(() => expect(screen.getAllByText("Scenario Catalog").length).toBeGreaterThan(0));
    expect(screen.getAllByTestId("sda-nav-catalog").some((item) => item.getAttribute("aria-current") === "page")).toBe(true);
  });

  it("has no obvious shell accessibility violations", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });

    await waitFor(() => expect(screen.getByTestId("sda-layout-shell")).toBeInTheDocument());
    await expectNoA11yViolations(document.body);
  });

  it("surfaces graph loading failures without hiding the graph controls", async () => {
    failGraph = true;

    renderWithProviders(<TestAppRouter initialEntries={["/graph"]} />, { withoutRouter: true });

    await waitFor(() => expect(screen.getByText("We hit turbulence while loading data.")).toBeInTheDocument());
    expect(screen.getByText("Scenario API request failed (500)")).toBeInTheDocument();
    expect(screen.getByText("Scenario Controls")).toBeInTheDocument();
  });

  it("runs catalog scan and apply through the API client seam", async () => {
    const fetchMock = vi.mocked(globalThis.fetch);
    renderWithProviders(<TestAppRouter initialEntries={["/catalog?scenario=alpha"]} />, { withoutRouter: true });

    await waitFor(() => expect(screen.getAllByText("Alpha Scenario").length).toBeGreaterThan(0));
    fireEvent.click(screen.getByRole("button", { name: "Scan & apply" }));

    await waitFor(() => {
      const scanCall = fetchMock.mock.calls.find(([url]) => String(url).includes("/scenarios/alpha/scan"));
      expect(scanCall).toBeDefined();
      const scanInit = scanCall?.[1] as RequestInit | undefined;
      expect(scanInit?.method).toBe("POST");
      expect(String(scanInit?.body)).toContain("\"apply\":true");
    });
  });

  it("renders deployment degraded states from metadata gaps and blockers", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/deployment"]} />, { withoutRouter: true });

    await waitFor(() => expect(screen.getByText("Alpha Scenario")).toBeInTheDocument());
    expect(screen.getByText("Critical gaps")).toBeInTheDocument();
    expect(screen.getAllByText("Critical").length).toBeGreaterThan(0);
    expect(screen.getByText("Missing metadata: 3")).toBeInTheDocument();
    expect(screen.getByText("Blockers (desktop): 1")).toBeInTheDocument();
  });

  it("opens deployment details from the keyboard-friendly status list", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/deployment"]} />, { withoutRouter: true });

    const statusRow = await screen.findByRole("button", { name: /Alpha Scenario/i });
    fireEvent.keyDown(statusRow, { key: "Enter" });

    await waitFor(() => expect(screen.getByText("Deployment details: Alpha Scenario")).toBeInTheDocument());
    expect(screen.getByText("Requirements estimate")).toBeInTheDocument();
    expect(screen.getByText("Blocking dependencies")).toBeInTheDocument();
  });

  it("surfaces deployment scan failures from the mutation path", async () => {
    failScan = true;
    renderWithProviders(<TestAppRouter initialEntries={["/deployment"]} />, { withoutRouter: true });

    await screen.findByRole("button", { name: /Alpha Scenario/i });
    const scanButton = screen.getAllByRole("button", { name: "Scan" })[0];
    if (!scanButton) {
      throw new Error("Expected a deployment scan button");
    }
    fireEvent.click(scanButton);

    await waitFor(() => {
      expect(screen.getByText(/Scan failed\. Ensure the API is running/)).toBeInTheDocument();
    });
  });
});
