/**
 * Agent Schema Registry
 * 
 * Registry for managing agent test schemas
 */

export interface AgentSchema {
    identity: {
        name: string;
        version: string;
    };
    description: string;
    goal: string;
    subscriptions?: string[];
    behaviors?: Array<{
        trigger: {
            topic: string;
            conditions?: Record<string, unknown>;
        };
        action: {
            id: string;
            type: "routine" | "emit" | "accumulate" | "decision";
            label?: string;
            topic?: string;
        };
    }>;
    resources?: {
        maxCredits?: string;
        maxDurationMs?: number;
        preferredModel?: string;
    };
    testConfig?: {
        mockBehaviors?: Record<string, unknown>;
        expectedRoutineCalls?: string[];
        expectedEvents?: string[];
        expectedBlackboard?: Record<string, unknown>;
    };
}

export class AgentSchemaRegistry {
    private static schemas: Map<string, AgentSchema> = new Map();

    static get(schemaPath: string): AgentSchema {
        const schema = this.schemas.get(schemaPath);
        if (!schema) {
            throw new Error(`Agent schema not found: ${schemaPath}`);
        }
        return schema;
    }

    static register(name: string, schema: AgentSchema): void {
        this.schemas.set(name, schema);
    }

    static list(): string[] {
        return Array.from(this.schemas.keys());
    }

    static async validate(schema: unknown): Promise<AgentSchema> {
        if (!schema || typeof schema !== "object") {
            throw new Error("Schema must be an object");
        }

        const s = schema as any;
        if (!s.identity?.name || !s.identity?.version) {
            throw new Error("Schema must have identity.name and identity.version");
        }

        if (!s.description) {
            throw new Error("Schema must have description");
        }

        if (!s.goal) {
            throw new Error("Schema must have goal");
        }

        return s as AgentSchema;
    }

    static clear(): void {
        this.schemas.clear();
    }
}
