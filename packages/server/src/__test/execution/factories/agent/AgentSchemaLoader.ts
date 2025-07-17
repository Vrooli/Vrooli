/**
 * Agent Schema Loader
 * 
 * Loads and validates agent schemas from JSON files
 */

import { AgentSchemaRegistry, type AgentSchema } from "../../schemas/agents/index.js";

export class AgentSchemaLoader {
    async load(schemaPath: string): Promise<AgentSchema> {
        return AgentSchemaRegistry.get(schemaPath);
    }

    async loadBatch(schemaPaths: string[]): Promise<AgentSchema[]> {
        return Promise.all(schemaPaths.map(path => this.load(path)));
    }

    async validate(schema: unknown): Promise<AgentSchema> {
        return AgentSchemaRegistry.validate(schema);
    }

    list(): string[] {
        return AgentSchemaRegistry.list();
    }
}
