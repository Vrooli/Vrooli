import { vi } from "vitest";
import { create } from "@bufbuild/protobuf";

import {
  ScanScenarioResponseSchema,
} from "@vrooli/proto-types/ai-gateway/v1/conformance/conformance_pb";
import {
  ListProviderRolesResponseSchema,
} from "@vrooli/proto-types/ai-gateway/v1/inventory/inventory_pb";
import {
  ListRouteEvidenceResponseSchema,
  PreviewRouteResponseSchema,
} from "@vrooli/proto-types/ai-gateway/v1/routing/routing_pb";

export const providerRolesFixture = create(ListProviderRolesResponseSchema, {
  roles: [
    {
      provider: "ollama",
      role: "chat.default",
      capabilities: ["text-generation"],
      locality: "local",
      status: "ready",
      policySchemaVersion: "policy/v1",
    },
    {
      provider: "openrouter",
      role: "chat.default",
      capabilities: ["text-generation"],
      locality: "remote",
      status: "ready",
      policySchemaVersion: "policy/v1",
    },
  ],
  warnings: [],
});

export const routeEvidenceFixture = create(ListRouteEvidenceResponseSchema, {
  events: [
    {
      eventId: "evt-1",
      requestId: "req-1",
      scenario: "prompt-injection-arena",
      operation: "judge.prompt",
      role: "chat.default",
      profile: 2,
      privacyClass: 2,
      selectedProvider: "ollama",
      selectedLocality: "local",
      status: "ok",
      policyReasons: ["local-first profile selected local provider"],
      failureReasons: [],
      fallbackUsed: false,
      promptRedacted: true,
      responseRedacted: true,
      latencyMs: 42n,
      createdAt: "2026-07-06T12:00:00Z",
    },
  ],
});

export const conformanceFixture = create(ScanScenarioResponseSchema, {
  scenario: "ai-gateway",
  maturityLevel: "L3",
  findings: [
    {
      ruleId: "AIGW_DIRECT_PROVIDER_URL",
      severity: "medium",
      path: "api/main.go",
      message: "Direct provider URL usage found.",
      remediation: "Route through AI Gateway roles.",
    },
  ],
  recommendations: ["Replace direct provider assumptions with role/profile calls."],
});

export const previewFixture = create(PreviewRouteResponseSchema, {
  valid: true,
  issues: [],
  candidates: [
    {
      provider: "ollama",
      role: "chat.default",
      locality: "local",
      selected: true,
      reasons: ["local provider satisfies local-first profile"],
      fallbackEligible: true,
    },
    {
      provider: "openrouter",
      role: "chat.default",
      locality: "remote",
      selected: false,
      reasons: ["remote candidate kept as fallback"],
      fallbackEligible: true,
    },
  ],
  selectedProvider: "ollama",
  policyReasons: ["profile local-first prefers local providers"],
  fallbackAllowed: true,
  routePlanId: "plan-ui-test",
});

export const makeGatewayApiMocks = () => ({
  listProviderRoles: vi.fn().mockResolvedValue(providerRolesFixture),
  listRouteEvidence: vi.fn().mockResolvedValue(routeEvidenceFixture),
  previewRoute: vi.fn().mockResolvedValue(previewFixture),
  scanScenario: vi.fn().mockResolvedValue(conformanceFixture),
  smokeProvider: vi.fn().mockResolvedValue({
    provider: "ollama",
    status: "ready",
    code: "ok",
    message: "provider ready",
    exitCode: 0,
    warnings: [],
  }),
});
