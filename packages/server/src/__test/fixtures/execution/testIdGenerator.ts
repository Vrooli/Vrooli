/**
 * Test ID Generator for Fixtures
 * 
 * Provides consistent ID generation for test fixtures without requiring
 * snowflake initialization. Uses predictable IDs for testing.
 */

/**
 * Generate a test ID with a prefix and counter
 * Returns a string ID suitable for test fixtures
 */
export function generateTestId(prefix: string, counter?: number): string {
    const timestamp = Date.now();
    const count = counter !== undefined ? counter : Math.floor(Math.random() * 10000);
    return `${prefix}_${timestamp}_${count}`;
}

/**
 * Generate a mock bigint ID for fixtures that need numeric IDs
 * Returns a string representation of a bigint
 */
export function generateMockBigInt(prefix: string, counter?: number): string {
    // Create a large number that looks like a snowflake ID
    // Format: [timestamp: 41 bits][worker: 10 bits][sequence: 12 bits]
    const timestamp = Date.now() - 1609459200000; // Offset from 2021-01-01
    const workerId = prefix.charCodeAt(0) % 1024; // Use first char as worker ID
    const sequence = counter !== undefined ? counter % 4096 : Math.floor(Math.random() * 4096);
    
    // Combine into a bigint-like string
    const bigintValue = (BigInt(timestamp) << 22n) | (BigInt(workerId) << 12n) | BigInt(sequence);
    return bigintValue.toString();
}

/**
 * Pre-generated test IDs for consistent testing
 * These can be used directly in const fixtures
 */
export const TEST_IDS = {
    // Organization IDs
    HEALTHCARE_ORG: "healthcare_org_1700000000000_1001",
    FINANCIAL_ORG: "financial_org_1700000000001_1002",
    RESEARCH_ORG: "research_org_1700000000002_1003",
    
    // Agent IDs (as bigint strings)
    AGENT_1: "7500000000000000001",
    AGENT_2: "7500000000000000002",
    AGENT_3: "7500000000000000003",
    AGENT_4: "7500000000000000004",
    AGENT_5: "7500000000000000005",
    
    // Swarm IDs
    SWARM_1: "swarm_1700000000000_2001",
    SWARM_2: "swarm_1700000000001_2002",
    SWARM_3: "swarm_1700000000002_2003",
    
    // Run IDs (as bigint strings)
    RUN_1: "8500000000000000001",
    RUN_2: "8500000000000000002",
    RUN_3: "8500000000000000003",
    RUN_4: "8500000000000000004",
    RUN_5: "8500000000000000005",
    
    // Event IDs
    EVENT_1: "event_1700000000000_3001",
    EVENT_2: "event_1700000000001_3002",
    EVENT_3: "event_1700000000002_3003",
    
    // Context IDs
    CONTEXT_1: "context_1700000000000_4001",
    CONTEXT_2: "context_1700000000001_4002",
    CONTEXT_3: "context_1700000000002_4003",
} as const;

/**
 * Factory functions for creating test entities with proper IDs
 */
export const TestIdFactory = {
    organization: (suffix?: number) => generateTestId("org", suffix),
    agent: (suffix?: number) => generateMockBigInt("agent", suffix),
    swarm: (suffix?: number) => generateTestId("swarm", suffix),
    run: (suffix?: number) => generateMockBigInt("run", suffix),
    event: (suffix?: number) => generateTestId("event", suffix),
    context: (suffix?: number) => generateTestId("context", suffix),
    routine: (suffix?: number) => generateTestId("routine", suffix),
    team: (suffix?: number) => generateTestId("team", suffix),
};
