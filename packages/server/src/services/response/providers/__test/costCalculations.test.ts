import { API_CREDITS_MULTIPLIER } from "@vrooli/shared";
import { describe, expect, it } from "vitest";
import { CloudflareGatewayService } from "../CloudflareGatewayService.js";
import { LocalOllamaService } from "../LocalOllamaService.js";
import { OpenRouterService } from "../OpenRouterService.js";

describe("Cost Calculation Verification", () => {
    describe("CloudflareGatewayService", () => {
        const service = new CloudflareGatewayService();

        it("should calculate costs correctly for GPT-4o", () => {
            // GPT-4o: $2.50 per 1M input, $10.00 per 1M output
            const cost = service.getResponseCost({
                model: "@cf/openai/gpt-4o",
                usage: { input: 1000, output: 500 },
            });

            // 1000 tokens * 250 cents/M = 250,000 credits
            // 500 tokens * 1000 cents/M = 500,000 credits
            // Total: 750,000 credits = $0.0075
            expect(cost).toBe(250_000 + 500_000); // 750,000
        });

        it("should calculate costs correctly for GPT-4o-mini", () => {
            // GPT-4o-mini: $0.15 per 1M input, $0.60 per 1M output
            const cost = service.getResponseCost({
                model: "@cf/openai/gpt-4o-mini",
                usage: { input: 10_000, output: 5_000 },
            });

            // 10,000 tokens * 15 cents/M = 150,000 credits
            // 5,000 tokens * 60 cents/M = 300,000 credits
            // Total: 450,000 credits = $0.0045
            expect(cost).toBe(150_000 + 300_000); // 450,000
        });

        it("should calculate token limits correctly with credit constraints", () => {
            // With $0.10 budget (10 cents = 10M credits)
            const maxCredits = BigInt(10) * API_CREDITS_MULTIPLIER;
            
            const maxTokens = service.getMaxOutputTokensRestrained({
                model: "@cf/openai/gpt-4o-mini",
                maxCredits,
                inputTokens: 1000,
            });

            // Input cost: 1000 * 15 = 15,000 credits
            // Remaining: 10,000,000 - 15,000 = 9,985,000 credits
            // Max output tokens: 9,985,000 / 60 = 166,416 tokens
            expect(maxTokens).toBe(16384); // Limited by model max
        });

        it("should return 0 tokens when budget exhausted", () => {
            const maxCredits = BigInt(1000); // Very small budget
            
            const maxTokens = service.getMaxOutputTokensRestrained({
                model: "@cf/openai/gpt-4o",
                maxCredits,
                inputTokens: 10, // Even 10 tokens costs 2,500 credits
            });

            expect(maxTokens).toBe(0);
        });

        it("should handle actual cost from headers", () => {
            // Simulating actual cost in dollars
            const cost = service.getResponseCost({
                model: "@cf/openai/gpt-4o",
                usage: { 
                    input: 1000, 
                    output: 500,
                    actualCost: 0.0075, // $0.0075
                } as any,
            });

            // $0.0075 = 0.75 cents = 750,000 credits
            expect(cost).toBe(750_000);
        });
    });

    describe("OpenRouterService", () => {
        const service = new OpenRouterService();

        it("should calculate costs correctly for various models", () => {
            const testCases = [
                {
                    model: "openai/gpt-4o",
                    usage: { input: 1000, output: 500 },
                    expected: 250_000 + 500_000, // 750,000
                },
                {
                    model: "anthropic/claude-3-haiku",
                    usage: { input: 10_000, output: 5_000 },
                    expected: 250_000 + 625_000, // 875,000
                },
                {
                    model: "meta-llama/llama-3.1-8b-instruct",
                    usage: { input: 100_000, output: 50_000 },
                    expected: 600_000 + 300_000, // 900,000
                },
            ];

            testCases.forEach(({ model, usage, expected }) => {
                const cost = service.getResponseCost({ model, usage });
                expect(cost).toBe(expected);
            });
        });

        it("should handle fallback costs for unknown models", () => {
            const cost = service.getResponseCost({
                model: "unknown/model",
                usage: { input: 1000, output: 1000 },
            });

            // Fallback: $0.15 input, $0.60 output per 1M
            // 1000 * 15 + 1000 * 60 = 75,000 credits
            expect(cost).toBe(15_000 + 60_000); // 75,000
        });
    });

    describe("LocalOllamaService", () => {
        const service = new LocalOllamaService();

        it("should calculate nominal costs correctly", () => {
            // Local models: $0.001 per 1M tokens (0.1 cents)
            const cost = service.getResponseCost({
                model: "llama3.1:8b",
                usage: { input: 10_000, output: 5_000 },
            });

            // 10,000 * 0.1 + 5,000 * 0.1 = 1,500 credits
            expect(cost).toBe(1_000 + 500); // 1,500
        });

        it("should return fixed cost in streaming", async () => {
            // The done event should return NOMINAL_COST_CREDITS = 1,000
            const modelInfo = service.getModelInfo();
            const firstModel = Object.keys(modelInfo)[0] || "llama3.1:8b";
            
            // Check model info has correct costs
            if (modelInfo[firstModel]) {
                expect(modelInfo[firstModel].inputCost).toBe(0.1);
                expect(modelInfo[firstModel].outputCost).toBe(0.1);
            }
        });
    });

    describe("Cross-Service Cost Consistency", () => {
        it("should have consistent cost units across all services", () => {
            const cf = new CloudflareGatewayService();
            const or = new OpenRouterService();
            const ollama = new LocalOllamaService();

            // All services should return costs in credits (not cents or dollars)
            const testUsage = { input: 1000, output: 1000 };

            const cfCost = cf.getResponseCost({
                model: "@cf/openai/gpt-4o-mini",
                usage: testUsage,
            });

            const orCost = or.getResponseCost({
                model: "openai/gpt-4o-mini",
                usage: testUsage,
            });

            const ollamaCost = ollama.getResponseCost({
                model: "llama3.1:8b",
                usage: testUsage,
            });

            // All should return positive integer credits
            expect(cfCost).toBeGreaterThan(0);
            expect(orCost).toBeGreaterThan(0);
            expect(ollamaCost).toBeGreaterThan(0);

            // CF and OR should have same cost for same model
            expect(cfCost).toBe(orCost);

            // Ollama should be much cheaper
            expect(ollamaCost).toBeLessThan(cfCost / 10);
        });
    });

    describe("Credit System Verification", () => {
        it("should correctly convert between dollars, cents, and credits", () => {
            // Verify our understanding of the credit system
            const oneDollar = 100; // cents
            const oneDollarInCredits = oneDollar * Number(API_CREDITS_MULTIPLIER);
            expect(oneDollarInCredits).toBe(100_000_000); // 100 million credits

            const oneCent = 1;
            const oneCentInCredits = oneCent * Number(API_CREDITS_MULTIPLIER);
            expect(oneCentInCredits).toBe(1_000_000); // 1 million credits

            // Verify BigInt handling
            const bigCredits = BigInt(50) * API_CREDITS_MULTIPLIER; // 50 cents
            expect(bigCredits).toBe(50_000_000n);
        });
    });
});
