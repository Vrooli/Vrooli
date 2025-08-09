/**
 * Swarm Schema Registry
 * 
 * Registry for managing swarm test schemas
 */

export interface SwarmSchema {
    identity: {
        name: string;
        version: string;
    };
    description: string;
    businessPrompt: string;
    details?: string;
    profitTarget?: string;
    agents: string[];
    resources: {
        maxCredits: string;
        maxDurationMs: number;
        maxConcurrentAgents?: number;
    };
    testConfig?: {
        mockState?: Record<string, unknown>;
        expectedOutcomes?: string[];
        successCriteria?: Record<string, unknown>;
    };
}

export class SwarmSchemaRegistry {
    private static schemas: Map<string, SwarmSchema> = new Map();

    static get(schemaPath: string): SwarmSchema {
        const schema = this.schemas.get(schemaPath);
        if (!schema) {
            throw new Error(`Swarm schema not found: ${schemaPath}`);
        }
        return schema;
    }

    static register(name: string, schema: SwarmSchema): void {
        this.schemas.set(name, schema);
    }

    static list(): string[] {
        return Array.from(this.schemas.keys());
    }

    static async validate(schema: unknown): Promise<SwarmSchema> {
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

        if (!s.businessPrompt) {
            throw new Error("Schema must have businessPrompt");
        }

        if (!s.resources) {
            throw new Error("Schema must have resources configuration");
        }

        return s as SwarmSchema;
    }

    static clear(): void {
        this.schemas.clear();
    }
}
