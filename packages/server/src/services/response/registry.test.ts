import { AnthropicModel, MistralModel, OpenAIModel } from "@vrooli/shared";
import { expect, describe, it, beforeEach, afterEach, vi } from "vitest";
import { AIServiceErrorType, LlmServiceId, AIServiceRegistry, AIServiceState } from "./registry.js";

describe("AIServiceRegistry", () => {
    let registry: AIServiceRegistry;

    beforeEach(async () => {
        vi.useFakeTimers();
        AIServiceRegistry["instance"] = undefined;
        await AIServiceRegistry.init();
        registry = AIServiceRegistry.get();
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
        registry.updateServiceState("Anthropic", AIServiceErrorType.Overloaded);
        expect(registry.getBestService(AnthropicModel.Opus3)).not.toBe(LlmServiceId.Anthropic);
    });

    it("returns the next active fallback service when preferred and first fallback are on cooldown", () => {
        registry.updateServiceState("Anthropic", AIServiceErrorType.Overloaded); // Set Anthropic to cooldown
        registry.updateServiceState("Mistral", AIServiceErrorType.Overloaded); // Set Mistral to cooldown

        const bestServiceId = registry.getBestService(AnthropicModel.Sonnet3_5);
        expect(bestServiceId).not.toBe(LlmServiceId.Anthropic);
        expect(bestServiceId).not.toBe(LlmServiceId.Mistral);
    });

    it("returns null when all services are on cooldown or disabled", () => {
        Object.values(LlmServiceId).forEach(serviceId => {
            registry.updateServiceState(serviceId, AIServiceErrorType.Overloaded);
        });

        expect(registry.getBestService(AnthropicModel.Opus3)).toBeNull();
    });

    it("registers and retrieves service states correctly", () => {
        registry.registerService("testService");
        registry.registerService("testService2");
        expect(registry.getServiceState("testService")).toBe(AIServiceState.Active);
        expect(registry.getServiceState("testService2")).toBe(AIServiceState.Active);
    });

    it("automatically registers and activates an unregistered service when its state is requested", () => {
        const serviceState = registry.getServiceState("autoRegisteredService");
        expect(serviceState).toBe(AIServiceState.Active);
    });

    it("handles critical error by disabling service", () => {
        registry.registerService("criticalErrorService");
        registry.registerService("fineService");
        registry.updateServiceState("criticalErrorService", AIServiceErrorType.Authentication);
        expect(registry.getServiceState("criticalErrorService")).toBe(AIServiceState.Disabled);
        expect(registry.getServiceState("fineService")).toBe(AIServiceState.Active);

        // Fast-forward to make sure it stays disabled
        vi.advanceTimersByTime(1_000_000);
        expect(registry.getServiceState("criticalErrorService")).toBe(AIServiceState.Disabled);
    });

    it("handles cooldown error by setting service into cooldown", () => {
        registry.registerService("cooldownService");
        registry.updateServiceState("cooldownService", AIServiceErrorType.Overloaded);
        expect(registry.getServiceState("cooldownService")).toBe(AIServiceState.Cooldown);
    });

    it("resets service from cooldown to active after cooldown period", async () => {
        registry.registerService("resetService");
        registry.updateServiceState("resetService", AIServiceErrorType.Overloaded);

        // Fast-forward a few seconds to make sure it's still in cooldown
        vi.advanceTimersByTime(10_000);
        expect(registry.getServiceState("resetService")).toBe(AIServiceState.Cooldown);

        // Fast-forward to the end of the cooldown period (15 minutes)
        vi.advanceTimersByTime(15 * 60 * 1000);
        expect(registry.getServiceState("resetService")).toBe(AIServiceState.Active);
    });
});
