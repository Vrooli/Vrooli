import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import type { ReactElement } from "react";

import { HandoffState } from "@vrooli/proto-types/persona/v1/handoffs/handoffs_pb";
import { PersonaKind, PersonaStatus } from "@vrooli/proto-types/persona/v1/personas/personas_pb";
import { renderWithProviders } from "../test-utils";
import { HandoffsPage } from "./HandoffsPage";
import { JournalPage } from "./JournalPage";
import { PersonaDetailPage } from "./PersonaDetailPage";
import { PersonasPage } from "./PersonasPage";
import { DashboardPage } from "./DashboardPage";

const api = vi.hoisted(() => ({
  listPersonas: vi.fn(),
  createPersona: vi.fn(),
  getPersona: vi.fn(),
  checkPersonaHealth: vi.fn(),
  listHandoffs: vi.fn(),
  listAllHandoffs: vi.fn(),
  listChannels: vi.fn(),
  retrieveCode: vi.fn(),
  getHandoff: vi.fn(),
  completeHandoff: vi.fn(),
  listJournal: vi.fn(),
  listAllJournal: vi.fn(),
}));

vi.mock("../api/persona", () => api);
vi.mock("../features/health/HealthCard", () => ({ HealthCard: () => <div>Health summary</div> }));

const persona = {
  id: "p1",
  kind: PersonaKind.PERSONAL,
  status: PersonaStatus.ACTIVE,
  displayName: "Ada's persona",
  legalBasis: { subjectId: "subject-1", subjectName: "Ada", basisType: "operator-authorisation" },
};
const handoff = {
  id: "h1",
  title: "Confirm identity",
  kind: "identity",
  state: HandoffState.AWAITING_HUMAN,
  humanAction: "Review the identity",
  checkpoint: { completedFields: [{ name: "email", value: "ada@example.test" }], requiredDocumentIds: ["doc-1"] },
};
const channel = { id: "c1", address: "ada@example.test", adapter: "email", enabled: true };

function renderRoute(element: ReactElement, path: string) {
  return renderWithProviders(
    <MemoryRouter initialEntries={[path]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <Routes>
        <Route path="/personas/:personaId" element={element} />
        <Route path="/personas" element={element} />
        <Route path="/handoffs/:handoffId" element={element} />
        <Route path="/handoffs" element={element} />
        <Route path="/journal" element={element} />
        <Route path="/" element={element} />
      </Routes>
    </MemoryRouter>,
    { withoutRouter: true },
  );
}

describe("persona product pages", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.listPersonas.mockResolvedValue([persona]);
    api.createPersona.mockResolvedValue(persona);
    api.getPersona.mockResolvedValue(persona);
    api.checkPersonaHealth.mockResolvedValue([{ code: "mailbox_unavailable", message: "Mailbox is offline", blocking: true }]);
    api.listHandoffs.mockResolvedValue([handoff]);
    api.listAllHandoffs.mockResolvedValue([handoff]);
    api.listChannels.mockResolvedValue([channel]);
    api.retrieveCode.mockResolvedValue({ code: "123456", expiresAt: { seconds: BigInt(2000) } });
    api.getHandoff.mockResolvedValue(handoff);
    api.completeHandoff.mockResolvedValue({ ...handoff, state: HandoffState.COMPLETED });
    api.listJournal.mockResolvedValue([{ id: "j1", verb: "create", outcome: "recorded", actor: "agent", at: { seconds: BigInt(1000) } }]);
    api.listAllJournal.mockResolvedValue([{ id: "j1", verb: "create", outcome: "recorded", actor: "agent", at: { seconds: BigInt(1000) } }]);
  });

  afterEach(() => cleanup());

  it("creates a persona with a declared identity basis", async () => {
    const user = userEvent.setup();
    renderRoute(<PersonasPage />, "/personas");
    expect(await screen.findByText("Ada's persona")).toBeInTheDocument();
    await user.type(screen.getByLabelText("Subject ID"), "subject-2");
    await user.type(screen.getByLabelText("Subject name"), "Grace");
    await user.type(screen.getByLabelText("Identifier value"), "P-2");
    await user.click(screen.getByRole("button", { name: "Create persona" }));
    await waitFor(() => expect(api.createPersona).toHaveBeenCalled());
    expect(await screen.findByRole("status")).toHaveTextContent("Persona created");
  });

  it("shows the dashboard queue and its empty state", async () => {
    renderRoute(<DashboardPage />, "/");
    expect(await screen.findByText("Confirm identity")).toBeInTheDocument();
    expect(screen.getByText("Open")).toBeInTheDocument();

    cleanup();
    api.listAllHandoffs.mockResolvedValueOnce([]);
    renderRoute(<DashboardPage />, "/");
    expect(await screen.findByText(/Nothing is waiting/)).toBeInTheDocument();
    expect(screen.getByText("Ready")).toBeInTheDocument();
  });

  it("shows the controlled route and announces a retrieved code", async () => {
    const user = userEvent.setup();
    renderRoute(<PersonaDetailPage />, "/personas/p1");
    expect(await screen.findByText("Ada's persona")).toBeInTheDocument();
    expect(screen.getByText("Mailbox is offline")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Retrieve code" }));
    expect(await screen.findByRole("status")).toHaveTextContent("123456");
  });

  it("keeps handoffs resumable and completes the human step", async () => {
    const user = userEvent.setup();
    renderRoute(<HandoffsPage />, "/handoffs");
    expect(await screen.findByText("Confirm identity")).toBeInTheDocument();
    await user.click(screen.getByText("Confirm identity"));
    expect(await screen.findByText("Completed fields")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Mark human step complete" }));
    await waitFor(() => expect(api.completeHandoff).toHaveBeenCalledWith("h1"));
  });

  it("renders append-only journal evidence", async () => {
    renderRoute(<JournalPage />, "/journal");
    expect(await screen.findByText("Activity trail")).toBeInTheDocument();
    expect(await screen.findByText("create")).toBeInTheDocument();
    expect(screen.getByText("recorded")).toBeInTheDocument();
  });

  it("surfaces empty and refused registry states", async () => {
    api.listPersonas.mockRejectedValueOnce(new Error("registry unavailable"));
    renderRoute(<PersonasPage />, "/personas");
    expect(await screen.findByRole("alert")).toHaveTextContent("Unable to load personas");

    cleanup();
    api.listPersonas.mockResolvedValueOnce([]);
    renderRoute(<PersonasPage />, "/personas");
    expect(await screen.findByText(/No persona exists yet/)).toBeInTheDocument();
  });

  it("keeps degraded queue and journal reads explicit", async () => {
    api.listAllHandoffs.mockRejectedValueOnce(new Error("queue unavailable"));
    renderRoute(<HandoffsPage />, "/handoffs");
    expect(await screen.findByRole("alert")).toHaveTextContent("queue is unavailable");

    cleanup();
    api.listAllJournal.mockRejectedValueOnce(new Error("journal unavailable"));
    renderRoute(<JournalPage />, "/journal");
    expect(await screen.findByRole("alert")).toHaveTextContent("journal is unavailable");
  });

  it("renders empty queues and terminal handoff checkpoints", async () => {
    api.listAllHandoffs.mockResolvedValueOnce([]);
    renderRoute(<HandoffsPage />, "/handoffs");
    expect(await screen.findByText("Nothing is waiting for a human.")).toBeInTheDocument();

    cleanup();
    api.getHandoff.mockResolvedValueOnce({ ...handoff, title: "", kind: "completed-work", state: HandoffState.COMPLETED, checkpoint: { completedFields: [], requiredDocumentIds: [] } });
    renderRoute(<HandoffsPage />, "/handoffs/h1");
    expect(await screen.findByText(/This handoff is terminal or already resumed/)).toBeInTheDocument();
  });

  it("keeps an empty journal explicit", async () => {
    api.listAllJournal.mockResolvedValueOnce([]);
    renderRoute(<JournalPage />, "/journal");
    expect(await screen.findByText("No actions recorded yet.")).toBeInTheDocument();
  });

  it("renders archived business records in the registry", async () => {
    api.listPersonas.mockResolvedValueOnce([{ ...persona, kind: PersonaKind.BUSINESS, status: PersonaStatus.ARCHIVED, displayName: "", legalBasis: { ...persona.legalBasis, basisType: "" } }]);
    renderRoute(<PersonasPage />, "/personas");
    expect(await screen.findByText("Business · Archived")).toBeInTheDocument();
    expect(screen.getByText("Legal basis: Not declared")).toBeInTheDocument();
  });

  it("renders terminal handoffs and an archived business persona", async () => {
    api.getHandoff.mockRejectedValueOnce(new Error("handoff missing"));
    renderRoute(<HandoffsPage />, "/handoffs/h-missing");
    expect(await screen.findByRole("alert")).toHaveTextContent("could not be loaded");

    cleanup();
    api.getPersona.mockResolvedValueOnce({ ...persona, kind: PersonaKind.BUSINESS, status: PersonaStatus.ARCHIVED, displayName: "", legalBasis: { ...persona.legalBasis, subjectName: "Acme" } });
    api.checkPersonaHealth.mockResolvedValueOnce([]);
    api.listHandoffs.mockResolvedValueOnce([]);
    api.listChannels.mockResolvedValueOnce([]);
    renderRoute(<PersonaDetailPage />, "/personas/p1");
    expect(await screen.findByText("Business persona")).toBeInTheDocument();
    expect(screen.getByText("Archived")).toBeInTheDocument();
    expect(screen.getByText("No blocking findings.")).toBeInTheDocument();
    expect(screen.getByText("No controlled route is configured yet.")).toBeInTheDocument();
  });

  it("announces route refusal and dependency health errors", async () => {
    api.retrieveCode.mockRejectedValueOnce(new Error("provider refused"));
    renderRoute(<PersonaDetailPage />, "/personas/p1");
    await screen.findByText("Ada's persona");
    await userEvent.setup().click(screen.getByRole("button", { name: "Retrieve code" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("refused retrieval");

    cleanup();
    api.checkPersonaHealth.mockRejectedValueOnce(new Error("health unavailable"));
    renderRoute(<PersonaDetailPage />, "/personas/p1");
    expect(await screen.findByRole("alert")).toHaveTextContent("Health is unavailable");
  });
});
