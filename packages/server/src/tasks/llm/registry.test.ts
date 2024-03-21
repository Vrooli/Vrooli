import { AnthropicModel, LlmServiceErrorType, LlmServiceId, LlmServiceRegistry, LlmServiceState, OpenAIModel } from "./registry";

describe("LlmServiceRegistry", () => {
    let registry: LlmServiceRegistry;

    beforeEach(async () => {
        jest.useFakeTimers();
        LlmServiceRegistry["instance"] = undefined;
        await LlmServiceRegistry.init();
        registry = LlmServiceRegistry.get();
    });

    afterEach(() => {
        jest.runOnlyPendingTimers();
        jest.useRealTimers();
    });

    test("returns the preferred service when active", () => {
        expect(registry.getBestService(AnthropicModel.Opus)).toEqual(LlmServiceId.Anthropic);
        expect(registry.getBestService(OpenAIModel.Gpt3_5Turbo)).toEqual(LlmServiceId.OpenAI);
    });

    test("returns the first active fallback service when preferred is on cooldown", () => {
        registry.updateServiceState("Anthropic", LlmServiceErrorType.Overloaded);
        expect(registry.getBestService(AnthropicModel.Opus)).not.toEqual(LlmServiceId.Anthropic);
    });

    test("returns the next active fallback service when preferred and first fallback are on cooldown", () => {
        registry.updateServiceState("Anthropic", LlmServiceErrorType.Overloaded); // Set Anthropic to cooldown
        registry.updateServiceState("Mistral", LlmServiceErrorType.Overloaded); // Set Mistral to cooldown

        const bestServiceId = registry.getBestService(AnthropicModel.Sonnet);
        expect(bestServiceId).not.toEqual(LlmServiceId.Anthropic);
        expect(bestServiceId).not.toEqual(LlmServiceId.Mistral);
    });

    test("returns null when all services are on cooldown or disabled", () => {
        Object.values(LlmServiceId).forEach(serviceId => {
            registry.updateServiceState(serviceId, LlmServiceErrorType.Overloaded);
        });

        expect(registry.getBestService(AnthropicModel.Opus)).toBeNull();
    });

    test("registers and retrieves service states correctly", () => {
        registry.registerService("testService");
        registry.registerService("testService2");
        expect(registry.getServiceState("testService")).toEqual(LlmServiceState.Active);
        expect(registry.getServiceState("testService2")).toEqual(LlmServiceState.Active);
    });

    test("automatically registers and activates an unregistered service when its state is requested", () => {
        const serviceState = registry.getServiceState("autoRegisteredService");
        expect(serviceState).toEqual(LlmServiceState.Active);
    });

    test("handles critical error by disabling service", () => {
        registry.registerService("criticalErrorService");
        registry.registerService("fineService");
        registry.updateServiceState("criticalErrorService", LlmServiceErrorType.Authentication);
        expect(registry.getServiceState("criticalErrorService")).toEqual(LlmServiceState.Disabled);
        expect(registry.getServiceState("fineService")).toEqual(LlmServiceState.Active);

        // Fast-forward to make sure it stays disabled
        jest.advanceTimersByTime(1_000_000);
        expect(registry.getServiceState("criticalErrorService")).toEqual(LlmServiceState.Disabled);
    });

    test("handles cooldown error by setting service into cooldown", () => {
        registry.registerService("cooldownService");
        registry.updateServiceState("cooldownService", LlmServiceErrorType.Overloaded);
        expect(registry.getServiceState("cooldownService")).toEqual(LlmServiceState.Cooldown);
    });

    test("resets service from cooldown to active after cooldown period", async () => {
        registry.registerService("resetService");
        registry.updateServiceState("resetService", LlmServiceErrorType.Overloaded);

        // Fast-forward a few seconds to make sure it's still in cooldown
        jest.advanceTimersByTime(10_000);
        expect(registry.getServiceState("resetService")).toEqual(LlmServiceState.Cooldown);

        // Fast-forward to the end of the cooldown period (15 minutes)
        jest.advanceTimersByTime(15 * 60 * 1000);
        expect(registry.getServiceState("resetService")).toEqual(LlmServiceState.Active);
    });
});
