/**
 * Agent Factory
 * 
 * Creates agent test objects from schemas with behavior control
 */

import { generatePK } from "@vrooli/shared";
import type { AgentSchema } from "../../schemas/agents/index.js";
import type { AgentBehaviorTestData, AgentTestData } from "../../types.js";
import { AgentBehaviorMocker } from "./AgentBehaviorMocker.js";
import { AgentDbFactory } from "./AgentDbFactory.js";
import { AgentSchemaLoader } from "./AgentSchemaLoader.js";

export interface AgentFactoryOptions {
    saveToDb?: boolean;
    mockBehaviors?: boolean;
    userId?: string;
    teamId?: string;
}

export class AgentFactory {
    private schemaLoader: AgentSchemaLoader;
    private dbFactory: AgentDbFactory;
    private behaviorMocker: AgentBehaviorMocker;

    constructor() {
        this.schemaLoader = new AgentSchemaLoader();
        this.dbFactory = new AgentDbFactory();
        this.behaviorMocker = new AgentBehaviorMocker();
    }

    async createFromSchema(
        schemaPath: string,
        options: AgentFactoryOptions = {},
    ): Promise<AgentTestData> {
        const schema = await this.schemaLoader.load(schemaPath);

        const agent: AgentTestData = {
            __typename: "Agent" as const,
            id: generatePK().toString(),
            name: schema.identity.name,
            goal: schema.goal,
            subscriptions: schema.subscriptions || [],
            behaviors: this.convertBehaviors(schema.behaviors),
            resources: schema.resources,
            teamId: options.teamId,
            created_by: options.userId || generatePK().toString(),
            created_at: new Date(),
            updated_at: new Date(),
        };

        // Register behavior mocks if needed
        if (options.mockBehaviors && schema.testConfig) {
            await this.behaviorMocker.registerAgent(agent.id, {
                routineCalls: schema.testConfig.expectedRoutineCalls,
                eventEmissions: schema.testConfig.expectedEvents,
                blackboardUpdates: schema.testConfig.expectedBlackboard,
            });
        }

        // Save to database if needed
        if (options.saveToDb) {
            return this.dbFactory.create(agent);
        }

        return agent;
    }

    async createBatch(
        schemas: string[],
        options: AgentFactoryOptions = {},
    ): Promise<AgentTestData[]> {
        return Promise.all(schemas.map(s => this.createFromSchema(s, options)));
    }

    private convertBehaviors(behaviors?: AgentSchema["behaviors"]): AgentBehaviorTestData[] {
        if (!behaviors || !Array.isArray(behaviors)) {
            return [];
        }

        return behaviors.map(behavior => ({
            id: generatePK().toString(),
            trigger: behavior.trigger,
            action: {
                ...behavior.action,
                id: generatePK().toString(),
            },
        }));
    }
}
