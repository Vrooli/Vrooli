/**
 * Routine Schema Registry
 * 
 * Registry for managing routine test schemas
 */

import type { MockResponse } from "../../types.js";

export interface RoutineSchema {
    identity: {
        name: string;
        version: string;
    };
    description: string;
    inputs?: Array<{
        name: string;
        type: string;
        required?: boolean;
        default?: unknown;
    }>;
    outputs?: Array<{
        name: string;
        type: string;
    }>;
    steps?: Array<{
        id: string;
        type: string;
        config: Record<string, unknown>;
    }>;
    testConfig?: {
        mockResponses?: MockResponse[];
    };
}

export class RoutineSchemaRegistry {
    private static schemas: Map<string, RoutineSchema> = new Map();

    static get(schemaPath: string): RoutineSchema {
        const schema = this.schemas.get(schemaPath);
        if (!schema) {
            throw new Error(`Routine schema not found: ${schemaPath}`);
        }
        return schema;
    }

    static register(name: string, schema: RoutineSchema): void {
        this.schemas.set(name, schema);
    }

    static list(): string[] {
        return Array.from(this.schemas.keys());
    }

    static async validate(schema: unknown): Promise<RoutineSchema> {
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

        return s as RoutineSchema;
    }

    static clear(): void {
        this.schemas.clear();
    }
}
