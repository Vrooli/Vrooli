import { beforeEach, describe, expect, it, vi } from "vitest";

import { Profile, PrivacyClass, RequestKind } from "@vrooli/proto-types/ai-gateway/v1/shared/gateway_pb";

const clients = {
  inventory: {
    listProviderRoles: vi.fn(),
    smokeProvider: vi.fn(),
  },
  routing: {
    previewRoute: vi.fn(),
    listRouteEvidence: vi.fn(),
  },
  conformance: {
    scanScenario: vi.fn(),
  },
};

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn((service: { typeName?: string }) => {
    if (service.typeName?.includes("InventoryService")) {
      return clients.inventory;
    }
    if (service.typeName?.includes("RoutingService")) {
      return clients.routing;
    }
    return clients.conformance;
  }),
}));

describe("api/gateway", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("builds provider-neutral gateway requests", async () => {
    vi.spyOn(Date, "now").mockReturnValueOnce(1234);
    const { buildGatewayRequest } = await import("./gateway");

    const req = buildGatewayRequest({
      scenario: "portal",
      operation: "chat.complete",
      role: "chat.default",
      profile: Profile.PRIVACY_SENSITIVE,
      privacyClass: PrivacyClass.CONFIDENTIAL,
      timeoutMs: 1500,
      maxCostUsd: 0.02,
      maxOutputTokens: 256,
    });

    expect(req).toMatchObject({
      kind: RequestKind.TEXT_GENERATION,
      role: "chat.default",
      profile: Profile.PRIVACY_SENSITIVE,
      privacyClass: PrivacyClass.CONFIDENTIAL,
      operation: "chat.complete",
      scenario: "portal",
      timeoutMs: 1500,
      maxCostUsd: 0.02,
      maxOutputTokens: 256,
      requestId: "ui-1234",
      metadata: {},
    });
  });

  it("passes typed requests to generated Connect clients", async () => {
    clients.inventory.listProviderRoles.mockResolvedValueOnce({ roles: [] });
    clients.inventory.smokeProvider.mockResolvedValueOnce({ provider: "ollama" });
    clients.routing.previewRoute.mockResolvedValueOnce({ valid: true });
    clients.routing.listRouteEvidence.mockResolvedValueOnce({ events: [] });
    clients.conformance.scanScenario.mockResolvedValueOnce({ findings: [] });
    const {
      defaultPreviewInput,
      listProviderRoles,
      listRouteEvidence,
      previewRoute,
      scanScenario,
      smokeProvider,
    } = await import("./gateway");

    await listProviderRoles("ollama");
    await smokeProvider("openrouter");
    await previewRoute(defaultPreviewInput);
    await listRouteEvidence(25, "portal");
    await scanScenario("portal", "/tmp/portal");

    expect(clients.inventory.listProviderRoles.mock.calls[0]?.[0].provider).toBe("ollama");
    expect(clients.inventory.smokeProvider.mock.calls[0]?.[0].provider).toBe("openrouter");
    expect(clients.routing.previewRoute.mock.calls[0]?.[0].request.role).toBe("chat.default");
    expect(clients.routing.listRouteEvidence.mock.calls[0]?.[0]).toMatchObject({
      limit: 25,
      scenario: "portal",
    });
    expect(clients.conformance.scanScenario.mock.calls[0]?.[0]).toMatchObject({
      scenario: "portal",
      path: "/tmp/portal",
    });
  });
});
