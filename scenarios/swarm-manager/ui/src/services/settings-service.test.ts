import { describe, it, expect, vi, beforeEach } from "vitest";
import { createSettingsService, DEFAULT_SETTINGS, type ISettingsService } from "./settings-service";
import type { IApiClient } from "../lib/api-client";

describe("Settings Service update() write path", () => {
  let mockApiClient: IApiClient;
  let service: ISettingsService;

  beforeEach(() => {
    mockApiClient = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    };
    // update() re-parses the response against the strict domain Settings
    // protovalidate constraints, so return a minimally-valid settings body.
    vi.mocked(mockApiClient.put).mockResolvedValue({
      settings: {
        theme: "dark",
        defaultMode: "manual",
        circuitBreakerThreshold: 1,
        circuitBreakerCooldownMinutes: 5,
        agentMaxTurns: 5,
        agentTimeoutSeconds: 60,
        searchDebounceMs: 100,
        toastDurationMs: 1000,
        fixBeforeFeature: "suggest",
        autoFiler: {
          mode: "suggest",
          strategy: "feature_pending",
          maxOpenAutoFiled: 10,
          velocityWindowDays: 7,
          minVelocityTransitions: 1,
          intervalMinutes: 30,
          goalName: "automated-maintenance",
        },
      },
    });
    service = createSettingsService(mockApiClient);
  });

  // Regression: these fields are editable in the Execution tab but were
  // previously dropped from the proto-JSON update body, so saving silently
  // discarded them. Lock the full Governance + fix-before-feature set in.
  it("serializes governance and fix-before-feature fields into the update body", async () => {
    await service.update({
      ...DEFAULT_SETTINGS,
      maxQueueDepth: 42,
      circuitBreakerThreshold: 7,
      circuitBreakerCooldownMinutes: 30,
      executionCostCapPerRun: 12.5,
      costPerTurnEstimate: 0.25,
      fixBeforeFeature: "block",
      autoFiler: {
        enabled: true,
        mode: "auto_add",
        strategy: "importance",
        maxOpenAutoFiled: 4,
        velocityWindowDays: 14,
        minVelocityTransitions: 2,
        intervalMinutes: 5,
        goalName: "maintenance",
      },
    });

    expect(mockApiClient.put).toHaveBeenCalledTimes(1);
    // toProtoJson emits proto field (snake_case) names.
    const [, body] = vi.mocked(mockApiClient.put).mock.calls[0]!;
    expect(body).toMatchObject({
      max_queue_depth: 42,
      circuit_breaker_threshold: 7,
      circuit_breaker_cooldown_minutes: 30,
      execution_cost_cap_per_run: 12.5,
      cost_per_turn_estimate: 0.25,
      fix_before_feature: "block",
      auto_filer: {
        enabled: true,
        mode: "auto_add",
        strategy: "importance",
        max_open_auto_filed: 4,
        velocity_window_days: 14,
        min_velocity_transitions: 2,
        interval_minutes: 5,
        goal_name: "maintenance",
      },
    });
  });

  it("omits fix-before-feature and auto-filer keys when not present in the patch", async () => {
    await service.update({ maxQueueDepth: 10 });

    const [, body] = vi.mocked(mockApiClient.put).mock.calls[0]!;
    expect(body).not.toHaveProperty("fix_before_feature");
    expect(body).not.toHaveProperty("auto_filer");
  });
});
