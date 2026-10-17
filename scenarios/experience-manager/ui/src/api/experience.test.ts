import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const clients = vi.hoisted(() => ({
  contract: {
    listFleet: vi.fn(),
    validateScenario: vi.fn(),
  },
  studio: {
    listSpec: vi.fn(),
    showSpec: vi.fn(),
    renderSpec: vi.fn(),
    suggestBindings: vi.fn(),
    compareVariants: vi.fn(),
    startAuthoringSession: vi.fn(),
    submitPage: vi.fn(),
    previewSession: vi.fn(),
    applySession: vi.fn(),
    promoteVariant: vi.fn(),
  },
  scenarioValidation: {
    previewFix: vi.fn(),
    applyFix: vi.fn(),
  },
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi
    .fn()
    .mockReturnValueOnce(clients.contract)
    .mockReturnValueOnce(clients.studio)
    .mockReturnValueOnce(clients.scenarioValidation),
}));

import {
  applyFindingsFixes,
  applyStudioDraft,
  compareStudioVariants,
  fetchEvidence,
  fetchFindings,
  fetchFleet,
  fetchScenarioSpec,
  previewFindingsFixes,
  promoteStudioVariant,
  recaptureScenario,
  renderStudioSpec,
  suggestStudioBindings,
  type StudioPageDraft,
} from "./experience";

const draft: StudioPageDraft = {
  id: "fleet",
  title: "Fleet",
  purpose: "",
  routes: ["/"],
  prdRefs: [],
  status: "",
  priorities: [],
  states: [],
  elements: [],
  claims: [],
  bindings: [],
  sketchRegions: [],
};

describe("api/experience wrappers", () => {
  beforeEach(() => {
    for (const client of Object.values(clients)) {
      for (const fn of Object.values(client)) {
        fn.mockReset();
      }
    }
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads fleet and scenario specs through typed clients", async () => {
    clients.contract.listFleet.mockResolvedValueOnce({ scenarios: [] });
    clients.studio.listSpec.mockResolvedValueOnce({ pages: [{ id: "fleet", title: "Fleet" }] });
    clients.studio.showSpec.mockResolvedValueOnce({
      json: JSON.stringify({ page: { id: "fleet", title: "Fleet" } }),
    });

    await expect(fetchFleet()).resolves.toEqual({ scenarios: [] });
    await expect(fetchScenarioSpec("experience-manager")).resolves.toEqual([
      {
        document: { id: "fleet", title: "Fleet" },
        spec: { page: { id: "fleet", title: "Fleet" } },
      },
    ]);
    expect(clients.studio.showSpec).toHaveBeenCalledWith({ scenario: "experience-manager", page: "fleet" });
  });

  it("loads REST evidence and decodes REST errors", async () => {
    const fetchSpy = vi.fn().mockResolvedValueOnce(
      new Response(JSON.stringify({ evidence: [{ id: "ev-1" }] }), { status: 200 }),
    );
    vi.stubGlobal("fetch", fetchSpy);

    await expect(fetchEvidence({ scenario: "experience-manager", page: "fleet", limit: 3 })).resolves.toEqual([
      { id: "ev-1" },
    ]);
    expect(fetchSpy.mock.calls[0]?.[1]).toMatchObject({
      method: "POST",
      body: JSON.stringify({ scenario: "experience-manager", page: "fleet", claim: "", limit: 3 }),
      cache: "no-store",
    });

    fetchSpy.mockResolvedValueOnce(
      new Response(JSON.stringify({ code: "internal", message: "down" }), { status: 500 }),
    );
    await expect(fetchEvidence({ scenario: "experience-manager", page: "fleet" })).rejects.toThrow("down");
  });

  it("wraps validation, findings, fixes, rendering, and binding calls", async () => {
    clients.contract.validateScenario
      .mockResolvedValueOnce({ report: { findings: [{ code: "x" }] } })
      .mockResolvedValueOnce({ validation: "ok" })
      .mockResolvedValueOnce({ report: { findings: [] } });
    clients.scenarioValidation.previewFix.mockResolvedValueOnce({ candidates: [] });
    clients.scenarioValidation.applyFix.mockResolvedValueOnce({ applied: true });
    clients.studio.renderSpec.mockResolvedValueOnce({ html: "<main />" });
    clients.studio.suggestBindings.mockResolvedValueOnce({ suggestions: [{ elementId: "debt-table" }] });
    clients.studio.compareVariants.mockResolvedValueOnce({
      html: "<main />",
      degradedReason: "",
      variants: [{ id: "draft" }],
    });

    await expect(recaptureScenario("experience-manager")).resolves.toEqual({ report: { findings: [{ code: "x" }] } });
    await expect(fetchFindings("experience-manager")).resolves.toEqual([]);
    await expect(previewFindingsFixes("experience-manager")).resolves.toEqual({ candidates: [] });
    await expect(applyFindingsFixes("experience-manager", ["rule"])).resolves.toMatchObject({
      preview: { applied: true },
      validation: { report: { findings: [] } },
    });
    await expect(renderStudioSpec("experience-manager", "fleet")).resolves.toEqual({ html: "<main />" });
    await expect(suggestStudioBindings("experience-manager", "fleet")).resolves.toEqual([{ elementId: "debt-table" }]);
    await expect(compareStudioVariants("experience-manager", "fleet", [{ id: "draft", title: "Draft", page: draft }])).resolves.toEqual({
      html: "<main />",
      degradedReason: "",
      variants: [{ id: "draft" }],
    });
  });

  it("applies Studio drafts only after a clean preview", async () => {
    clients.studio.startAuthoringSession.mockResolvedValueOnce({ session: { id: "sess-1" } });
    clients.studio.submitPage.mockResolvedValueOnce({});
    clients.studio.previewSession.mockResolvedValueOnce({ diffs: [{ path: "fleet.json" }], validation: { report: { findings: [] } } });
    clients.studio.applySession.mockResolvedValueOnce({ diffs: [{ path: "fleet.json" }], validation: { status: "passed" } });

    await expect(applyStudioDraft("experience-manager", draft)).resolves.toMatchObject({
      applied: true,
      diffs: [{ path: "fleet.json" }],
      validation: { status: "passed" },
    });
    expect(clients.studio.submitPage).toHaveBeenCalledWith({ sessionId: "sess-1", page: draft });
  });

  it("keeps Studio draft previews unapplied when validation has errors", async () => {
    clients.studio.startAuthoringSession.mockResolvedValueOnce({ session: { id: "sess-2" } });
    clients.studio.submitPage.mockResolvedValueOnce({});
    clients.studio.previewSession.mockResolvedValueOnce({
      diffs: [{ path: "fleet.json" }],
      validation: { report: { findings: [{ severity: "error" }] } },
    });

    await expect(applyStudioDraft("experience-manager", draft)).resolves.toMatchObject({
      applied: false,
      diffs: [{ path: "fleet.json" }],
    });
    expect(clients.studio.applySession).not.toHaveBeenCalled();
  });

  it("fails Studio draft application when no session is returned and promotes variants", async () => {
    clients.studio.startAuthoringSession.mockResolvedValueOnce({ session: undefined });
    await expect(applyStudioDraft("experience-manager", draft)).rejects.toThrow("studio session did not return an id");

    clients.studio.promoteVariant.mockResolvedValueOnce({ diffs: [], validation: { status: "passed" } });
    await expect(promoteStudioVariant("experience-manager", "fleet", { id: "draft", title: "Draft", page: draft })).resolves.toMatchObject({
      applied: true,
      validation: { status: "passed" },
    });
  });
});

