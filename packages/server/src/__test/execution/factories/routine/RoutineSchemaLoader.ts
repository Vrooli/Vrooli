/**
 * Routine Schema Loader
 * 
 * Loads and validates routine schemas from JSON files
 */

import { RoutineSchemaRegistry, type RoutineSchema } from "../../schemas/routines/index.js";

export class RoutineSchemaLoader {
    async load(schemaPath: string): Promise<RoutineSchema> {
        return RoutineSchemaRegistry.get(schemaPath);
    }

    async loadBatch(schemaPaths: string[]): Promise<RoutineSchema[]> {
        return Promise.all(schemaPaths.map(path => this.load(path)));
    }

    async validate(schema: unknown): Promise<RoutineSchema> {
        return RoutineSchemaRegistry.validate(schema);
    }

    list(): string[] {
        return RoutineSchemaRegistry.list();
    }
}
