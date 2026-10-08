import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const client = vi.hoisted(() => ({
  listPersonas: vi.fn(),
  getPersona: vi.fn(),
  createPersona: vi.fn(),
  checkHealth: vi.fn(),
  listHandoffs: vi.fn(),
  listChannels: vi.fn(),
  retrieveCode: vi.fn(),
  getHandoff: vi.fn(),
  completeHandoff: vi.fn(),
  list: vi.fn(),
}));

vi.mock("@connectrpc/connect", () => ({ createClient: vi.fn(() => client) }));

import { PersonaKind } from "@vrooli/proto-types/persona/v1/personas/personas_pb";
import {
  checkPersonaHealth,
  completeHandoff,
  createPersona,
  getHandoff,
  getPersona,
  listAllHandoffs,
  listAllJournal,
  listChannels,
  listHandoffs,
  listJournal,
  listPersonas,
  retrieveCode,
} from "./persona";

describe("persona API adapters", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("maps persona and health operations onto typed clients", async () => {
    const persona = { id: "p1" };
    client.listPersonas.mockResolvedValue({ personas: [persona] });
    client.getPersona.mockResolvedValue({ persona });
    client.createPersona.mockResolvedValue({ persona });
    client.checkHealth.mockResolvedValue({ findings: [{ code: "ok" }] });

    await expect(listPersonas(true)).resolves.toEqual([persona]);
    await expect(getPersona("p1")).resolves.toEqual(persona);
    await expect(createPersona({
      kind: PersonaKind.PERSONAL,
      subjectId: "subject",
      subjectName: "Ada",
      basisType: "operator-authorisation",
      displayName: "Ada",
      identifierType: "passport",
      identifierValue: "P-1",
    })).resolves.toEqual(persona);
    await expect(checkPersonaHealth("p1")).resolves.toEqual([{ code: "ok" }]);
    expect(client.listPersonas).toHaveBeenCalledWith({ limit: 100, includeArchived: true });
    expect(client.createPersona).toHaveBeenCalledWith(expect.objectContaining({ kind: PersonaKind.PERSONAL }));
  });

  it("maps handoff, channel, and journal operations", async () => {
    const handoff = { id: "h1" };
    const channel = { id: "c1" };
    const entry = { id: "j1" };
    client.listHandoffs.mockResolvedValue({ handoffs: [handoff] });
    client.listChannels.mockResolvedValue({ channels: [channel] });
    client.retrieveCode.mockResolvedValue({ code: "123456" });
    client.getHandoff.mockResolvedValue({ handoff });
    client.completeHandoff.mockResolvedValue({ handoff });
    client.list.mockResolvedValue({ entries: [entry] });

    await expect(listHandoffs("p1")).resolves.toEqual([handoff]);
    await expect(listChannels("p1")).resolves.toEqual([channel]);
    await expect(retrieveCode("p1", "c1", "login")).resolves.toEqual({ code: "123456" });
    await expect(getHandoff("h1")).resolves.toEqual(handoff);
    await expect(completeHandoff("h1")).resolves.toEqual(handoff);
    await expect(listJournal("p1")).resolves.toEqual([entry]);
    await expect(listAllHandoffs([{ id: "p1" }, { id: "p2" }])).resolves.toEqual([handoff, handoff]);
    await expect(listAllJournal([{ id: "p1" }, { id: "p2" }])).resolves.toEqual([entry, entry]);
    expect(client.completeHandoff).toHaveBeenCalledWith({ handoffId: "h1", completedBy: "operator" });
  });
});
