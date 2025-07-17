/**
 * Swarm Schema Loader
 * 
 * Loads and validates swarm/team schemas from JSON files
 */

import { SwarmSchemaRegistry, type SwarmSchema } from "../../schemas/swarms/index.js";

export class SwarmSchemaLoader {
    async load(schemaPath: string): Promise<SwarmSchema> {
        return SwarmSchemaRegistry.get(schemaPath);
    }

    async loadBatch(schemaPaths: string[]): Promise<SwarmSchema[]> {
        return Promise.all(schemaPaths.map(path => this.load(path)));
    }

    async validate(schema: unknown): Promise<SwarmSchema> {
        return SwarmSchemaRegistry.validate(schema);
    }

    list(): string[] {
        return SwarmSchemaRegistry.list();
    }
}
