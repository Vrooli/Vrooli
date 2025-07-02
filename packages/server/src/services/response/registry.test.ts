import { AnthropicModel, MistralModel, OpenAIModel } from "@vrooli/shared";
import { expect, describe, it, beforeEach, afterEach, vi } from "vitest";
import { LlmServiceErrorType, LlmServiceId, LlmServiceRegistry, LlmServiceState } from "./registry.js";

describe("LlmServiceRegistry", () => {
    let registry: LlmServiceRegistry;

    beforeEach(async () => {
        vi.useFakeTimers();
        LlmServiceRegistry["instance"] = undefined;
        await LlmServiceRegistry.init();
        registry = LlmServiceRegistry.get();
    });

    afterEach(() => {
        vi.runAllTimers();
        vi.restoreAllMocks();
        vi.useRealTimers();
    });

    it("returns the preferred service when active", () => {
        expect(registry.getBestService(AnthropicModel.Opus3)).toBe(LlmServiceId.Anthropic);
        expect(registry.getBestService(OpenAIModel.Gpt4o_Mini)).toBe(LlmServiceId.OpenAI);
        expect(registry.getBestService(MistralModel.Codestral)).toBe(LlmServiceId.Mistral);
    });

    it("returns the first active fallback service when preferred is on cooldown", () => {
        registry.updateServiceState("Anthropic", LlmServiceErrorType.Overloaded);
        expect(registry.getBestService(AnthropicModel.Opus3)).not.toBe(LlmServiceId.Anthropic);
    });

    it("returns the next active fallback service when preferred and first fallback are on cooldown", () => {
        registry.updateServiceState("Anthropic", LlmServiceErrorType.Overloaded); // Set Anthropic to cooldown
        registry.updateServiceState("Mistral", LlmServiceErrorType.Overloaded); // Set Mistral to cooldown

        const bestServiceId = registry.getBestService(AnthropicModel.Sonnet3_5);
        expect(bestServiceId).not.toBe(LlmServiceId.Anthropic);
        expect(bestServiceId).not.toBe(LlmServiceId.Mistral);
    });

    it("returns null when all services are on cooldown or disabled", () => {
        Object.values(LlmServiceId).forEach(serviceId => {
            registry.updateServiceState(serviceId, LlmServiceErrorType.Overloaded);
        });

        expect(registry.getBestService(AnthropicModel.Opus3)).toBeNull();
    });

    it("registers and retrieves service states correctly", () => {
        registry.registerService("testService");
        registry.registerService("testService2");
        expect(registry.getServiceState("testService")).toBe(LlmServiceState.Active);
        expect(registry.getServiceState("testService2")).toBe(LlmServiceState.Active);
    });

    it("automatically registers and activates an unregistered service when its state is requested", () => {
        const serviceState = registry.getServiceState("autoRegisteredService");
        expect(serviceState).toBe(LlmServiceState.Active);
    });

    it("handles critical error by disabling service", () => {
        registry.registerService("criticalErrorService");
        registry.registerService("fineService");
        registry.updateServiceState("criticalErrorService", LlmServiceErrorType.Authentication);
        expect(registry.getServiceState("criticalErrorService")).toBe(LlmServiceState.Disabled);
        expect(registry.getServiceState("fineService")).toBe(LlmServiceState.Active);

        // Fast-forward to make sure it stays disabled
        vi.advanceTimersByTime(1_000_000);
        expect(registry.getServiceState("criticalErrorService")).toBe(LlmServiceState.Disabled);
    });

    it("handles cooldown error by setting service into cooldown", () => {
        registry.registerService("cooldownService");
        registry.updateServiceState("cooldownService", LlmServiceErrorType.Overloaded);
        expect(registry.getServiceState("cooldownService")).toBe(LlmServiceState.Cooldown);
    });

    it("resets service from cooldown to active after cooldown period", async () => {
        registry.registerService("resetService");
        registry.updateServiceState("resetService", LlmServiceErrorType.Overloaded);

        // Fast-forward a few seconds to make sure it's still in cooldown
        vi.advanceTimersByTime(10_000);
        expect(registry.getServiceState("resetService")).toBe(LlmServiceState.Cooldown);

        // Fast-forward to the end of the cooldown period (15 minutes)
        vi.advanceTimersByTime(15 * 60 * 1000);
        expect(registry.getServiceState("resetService")).toBe(LlmServiceState.Active);
    });
});
