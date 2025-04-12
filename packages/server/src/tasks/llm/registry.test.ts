import { AnthropicModel, MistralModel, OpenAIModel } from "@local/shared";
import { expect } from "chai";
import sinon from "sinon";
import { LlmServiceErrorType, LlmServiceId, LlmServiceRegistry, LlmServiceState } from "./registry.js";

describe("LlmServiceRegistry", () => {
    let registry: LlmServiceRegistry;
    const sandbox = sinon.createSandbox();
    let clock: sinon.SinonFakeTimers;

    beforeEach(async () => {
        clock = sandbox.useFakeTimers();
        LlmServiceRegistry["instance"] = undefined;
        await LlmServiceRegistry.init();
        registry = LlmServiceRegistry.get();
    });

    afterEach(() => {
        clock.runAll();
        sandbox.restore();
    });

    it("returns the preferred service when active", () => {
        expect(registry.getBestService(AnthropicModel.Opus3)).to.equal(LlmServiceId.Anthropic);
        expect(registry.getBestService(OpenAIModel.Gpt4o_Mini)).to.equal(LlmServiceId.OpenAI);
        expect(registry.getBestService(MistralModel.Codestral)).to.equal(LlmServiceId.Mistral);
    });

    it("returns the first active fallback service when preferred is on cooldown", () => {
        registry.updateServiceState("Anthropic", LlmServiceErrorType.Overloaded);
        expect(registry.getBestService(AnthropicModel.Opus3)).to.not.equal(LlmServiceId.Anthropic);
    });

    it("returns the next active fallback service when preferred and first fallback are on cooldown", () => {
        registry.updateServiceState("Anthropic", LlmServiceErrorType.Overloaded); // Set Anthropic to cooldown
        registry.updateServiceState("Mistral", LlmServiceErrorType.Overloaded); // Set Mistral to cooldown

        const bestServiceId = registry.getBestService(AnthropicModel.Sonnet3_5);
        expect(bestServiceId).to.not.equal(LlmServiceId.Anthropic);
        expect(bestServiceId).to.not.equal(LlmServiceId.Mistral);
    });

    it("returns null when all services are on cooldown or disabled", () => {
        Object.values(LlmServiceId).forEach(serviceId => {
            registry.updateServiceState(serviceId, LlmServiceErrorType.Overloaded);
        });

        expect(registry.getBestService(AnthropicModel.Opus3)).to.be.null;
    });

    it("registers and retrieves service states correctly", () => {
        registry.registerService("testService");
        registry.registerService("testService2");
        expect(registry.getServiceState("testService")).to.equal(LlmServiceState.Active);
        expect(registry.getServiceState("testService2")).to.equal(LlmServiceState.Active);
    });

    it("automatically registers and activates an unregistered service when its state is requested", () => {
        const serviceState = registry.getServiceState("autoRegisteredService");
        expect(serviceState).to.equal(LlmServiceState.Active);
    });

    it("handles critical error by disabling service", () => {
        registry.registerService("criticalErrorService");
        registry.registerService("fineService");
        registry.updateServiceState("criticalErrorService", LlmServiceErrorType.Authentication);
        expect(registry.getServiceState("criticalErrorService")).to.equal(LlmServiceState.Disabled);
        expect(registry.getServiceState("fineService")).to.equal(LlmServiceState.Active);

        // Fast-forward to make sure it stays disabled
        clock.tick(1_000_000);
        expect(registry.getServiceState("criticalErrorService")).to.equal(LlmServiceState.Disabled);
    });

    it("handles cooldown error by setting service into cooldown", () => {
        registry.registerService("cooldownService");
        registry.updateServiceState("cooldownService", LlmServiceErrorType.Overloaded);
        expect(registry.getServiceState("cooldownService")).to.equal(LlmServiceState.Cooldown);
    });

    it("resets service from cooldown to active after cooldown period", async () => {
        registry.registerService("resetService");
        registry.updateServiceState("resetService", LlmServiceErrorType.Overloaded);

        // Fast-forward a few seconds to make sure it's still in cooldown
        clock.tick(10_000);
        expect(registry.getServiceState("resetService")).to.equal(LlmServiceState.Cooldown);

        // Fast-forward to the end of the cooldown period (15 minutes)
        clock.tick(15 * 60 * 1000);
        expect(registry.getServiceState("resetService")).to.equal(LlmServiceState.Active);
    });
});
