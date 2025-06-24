import { describe, it, expect, beforeAll, vi } from "vitest";
import { TokenEstimationRegistry, type DefaultTokenEstimator } from "./tokens.js";
import { TokenEstimatorType } from "./types/tokenTypes.js";

describe("Token Estimation System", () => {
    describe("DefaultTokenEstimator", () => {
        let registry: TokenEstimationRegistry;

        beforeAll(() => {
            registry = TokenEstimationRegistry.get();
        });

        describe("Basic token estimation", () => {
            it("should estimate tokens for basic English text", () => {
                const result = registry.estimateTokens(TokenEstimatorType.Default, {
                    aiModel: "gpt-4",
                    text: "Hello world",
                });

                expect(result.tokens).toBe(6); // "Hello world" is 11 bytes, ceil(11/2) = 6
                expect(result.estimationModel).toBe(TokenEstimatorType.Default);
                expect(result.encoding).toBe("default");
            });

            it("should handle empty string", () => {
                const result = registry.estimateTokens(TokenEstimatorType.Default, {
                    aiModel: "gpt-4",
                    text: "",
                });

                expect(result.tokens).toBe(0);
                expect(result.estimationModel).toBe(TokenEstimatorType.Default);
            });

            it("should handle single character", () => {
                const result = registry.estimateTokens(TokenEstimatorType.Default, {
                    aiModel: "gpt-4", 
                    text: "a",
                });

                expect(result.tokens).toBe(1); // 1 byte, ceil(1/2) = 1
            });

            it("should handle two characters", () => {
                const result = registry.estimateTokens(TokenEstimatorType.Default, {
                    aiModel: "gpt-4",
                    text: "ab",
                });

                expect(result.tokens).toBe(1); // 2 bytes, ceil(2/2) = 1
            });

            it("should handle three characters", () => {
                const result = registry.estimateTokens(TokenEstimatorType.Default, {
                    aiModel: "gpt-4",
                    text: "abc",
                });

                expect(result.tokens).toBe(2); // 3 bytes, ceil(3/2) = 2
            });
        });

        describe("Unicode handling", () => {
            it("should handle Unicode emoji correctly", () => {
                const result = registry.estimateTokens(TokenEstimatorType.Default, {
                    aiModel: "gpt-4",
                    text: "👋",
                });

                // Emoji are typically 4 bytes in UTF-8
                expect(result.tokens).toBe(2); // 4 bytes, ceil(4/2) = 2
            });

            it("should handle mixed Unicode and ASCII", () => {
                const result = registry.estimateTokens(TokenEstimatorType.Default, {
                    aiModel: "gpt-4",
                    text: "Hello 👋 world",
                });

                // "Hello " (6) + "👋" (4) + " world" (6) = 16 bytes
                expect(result.tokens).toBe(8); // ceil(16/2) = 8
            });

            it("should handle Chinese characters", () => {
                const result = registry.estimateTokens(TokenEstimatorType.Default, {
                    aiModel: "gpt-4",
                    text: "你好",
                });

                // Chinese characters are typically 3 bytes each in UTF-8
                expect(result.tokens).toBe(3); // 6 bytes, ceil(6/2) = 3
            });

            it("should handle combining characters", () => {
                const result = registry.estimateTokens(TokenEstimatorType.Default, {
                    aiModel: "gpt-4",
                    text: "é", // e + combining acute accent
                });

                // This could be either a single é (2 bytes) or e + combining (3 bytes)
                expect(result.tokens).toBeGreaterThanOrEqual(1);
                expect(result.tokens).toBeLessThanOrEqual(2);
            });
        });

        describe("Long text handling", () => {
            it("should handle long text efficiently", () => {
                const longText = "This is a test sentence. ".repeat(100);
                const result = registry.estimateTokens(TokenEstimatorType.Default, {
                    aiModel: "gpt-4",
                    text: longText,
                });

                expect(result.tokens).toBeGreaterThan(0);
                expect(result.estimationModel).toBe(TokenEstimatorType.Default);
            });

            it("should be consistent with repeated calls", () => {
                const text = "Consistency test";
                
                const result1 = registry.estimateTokens(TokenEstimatorType.Default, {
                    aiModel: "gpt-4",
                    text,
                });

                const result2 = registry.estimateTokens(TokenEstimatorType.Default, {
                    aiModel: "gpt-4", 
                    text,
                });

                expect(result1.tokens).toBe(result2.tokens);
                expect(result1.estimationModel).toBe(result2.estimationModel);
                expect(result1.encoding).toBe(result2.encoding);
            });
        });

        describe("Edge cases", () => {
            it("should handle whitespace correctly", () => {
                const result = registry.estimateTokens(TokenEstimatorType.Default, {
                    aiModel: "gpt-4",
                    text: "   \n\t   ",
                });

                expect(result.tokens).toBeGreaterThan(0);
            });

            it("should handle special characters", () => {
                const result = registry.estimateTokens(TokenEstimatorType.Default, {
                    aiModel: "gpt-4",
                    text: "!@#$%^&*()_+-=[]{}|;:,.<>?",
                });

                expect(result.tokens).toBeGreaterThan(0);
            });

            it("should handle newlines and tabs", () => {
                const result = registry.estimateTokens(TokenEstimatorType.Default, {
                    aiModel: "gpt-4",
                    text: "Line 1\nLine 2\tTabbed",
                });

                expect(result.tokens).toBeGreaterThan(0);
            });
        });
    });

    describe("TokenEstimationRegistry", () => {
        it("should be a singleton", () => {
            const registry1 = TokenEstimationRegistry.get();
            const registry2 = TokenEstimationRegistry.get();

            expect(registry1).toBe(registry2);
        });

        it("should fall back to default estimator for uninitialized estimators", () => {
            const registry = TokenEstimationRegistry.get();
            
            // Request Tiktoken but it may not be initialized
            const result = registry.estimateTokens(TokenEstimatorType.Tiktoken, {
                aiModel: "gpt-4",
                text: "Hello world",
            });

            // Should fall back to default if Tiktoken isn't initialized
            expect(result.estimationModel).toBeDefined();
            expect(result.tokens).toBeGreaterThan(0);
        });

        it("should return estimation info without estimating tokens", () => {
            const registry = TokenEstimationRegistry.get();
            
            const info = registry.getEstimationInfo(TokenEstimatorType.Default);

            expect(info.estimationModel).toBe(TokenEstimatorType.Default);
            expect(info.encoding).toBe("default");
        });

        it("should handle different AI models consistently", () => {
            const registry = TokenEstimationRegistry.get();
            const text = "Test text";

            const result1 = registry.estimateTokens(TokenEstimatorType.Default, {
                aiModel: "gpt-4",
                text,
            });

            const result2 = registry.estimateTokens(TokenEstimatorType.Default, {
                aiModel: "claude-3",
                text,
            });

            // Default estimator should give same results regardless of model
            expect(result1.tokens).toBe(result2.tokens);
            expect(result1.estimationModel).toBe(result2.estimationModel);
        });
    });

    describe("TiktokenWasmEstimator integration", () => {
        it("should handle uninitialized Tiktoken gracefully", () => {
            const registry = TokenEstimationRegistry.get();
            
            // This should fall back to default since Tiktoken may not be initialized
            const result = registry.estimateTokens(TokenEstimatorType.Tiktoken, {
                aiModel: "gpt-4",
                text: "Hello world",
            });

            expect(result.tokens).toBeGreaterThan(0);
            expect(result.estimationModel).toBeDefined();
            expect(result.encoding).toBeDefined();
        });
    });

    describe("Mathematical properties", () => {
        it("should satisfy basic mathematical properties", () => {
            const registry = TokenEstimationRegistry.get();
            
            const shortText = "Hi";
            const longText = "Hi there, this is a much longer text";

            const shortResult = registry.estimateTokens(TokenEstimatorType.Default, {
                aiModel: "gpt-4",
                text: shortText,
            });

            const longResult = registry.estimateTokens(TokenEstimatorType.Default, {
                aiModel: "gpt-4",
                text: longText,
            });

            // Longer text should have more tokens
            expect(longResult.tokens).toBeGreaterThan(shortResult.tokens);
        });

        it("should give reasonable estimates for typical use cases", () => {
            const registry = TokenEstimationRegistry.get();
            
            const typicalSentence = "This is a typical English sentence with some common words.";
            const result = registry.estimateTokens(TokenEstimatorType.Default, {
                aiModel: "gpt-4",
                text: typicalSentence,
            });

            // Should be reasonable - not too high or too low
            expect(result.tokens).toBeGreaterThan(10);
            expect(result.tokens).toBeLessThan(100);
        });
    });
});
