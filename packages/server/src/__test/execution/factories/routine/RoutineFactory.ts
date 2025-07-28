/**
 * Routine Factory
 * 
 * Creates routine test objects from schemas with proper database integration
 */

import { generatePK } from "@vrooli/shared";
import { RoutineSchemaLoader } from "./RoutineSchemaLoader.js";
import { RoutineDbFactory } from "./RoutineDbFactory.js";
import { RoutineResponseMocker } from "./RoutineResponseMocker.js";
import type { RoutineSchema } from "../../schemas/routines/index.js";
import type { RoutineTestData, RoutineFactoryOptions } from "../../types.js";

export class RoutineFactory {
    private schemaLoader: RoutineSchemaLoader;
    private dbFactory: RoutineDbFactory;
    private responseMocker: RoutineResponseMocker;

    constructor() {
        this.schemaLoader = new RoutineSchemaLoader();
        this.dbFactory = new RoutineDbFactory();
        this.responseMocker = new RoutineResponseMocker();
    }

    async createFromSchema(
        schemaPath: string, 
        options: RoutineFactoryOptions = {},
    ): Promise<RoutineTestData> {
        // 1. Load and validate schema
        const schema = await this.schemaLoader.load(schemaPath);
        
        // 2. Convert to routine object
        const routine = this.convertSchemaToRoutine(schema, options.userId);
        
        // 3. Save to database if needed
        if (options.saveToDb) {
            const dbRoutine = await this.dbFactory.create(routine);
            
            // 4. Set up response mocking
            if (options.mockResponses && schema.testConfig?.mockResponses) {
                await this.responseMocker.register(
                    dbRoutine.id, 
                    schema.testConfig.mockResponses,
                );
            }
            
            return dbRoutine;
        }
        
        return routine;
    }

    async createBatch(
        schemas: string[], 
        options: RoutineFactoryOptions = {},
    ): Promise<RoutineTestData[]> {
        return Promise.all(schemas.map(s => this.createFromSchema(s, options)));
    }

    private convertSchemaToRoutine(schema: RoutineSchema, userId?: string): RoutineTestData {
        const now = new Date();
        const routineId = generatePK();
        const routineVersionId = generatePK();

        return {
            __typename: "Routine",
            id: routineId.toString(),
            name: schema.identity.name,
            description: schema.description,
            created_at: now,
            updated_at: now,
            created_by: userId || generatePK().toString(),
            versions: [{
                __typename: "RoutineVersion",
                id: routineVersionId.toString(),
                routineId: routineId.toString(),
                version: schema.identity.version,
                isLatest: true,
                isPublic: true,
                inputs: schema.inputs?.map((input, index) => ({
                    __typename: "RoutineVersionInput",
                    id: generatePK().toString(),
                    index,
                    name: input.name,
                    type: input.type,
                    required: input.required || false,
                    default: input.default,
                    routineVersionId: routineVersionId.toString(),
                })) || [],
                outputs: schema.outputs?.map((output, index) => ({
                    __typename: "RoutineVersionOutput",
                    id: generatePK().toString(),
                    index,
                    name: output.name,
                    type: output.type,
                    routineVersionId: routineVersionId.toString(),
                })) || [],
                nodes: schema.steps?.map((step, index) => ({
                    __typename: "Node",
                    id: step.id,
                    type: this.mapStepTypeToNodeType(step.type),
                    index,
                    routineVersionId: routineVersionId.toString(),
                    data: step.config,
                })) || [],
                complexity: schema.steps?.length || 1,
                created_at: now,
                updated_at: now,
            }],
        };
    }

    private mapStepTypeToNodeType(stepType: string): string {
        const typeMap: Record<string, string> = {
            "code": "Code",
            "prompt": "Prompt",
            "decision": "Decision",
            "workflow": "Routine",
        };
        return typeMap[stepType] || "Code";
    }
}
