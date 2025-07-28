/**
 * Swarm Factory
 * 
 * Creates swarm/team test objects from schemas
 */

import { generatePK } from "@vrooli/shared";
import { SwarmSchemaLoader } from "./SwarmSchemaLoader.js";
import { SwarmStateMocker } from "./SwarmStateMocker.js";
import { TeamDbFactory } from "./TeamDbFactory.js";

export interface SwarmFactoryOptions {
    saveToDb?: boolean;
    mockState?: boolean;
    userId?: string;
}

export class SwarmFactory {
    private schemaLoader: SwarmSchemaLoader;
    private dbFactory: TeamDbFactory;
    private stateMocker: SwarmStateMocker;

    constructor() {
        this.schemaLoader = new SwarmSchemaLoader();
        this.dbFactory = new TeamDbFactory();
        this.stateMocker = new SwarmStateMocker();
    }

    async createFromSchema(
        schemaPath: string,
        options: SwarmFactoryOptions = {},
    ): Promise<any> {
        const schema = await this.schemaLoader.load(schemaPath);

        const team = {
            __typename: "Team" as const,
            id: generatePK().toString(),
            name: schema.identity.name,
            description: schema.details || schema.businessPrompt || "",
            businessPrompt: schema.businessPrompt,
            details: schema.details,
            profitTarget: schema.profitTarget,
            agents: schema.agents, // Agent schema references
            resources: schema.resources,
            created_by: options.userId || generatePK().toString(),
            created_at: new Date(),
            updated_at: new Date(),
        };

        // Set up state mocking if needed
        if (options.mockState && schema.testConfig) {
            await this.stateMocker.register(team.id, {
                expectedOutcomes: schema.testConfig.expectedOutcomes,
                successCriteria: schema.testConfig.successCriteria,
            });
        }

        // Save to database if needed
        if (options.saveToDb) {
            return this.dbFactory.create(team);
        }

        return team;
    }

    async createBatch(
        schemas: string[],
        options: SwarmFactoryOptions = {},
    ): Promise<any[]> {
        return Promise.all(schemas.map(s => this.createFromSchema(s, options)));
    }
}
