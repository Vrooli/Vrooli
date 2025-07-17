/**
 * Agent Factory
 * 
 * Creates agent test objects from schemas with behavior control
 */

import { generatePK } from "@vrooli/shared";
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
    ): Promise<any> {
        const schema = await this.schemaLoader.load(schemaPath);

        const agent = {
            __typename: "Agent",
            id: generatePK(),
            name: schema.identity.name,
            goal: schema.goal,
            subscriptions: schema.subscriptions,
            behaviors: this.convertBehaviors(schema.behaviors),
            resources: schema.resources,
            teamId: options.teamId,
            created_by: options.userId || generatePK(),
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
    ): Promise<any[]> {
        return Promise.all(schemas.map(s => this.createFromSchema(s, options)));
    }

    private convertBehaviors(behaviors: any[]): any[] {
        if (!behaviors || !Array.isArray(behaviors)) {
            console.warn("[AgentFactory] No behaviors provided or behaviors is not an array");
            return [];
        }

        return behaviors.map(behavior => ({
            id: generatePK(),
            trigger: behavior.trigger,
            action: {
                ...behavior.action,
                id: generatePK(),
            },
        }));
    }
}
