import { expect, describe, it } from "vitest";
import winston from "winston";
import { SwarmState, type Swarm } from "@vrooli/shared";
import { SwarmStateMachine } from "../../tier1/coordination/swarmStateMachine.js";

describe("SwarmStateMachine Isolated Test", () => {
    it("should be able to import SwarmState from shared package", () => {
        expect(SwarmState.UNINITIALIZED).toBe("UNINITIALIZED");
    });

    it("should be able to create a mock swarm", () => {
        const mockSwarm: Swarm = {
            id: "swarm-123",
            state: SwarmState.UNINITIALIZED,
            config: {
                name: "Test Swarm",
                description: "Test description",
                goal: "Test goal",
                model: "gpt-4o-mini",
                temperature: 0.7,
                autoApproveTools: false,
                parallelExecutionLimit: 5,
            },
            team: {
                agents: [],
                capabilities: [],
                activeMembers: 0,
            },
            resources: {
                allocated: {
                    maxCredits: 10000,
                    maxTokens: 100000,
                    maxTime: 3600000,
                },
                consumed: {
                    credits: 0,
                    tokens: 0,
                    time: 0,
                },
                remaining: {
                    maxCredits: 10000,
                    maxTokens: 100000,
                    maxTime: 3600000,
                },
            },
        };

        expect(mockSwarm.id).toBe("swarm-123");
        expect(mockSwarm.state).toBe(SwarmState.UNINITIALIZED);
    });
});