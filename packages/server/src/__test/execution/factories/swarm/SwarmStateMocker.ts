/**
 * Swarm State Mocker
 * 
 * Manages mock states for swarm testing
 */

export interface SwarmMockState {
    expectedOutcomes?: string[];
    successCriteria?: Record<string, unknown>;
    currentState?: Record<string, unknown>;
}

export interface SwarmStateHistoryEntry {
    state: Record<string, unknown>;
    timestamp: Date;
}

export class SwarmStateMocker {
    private stateRegistry = new Map<string, SwarmMockState>();
    private stateHistory = new Map<string, SwarmStateHistoryEntry[]>();

    async register(swarmId: string, state: SwarmMockState): Promise<void> {
        this.stateRegistry.set(swarmId, state);
        this.stateHistory.set(swarmId, []);
    }

    async updateState(swarmId: string, newState: Record<string, unknown>): Promise<void> {
        const mockState = this.stateRegistry.get(swarmId);
        if (!mockState) return;

        // Update current state
        mockState.currentState = newState;
        this.stateRegistry.set(swarmId, mockState);

        // Add to history
        const history = this.stateHistory.get(swarmId) || [];
        history.push({
            state: newState,
            timestamp: new Date(),
        });
        this.stateHistory.set(swarmId, history);
    }

    getState(swarmId: string): Record<string, unknown> | undefined {
        const mockState = this.stateRegistry.get(swarmId);
        return mockState?.currentState;
    }

    checkOutcome(swarmId: string, outcome: string): boolean {
        const mockState = this.stateRegistry.get(swarmId);
        if (!mockState?.expectedOutcomes) return true;
        
        return mockState.expectedOutcomes.includes(outcome);
    }

    checkSuccessCriteria(swarmId: string): boolean {
        const mockState = this.stateRegistry.get(swarmId);
        if (!mockState?.successCriteria) return true;

        const currentState = mockState.currentState;
        if (!currentState) return false;

        // Check all criteria
        for (const [key, expectedValue] of Object.entries(mockState.successCriteria)) {
            const actualValue = this.getNestedValue(currentState, key);
            if (!this.valuesMatch(actualValue, expectedValue)) {
                return false;
            }
        }

        return true;
    }

    getHistory(swarmId: string): SwarmStateHistoryEntry[] {
        return this.stateHistory.get(swarmId) || [];
    }

    getAllStates(): Map<string, SwarmMockState> {
        return new Map(this.stateRegistry);
    }

    clear(swarmId?: string): void {
        if (swarmId) {
            this.stateRegistry.delete(swarmId);
            this.stateHistory.delete(swarmId);
        } else {
            this.stateRegistry.clear();
            this.stateHistory.clear();
        }
    }

    private getNestedValue(obj: Record<string, unknown>, path: string): unknown {
        const keys = path.split(".");
        let value: unknown = obj;
        
        for (const key of keys) {
            if (value && typeof value === "object" && key in (value as Record<string, unknown>)) {
                value = (value as Record<string, unknown>)[key];
            } else {
                return undefined;
            }
        }
        
        return value;
    }

    private valuesMatch(actual: unknown, expected: unknown): boolean {
        // Handle objects that might contain comparison operators
        if (typeof expected === "object" && expected !== null) {
            const expectedObj = expected as Record<string, unknown>;
            
            // Check for comparison operators
            if ("$lt" in expectedObj) {
                return typeof actual === "number" && actual < (expectedObj.$lt as number);
            }
            if ("$lte" in expectedObj) {
                return typeof actual === "number" && actual <= (expectedObj.$lte as number);
            }
            if ("$gt" in expectedObj) {
                return typeof actual === "number" && actual > (expectedObj.$gt as number);
            }
            if ("$gte" in expectedObj) {
                return typeof actual === "number" && actual >= (expectedObj.$gte as number);
            }
        }
        
        // Default equality check
        return JSON.stringify(actual) === JSON.stringify(expected);
    }
}
