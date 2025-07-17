/**
 * Agent Behavior Mocker
 * 
 * Manages mock behaviors for agent testing
 */

export interface AgentMockBehavior {
    routineCalls?: string[];
    eventEmissions?: string[];
    blackboardUpdates?: Record<string, unknown>;
}

export interface AgentExecutionLogEntry {
    type: "routine_call" | "event_emission" | "blackboard_update";
    routineLabel?: string;
    topic?: string;
    key?: string;
    value?: unknown;
    expectedValue?: unknown;
    timestamp: Date;
    allowed: boolean;
}

export class AgentBehaviorMocker {
    private behaviorRegistry = new Map<string, AgentMockBehavior>();
    private executionLog = new Map<string, AgentExecutionLogEntry[]>();

    async registerAgent(agentId: string, behavior: AgentMockBehavior): Promise<void> {
        this.behaviorRegistry.set(agentId, behavior);
        this.executionLog.set(agentId, []);
    }

    async expectRoutineCall(agentId: string, routineLabel: string): Promise<boolean> {
        const behavior = this.behaviorRegistry.get(agentId);
        if (!behavior?.routineCalls) return true;

        const isExpected = behavior.routineCalls.includes(routineLabel);

        // Log the call
        const log = this.executionLog.get(agentId) || [];
        log.push({
            type: "routine_call",
            routineLabel,
            timestamp: new Date(),
            allowed: isExpected,
        });
        this.executionLog.set(agentId, log);

        return isExpected;
    }

    async expectEventEmission(agentId: string, topic: string): Promise<boolean> {
        const behavior = this.behaviorRegistry.get(agentId);
        if (!behavior?.eventEmissions) return true;

        const isExpected = behavior.eventEmissions.includes(topic);

        // Log the emission
        const log = this.executionLog.get(agentId) || [];
        log.push({
            type: "event_emission",
            topic,
            timestamp: new Date(),
            allowed: isExpected,
        });
        this.executionLog.set(agentId, log);

        return isExpected;
    }

    async expectBlackboardUpdate(
        agentId: string,
        key: string,
        value: unknown,
    ): Promise<boolean> {
        const behavior = this.behaviorRegistry.get(agentId);
        if (!behavior?.blackboardUpdates) return true;

        const expectedValue = behavior.blackboardUpdates[key];
        const isExpected = expectedValue !== undefined;

        // Log the update
        const log = this.executionLog.get(agentId) || [];
        log.push({
            type: "blackboard_update",
            key,
            value,
            expectedValue,
            timestamp: new Date(),
            allowed: isExpected,
        });
        this.executionLog.set(agentId, log);

        return isExpected;
    }

    getExecutionLog(agentId: string): AgentExecutionLogEntry[] {
        return this.executionLog.get(agentId) || [];
    }

    getAllExecutionLogs(): Map<string, AgentExecutionLogEntry[]> {
        return new Map(this.executionLog);
    }

    clear(agentId?: string): void {
        if (agentId) {
            this.behaviorRegistry.delete(agentId);
            this.executionLog.delete(agentId);
        } else {
            this.behaviorRegistry.clear();
            this.executionLog.clear();
        }
    }
}
