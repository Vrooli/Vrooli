import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { IncidentsPage } from "./IncidentsPage";
import type { Incident, IncidentObservation } from "../../lib/api";

const incidentMocks = vi.hoisted(() => ({
  fetchIncidents: vi.fn(),
  fetchIncidentObservations: vi.fn(),
  updateIncidentStatus: vi.fn(),
}));

vi.mock("../../lib/api", () => incidentMocks);

const incidents: Incident[] = [
  {
    id: "inc-1",
    fingerprint: "fp-1",
    type: "resource_failure",
    severity: "critical",
    status: "open",
    title: "Postgres unavailable",
    summary: "The database resource stopped responding.",
    detectedAt: new Date(Date.now() - 120000).toISOString(),
    lastSeenAt: new Date(Date.now() - 30000).toISOString(),
    updatedAt: new Date().toISOString(),
    sourceCheckIds: ["resource-postgres"],
    recommendations: ["Restart the resource"],
    eventCount: 4,
    observationCount: 1,
  },
  {
    id: "inc-2",
    fingerprint: "fp-2",
    type: "manual",
    severity: "info",
    status: "resolved",
    title: "Maintenance",
    summary: "A completed maintenance event.",
    detectedAt: new Date(Date.now() - 3600000).toISOString(),
    lastSeenAt: new Date(Date.now() - 1800000).toISOString(),
    updatedAt: new Date().toISOString(),
    eventCount: 1,
    observationCount: 0,
    recommendations: [],
  },
];

const observations: IncidentObservation[] = [
  {
    id: 1,
    incidentId: "inc-1",
    observedAt: new Date().toISOString(),
    severity: "critical",
    status: "open",
    message: "Connection refused",
  },
];

describe("IncidentsPage", () => {
  beforeEach(() => {
    incidentMocks.fetchIncidents.mockResolvedValue({ incidents, total: incidents.length, filters: {} });
    incidentMocks.fetchIncidentObservations.mockResolvedValue({ observations, total: observations.length });
    incidentMocks.updateIncidentStatus.mockResolvedValue(incidents[0]);
  });

  it("filters incidents, selects evidence, and performs recovery actions", async () => {
    renderWithProviders(<IncidentsPage />);
    expect((await screen.findAllByText("Postgres unavailable")).length).toBeGreaterThan(0);
    expect(await screen.findByText("Connection refused")).toBeInTheDocument();
    expect(screen.getByText("Restart the resource")).toBeInTheDocument();

    const incidentTitles = screen.getAllByText("Postgres unavailable");
    const firstIncidentTitle = incidentTitles[0];
    if (!firstIncidentTitle) throw new Error("Incident title was not rendered");
    fireEvent.click(firstIncidentTitle);
    await waitFor(() => expect(incidentMocks.fetchIncidentObservations).toHaveBeenCalledWith("inc-1"));
    const filters = screen.getAllByRole("combobox");
    expect(filters).toHaveLength(3);
    const [statusFilter, severityFilter, typeFilter] = filters;
    if (!statusFilter || !severityFilter || !typeFilter) throw new Error("Incident filters were not rendered");
    fireEvent.change(statusFilter, { target: { value: "acknowledged" } });
    fireEvent.change(severityFilter, { target: { value: "critical" } });
    fireEvent.change(typeFilter, { target: { value: "manual" } });
  });

  it("shows the empty state and handles a failed request", async () => {
    incidentMocks.fetchIncidents.mockResolvedValueOnce({ incidents: [], total: 0, filters: {} });
    renderWithProviders(<IncidentsPage />);
    expect(await screen.findByText(/no incidents match/i)).toBeInTheDocument();

    incidentMocks.fetchIncidents.mockRejectedValueOnce(new Error("network down"));
    renderWithProviders(<IncidentsPage />);
    expect(await screen.findByText(/unable to load incidents/i)).toBeInTheDocument();
  });
});
