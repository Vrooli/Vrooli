import { describe, it, expect, beforeEach } from "vitest";
import { McpToolName, McpSwarmToolName } from "@vrooli/shared";
import {
    ToolApprovalConfig,
    ToolRiskLevel,
    DEFAULT_APPROVAL_POLICY,
    toolApprovalConfig,
    type ToolApprovalPolicy,
} from "./toolApprovalConfig.js";

describe("ToolApprovalConfig", () => {
    let config: ToolApprovalConfig;

    beforeEach(() => {
        config = new ToolApprovalConfig();
    });

    describe("constructor", () => {
        it("should use default policy when no config provided", () => {
            const defaultConfig = new ToolApprovalConfig();
            
            expect(defaultConfig.requiresApproval("unknown_tool")).toBe(DEFAULT_APPROVAL_POLICY.defaultRequiresApproval);
        });

        it("should merge provided policy with defaults", () => {
            const customPolicy: Partial<ToolApprovalPolicy> = {
                defaultRequiresApproval: false,
                riskThreshold: ToolRiskLevel.HIGH,
                maxAutoApprovalCredits: BigInt(500),
                toolOverrides: new Map([["custom_tool", false]]),
                trustedBots: new Set(["trusted_bot_1"]),
            };

            const customConfig = new ToolApprovalConfig(customPolicy);

            expect(customConfig.requiresApproval("unknown_tool")).toBe(false);
            expect(customConfig.requiresApproval("custom_tool")).toBe(false);
            expect(customConfig.requiresApproval("test", "trusted_bot_1")).toBe(false);
        });

        it("should preserve existing tool overrides when adding new ones", () => {
            const initialOverrides = new Map([["tool1", true]]);
            const additionalOverrides = new Map([["tool2", false]]);

            const config1 = new ToolApprovalConfig({
                toolOverrides: initialOverrides,
            });

            const config2 = new ToolApprovalConfig({
                toolOverrides: additionalOverrides,
            });

            // Both configs should have their respective overrides
            expect(config1.requiresApproval("tool1")).toBe(true);
            expect(config2.requiresApproval("tool2")).toBe(false);
        });

        it("should preserve existing trusted bots when adding new ones", () => {
            const initialBots = new Set(["bot1"]);
            const additionalBots = new Set(["bot2"]);

            const config1 = new ToolApprovalConfig({
                trustedBots: initialBots,
            });

            const config2 = new ToolApprovalConfig({
                trustedBots: additionalBots,
            });

            expect(config1.requiresApproval("any_tool", "bot1")).toBe(false);
            expect(config2.requiresApproval("any_tool", "bot2")).toBe(false);
        });
    });

    describe("requiresApproval", () => {
        describe("tool-specific overrides", () => {
            it("should respect tool override requiring approval", () => {
                const policy: Partial<ToolApprovalPolicy> = {
                    defaultRequiresApproval: false,
                    toolOverrides: new Map([["force_approve", true]]),
                };
                const configWithOverride = new ToolApprovalConfig(policy);

                expect(configWithOverride.requiresApproval("force_approve")).toBe(true);
            });

            it("should respect tool override not requiring approval", () => {
                const policy: Partial<ToolApprovalPolicy> = {
                    defaultRequiresApproval: true,
                    toolOverrides: new Map([["allow_auto", false]]),
                };
                const configWithOverride = new ToolApprovalConfig(policy);

                expect(configWithOverride.requiresApproval("allow_auto")).toBe(false);
            });

            it("should prioritize tool overrides over risk levels", () => {
                const policy: Partial<ToolApprovalPolicy> = {
                    riskThreshold: ToolRiskLevel.LOW,
                    toolOverrides: new Map([[McpToolName.RunRoutine, false]]), // RunRoutine is HIGH risk
                };
                const configWithOverride = new ToolApprovalConfig(policy);

                // Should not require approval despite HIGH risk due to override
                expect(configWithOverride.requiresApproval(McpToolName.RunRoutine)).toBe(false);
            });
        });

        describe("trusted bots", () => {
            beforeEach(() => {
                config = new ToolApprovalConfig({
                    trustedBots: new Set(["trusted_bot"]),
                    maxAutoApprovalCredits: BigInt(100),
                });
            });

            it("should auto-approve for trusted bots", () => {
                expect(config.requiresApproval("any_tool", "trusted_bot")).toBe(false);
                expect(config.requiresApproval(McpToolName.RunRoutine, "trusted_bot")).toBe(false);
            });

            it("should require approval for untrusted bots", () => {
                expect(config.requiresApproval("any_tool", "untrusted_bot")).toBe(true);
            });

            it("should require approval for trusted bots exceeding credit limit", () => {
                const highCostCredits = BigInt(200); // Exceeds maxAutoApprovalCredits of 100

                expect(config.requiresApproval("expensive_tool", "trusted_bot", highCostCredits)).toBe(true);
            });

            it("should auto-approve for trusted bots within credit limit", () => {
                const lowCostCredits = BigInt(50); // Within maxAutoApprovalCredits of 100

                expect(config.requiresApproval("cheap_tool", "trusted_bot", lowCostCredits)).toBe(false);
            });

            it("should handle missing callerBotId gracefully", () => {
                expect(config.requiresApproval("any_tool")).toBe(true);
                expect(config.requiresApproval("any_tool", undefined)).toBe(true);
                expect(config.requiresApproval("any_tool", "")).toBe(true);
            });
        });

        describe("risk-based approval", () => {
            it("should require approval for tools at or above risk threshold", () => {
                const lowThresholdConfig = new ToolApprovalConfig({
                    riskThreshold: ToolRiskLevel.LOW,
                });

                expect(lowThresholdConfig.requiresApproval(McpToolName.DefineTool)).toBe(false); // NONE
                expect(lowThresholdConfig.requiresApproval(McpToolName.SendMessage)).toBe(true); // LOW
                expect(lowThresholdConfig.requiresApproval(McpToolName.ResourceManage)).toBe(true); // MEDIUM
                expect(lowThresholdConfig.requiresApproval(McpToolName.RunRoutine)).toBe(true); // HIGH
            });

            it("should not require approval for tools below risk threshold", () => {
                const highThresholdConfig = new ToolApprovalConfig({
                    riskThreshold: ToolRiskLevel.HIGH,
                });

                expect(highThresholdConfig.requiresApproval(McpToolName.DefineTool)).toBe(false); // NONE
                expect(highThresholdConfig.requiresApproval(McpToolName.SendMessage)).toBe(false); // LOW
                expect(highThresholdConfig.requiresApproval(McpToolName.ResourceManage)).toBe(false); // MEDIUM
                expect(highThresholdConfig.requiresApproval(McpToolName.RunRoutine)).toBe(true); // HIGH
                expect(highThresholdConfig.requiresApproval("code_interpreter")).toBe(true); // CRITICAL
            });

            it("should handle edge case risk thresholds", () => {
                const noneThresholdConfig = new ToolApprovalConfig({
                    riskThreshold: ToolRiskLevel.NONE,
                });

                // Everything should require approval with NONE threshold
                expect(noneThresholdConfig.requiresApproval(McpToolName.DefineTool)).toBe(true);

                const criticalThresholdConfig = new ToolApprovalConfig({
                    riskThreshold: ToolRiskLevel.CRITICAL,
                });

                // Only CRITICAL should require approval
                expect(criticalThresholdConfig.requiresApproval(McpToolName.RunRoutine)).toBe(false); // HIGH
                expect(criticalThresholdConfig.requiresApproval("code_interpreter")).toBe(true); // CRITICAL
            });
        });

        describe("default policy fallback", () => {
            it("should use default policy for unknown tools", () => {
                const requiresApprovalConfig = new ToolApprovalConfig({
                    defaultRequiresApproval: true,
                });

                const noApprovalConfig = new ToolApprovalConfig({
                    defaultRequiresApproval: false,
                });

                expect(requiresApprovalConfig.requiresApproval("unknown_tool")).toBe(true);
                expect(noApprovalConfig.requiresApproval("unknown_tool")).toBe(false);
            });

            it("should handle empty tool names", () => {
                expect(config.requiresApproval("")).toBe(true);
                expect(config.requiresApproval("   ")).toBe(true);
            });
        });

        describe("complex scenarios", () => {
            it("should handle combination of trusted bot with tool override", () => {
                const complexConfig = new ToolApprovalConfig({
                    trustedBots: new Set(["trusted_bot"]),
                    toolOverrides: new Map([["restricted_tool", true]]),
                });

                // Tool override should take precedence over trusted bot status
                expect(complexConfig.requiresApproval("restricted_tool", "trusted_bot")).toBe(true);
            });

            it("should handle trusted bot with high-risk tool and credit limit", () => {
                const complexConfig = new ToolApprovalConfig({
                    trustedBots: new Set(["trusted_bot"]),
                    maxAutoApprovalCredits: BigInt(50),
                    riskThreshold: ToolRiskLevel.LOW,
                });

                // Should auto-approve despite high risk due to trusted status, but within credit limit
                expect(complexConfig.requiresApproval(McpToolName.RunRoutine, "trusted_bot", BigInt(30))).toBe(false);

                // Should require approval due to exceeding credit limit
                expect(complexConfig.requiresApproval(McpToolName.RunRoutine, "trusted_bot", BigInt(100))).toBe(true);
            });
        });

        describe("MCP tool risk levels", () => {
            it("should correctly classify MCP tool risks", () => {
                const lowThresholdConfig = new ToolApprovalConfig({
                    riskThreshold: ToolRiskLevel.LOW,
                });

                // Verify known MCP tool classifications
                expect(lowThresholdConfig.requiresApproval(McpToolName.DefineTool)).toBe(false); // NONE
                expect(lowThresholdConfig.requiresApproval(McpToolName.SendMessage)).toBe(true); // LOW
                expect(lowThresholdConfig.requiresApproval(McpToolName.ResourceManage)).toBe(true); // MEDIUM
                expect(lowThresholdConfig.requiresApproval(McpToolName.RunRoutine)).toBe(true); // HIGH
                expect(lowThresholdConfig.requiresApproval(McpToolName.SpawnSwarm)).toBe(true); // HIGH
            });

            it("should correctly classify MCP swarm tool risks", () => {
                const mediumThresholdConfig = new ToolApprovalConfig({
                    riskThreshold: ToolRiskLevel.MEDIUM,
                });

                expect(mediumThresholdConfig.requiresApproval(McpSwarmToolName.UpdateSwarmSharedState)).toBe(false); // LOW
                expect(mediumThresholdConfig.requiresApproval(McpSwarmToolName.EndSwarm)).toBe(true); // MEDIUM
            });
        });

        describe("OpenAI tool risk levels", () => {
            it("should correctly classify OpenAI tool risks", () => {
                const mediumThresholdConfig = new ToolApprovalConfig({
                    riskThreshold: ToolRiskLevel.MEDIUM,
                });

                expect(mediumThresholdConfig.requiresApproval("web_search")).toBe(false); // LOW
                expect(mediumThresholdConfig.requiresApproval("file_search")).toBe(false); // LOW
                expect(mediumThresholdConfig.requiresApproval("image_generation")).toBe(true); // MEDIUM
                expect(mediumThresholdConfig.requiresApproval("code_interpreter")).toBe(true); // CRITICAL
                expect(mediumThresholdConfig.requiresApproval("computer-preview")).toBe(true); // CRITICAL
            });
        });
    });

    describe("getRiskLevel", () => {
        it("should return correct risk levels for known tools", () => {
            expect(config.getRiskLevel(McpToolName.DefineTool)).toBe(ToolRiskLevel.NONE);
            expect(config.getRiskLevel(McpToolName.SendMessage)).toBe(ToolRiskLevel.LOW);
            expect(config.getRiskLevel(McpToolName.ResourceManage)).toBe(ToolRiskLevel.MEDIUM);
            expect(config.getRiskLevel(McpToolName.RunRoutine)).toBe(ToolRiskLevel.HIGH);
            expect(config.getRiskLevel("code_interpreter")).toBe(ToolRiskLevel.CRITICAL);
        });

        it("should return MEDIUM risk for unknown tools", () => {
            expect(config.getRiskLevel("unknown_tool")).toBe(ToolRiskLevel.MEDIUM);
            expect(config.getRiskLevel("")).toBe(ToolRiskLevel.MEDIUM);
            expect(config.getRiskLevel("completely_new_tool")).toBe(ToolRiskLevel.MEDIUM);
        });
    });

    describe("updatePolicy", () => {
        beforeEach(() => {
            config = new ToolApprovalConfig({
                defaultRequiresApproval: true,
                riskThreshold: ToolRiskLevel.MEDIUM,
                maxAutoApprovalCredits: BigInt(100),
                toolOverrides: new Map([["initial_tool", false]]),
                trustedBots: new Set(["initial_bot"]),
            });
        });

        it("should update defaultRequiresApproval", () => {
            config.updatePolicy({ defaultRequiresApproval: false });
            expect(config.requiresApproval("unknown_tool")).toBe(false);
        });

        it("should update riskThreshold", () => {
            // Initially MEDIUM threshold should allow LOW risk tools
            expect(config.requiresApproval(McpToolName.SendMessage)).toBe(false);

            config.updatePolicy({ riskThreshold: ToolRiskLevel.LOW });

            // Now LOW risk tools should require approval
            expect(config.requiresApproval(McpToolName.SendMessage)).toBe(true);
        });

        it("should update maxAutoApprovalCredits", () => {
            config.updatePolicy({
                trustedBots: new Set(["test_bot"]),
            });

            // Initially should auto-approve within credit limit
            expect(config.requiresApproval("any_tool", "test_bot", BigInt(50))).toBe(false);

            config.updatePolicy({ maxAutoApprovalCredits: BigInt(25) });

            // Now should require approval due to lower credit limit
            expect(config.requiresApproval("any_tool", "test_bot", BigInt(50))).toBe(true);
        });

        it("should add new tool overrides", () => {
            config.updatePolicy({
                toolOverrides: new Map([["new_tool", true]]),
            });

            expect(config.requiresApproval("new_tool")).toBe(true);
            // Original override should still exist
            expect(config.requiresApproval("initial_tool")).toBe(false);
        });

        it("should add new trusted bots", () => {
            config.updatePolicy({
                trustedBots: new Set(["new_bot"]),
            });

            expect(config.requiresApproval("any_tool", "new_bot")).toBe(false);
            // Original trusted bot should still exist
            expect(config.requiresApproval("any_tool", "initial_bot")).toBe(false);
        });

        it("should handle multiple updates in single call", () => {
            config.updatePolicy({
                defaultRequiresApproval: false,
                riskThreshold: ToolRiskLevel.HIGH,
                maxAutoApprovalCredits: BigInt(200),
                toolOverrides: new Map([["multi_tool", true]]),
                trustedBots: new Set(["multi_bot"]),
            });

            expect(config.requiresApproval("unknown_tool")).toBe(false);
            expect(config.requiresApproval(McpToolName.ResourceManage)).toBe(false); // MEDIUM < HIGH
            expect(config.requiresApproval("multi_tool")).toBe(true);
            expect(config.requiresApproval("any_tool", "multi_bot")).toBe(false);
        });

        it("should ignore undefined values", () => {
            const originalDefaultRequiresApproval = config.requiresApproval("unknown_tool");

            config.updatePolicy({
                defaultRequiresApproval: undefined,
                riskThreshold: undefined,
                maxAutoApprovalCredits: undefined,
                toolOverrides: undefined,
                trustedBots: undefined,
            });

            // Should remain unchanged
            expect(config.requiresApproval("unknown_tool")).toBe(originalDefaultRequiresApproval);
        });
    });

    describe("risk level comparison", () => {
        it("should correctly order risk levels", () => {
            const noneConfig = new ToolApprovalConfig({ riskThreshold: ToolRiskLevel.NONE });
            const lowConfig = new ToolApprovalConfig({ riskThreshold: ToolRiskLevel.LOW });
            const mediumConfig = new ToolApprovalConfig({ riskThreshold: ToolRiskLevel.MEDIUM });
            const highConfig = new ToolApprovalConfig({ riskThreshold: ToolRiskLevel.HIGH });
            const criticalConfig = new ToolApprovalConfig({ riskThreshold: ToolRiskLevel.CRITICAL });

            // Test with a MEDIUM risk tool
            const mediumRiskTool = McpToolName.ResourceManage;

            expect(noneConfig.requiresApproval(mediumRiskTool)).toBe(true); // MEDIUM >= NONE
            expect(lowConfig.requiresApproval(mediumRiskTool)).toBe(true); // MEDIUM >= LOW
            expect(mediumConfig.requiresApproval(mediumRiskTool)).toBe(true); // MEDIUM >= MEDIUM
            expect(highConfig.requiresApproval(mediumRiskTool)).toBe(false); // MEDIUM < HIGH
            expect(criticalConfig.requiresApproval(mediumRiskTool)).toBe(false); // MEDIUM < CRITICAL
        });
    });

    describe("global instance", () => {
        it("should provide a default global instance", () => {
            expect(toolApprovalConfig).toBeInstanceOf(ToolApprovalConfig);
            expect(typeof toolApprovalConfig.requiresApproval).toBe("function");
        });

        it("should use default approval policy in global instance", () => {
            // Global instance should use DEFAULT_APPROVAL_POLICY
            expect(toolApprovalConfig.requiresApproval("unknown_tool")).toBe(DEFAULT_APPROVAL_POLICY.defaultRequiresApproval);
        });
    });

    describe("edge cases and error handling", () => {
        it("should handle null and undefined inputs gracefully", () => {
            expect(() => config.requiresApproval(null as any)).not.toThrow();
            expect(() => config.requiresApproval(undefined as any)).not.toThrow();
            expect(() => config.getRiskLevel(null as any)).not.toThrow();
            expect(() => config.getRiskLevel(undefined as any)).not.toThrow();
        });

        it("should handle very large credit values", () => {
            const trustedConfig = new ToolApprovalConfig({
                trustedBots: new Set(["trusted"]),
                maxAutoApprovalCredits: BigInt("999999999999999999"), // Very large limit
            });

            expect(trustedConfig.requiresApproval("tool", "trusted", BigInt("999999999999999998"))).toBe(false);
            expect(trustedConfig.requiresApproval("tool", "trusted", BigInt("1000000000000000000"))).toBe(true);
        });

        it("should handle zero credit values", () => {
            const trustedConfig = new ToolApprovalConfig({
                trustedBots: new Set(["trusted"]),
                maxAutoApprovalCredits: BigInt(0),
            });

            expect(trustedConfig.requiresApproval("tool", "trusted", BigInt(0))).toBe(false);
            expect(trustedConfig.requiresApproval("tool", "trusted", BigInt(1))).toBe(true);
        });

        it("should handle negative credit values", () => {
            const trustedConfig = new ToolApprovalConfig({
                trustedBots: new Set(["trusted"]),
                maxAutoApprovalCredits: BigInt(100),
            });

            // Negative credits should always be auto-approved for trusted bots
            expect(trustedConfig.requiresApproval("tool", "trusted", BigInt(-1))).toBe(false);
        });

        it("should handle whitespace in bot IDs and tool names", () => {
            const spaceConfig = new ToolApprovalConfig({
                trustedBots: new Set([" trusted_bot ", "trusted_bot"]),
                toolOverrides: new Map([[" tool_name ", false], ["tool_name", true]]),
            });

            // Exact matches required - whitespace matters
            expect(spaceConfig.requiresApproval("any", " trusted_bot ")).toBe(false);
            expect(spaceConfig.requiresApproval("any", "trusted_bot")).toBe(false);
            expect(spaceConfig.requiresApproval(" tool_name ")).toBe(false);
            expect(spaceConfig.requiresApproval("tool_name")).toBe(true);
        });

        it("should handle case sensitivity", () => {
            const caseConfig = new ToolApprovalConfig({
                trustedBots: new Set(["TrustedBot"]),
                toolOverrides: new Map([["ToolName", false]]),
            });

            // Should be case-sensitive
            expect(caseConfig.requiresApproval("any", "trustedbot")).toBe(true);
            expect(caseConfig.requiresApproval("any", "TrustedBot")).toBe(false);
            expect(caseConfig.requiresApproval("toolname")).toBe(true);
            expect(caseConfig.requiresApproval("ToolName")).toBe(false);
        });

        it("should handle empty Sets and Maps", () => {
            const emptyConfig = new ToolApprovalConfig({
                toolOverrides: new Map(),
                trustedBots: new Set(),
            });

            expect(emptyConfig.requiresApproval("any_tool", "any_bot")).toBe(true);
        });
    });

    describe("security considerations", () => {
        it("should not allow trusted bots to bypass tool overrides", () => {
            const securityConfig = new ToolApprovalConfig({
                trustedBots: new Set(["trusted_bot"]),
                toolOverrides: new Map([["dangerous_tool", true]]),
            });

            // Even trusted bots should not bypass explicit tool overrides
            expect(securityConfig.requiresApproval("dangerous_tool", "trusted_bot")).toBe(true);
        });

        it("should enforce credit limits even for low-risk tools", () => {
            const creditConfig = new ToolApprovalConfig({
                trustedBots: new Set(["trusted_bot"]),
                maxAutoApprovalCredits: BigInt(10),
                riskThreshold: ToolRiskLevel.HIGH, // Allow most tools
            });

            // Should require approval due to credit limit, not risk level
            expect(creditConfig.requiresApproval(McpToolName.DefineTool, "trusted_bot", BigInt(20))).toBe(true);
        });

        it("should default to requiring approval for security", () => {
            // Default should be secure (require approval)
            expect(DEFAULT_APPROVAL_POLICY.defaultRequiresApproval).toBe(true);
            expect(DEFAULT_APPROVAL_POLICY.riskThreshold).toBe(ToolRiskLevel.MEDIUM);
        });
    });
});
