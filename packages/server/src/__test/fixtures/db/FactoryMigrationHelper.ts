import { generatePK, generatePublicId } from "@vrooli/shared";
import { DatabaseFixtureFactory } from "./DatabaseFixtureFactory.js";
import type { DbTestFixtures } from "./types.js";

/**
 * Helper class to assist in migrating existing fixtures to the factory pattern
 * Provides utilities to convert static fixtures to dynamic factories
 */
export class FactoryMigrationHelper {
    /**
     * Convert static fixtures to factory methods
     */
    static createFactoryFromFixtures<TCreate, TUpdate = Partial<TCreate>>(
        modelName: string,
        fixtures: DbTestFixtures<TCreate, TUpdate>,
        prismaDelegate: string,
    ): string {
        return `import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "./DatabaseFixtureFactory.js";
import type { RelationConfig } from "./DatabaseFixtureFactory.js";
${this.generateConfigImports(fixtures)}

/**
 * Database fixture factory for ${modelName}
 * Automatically generated from existing fixtures
 */
export class ${modelName}DbFactory extends DatabaseFixtureFactory<
    Prisma.${modelName},
    Prisma.${modelName}CreateInput,
    Prisma.${modelName}Include,
    Prisma.${modelName}UpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('${modelName}', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.${prismaDelegate};
    }

    protected getMinimalData(overrides?: Partial<Prisma.${modelName}CreateInput>): Prisma.${modelName}CreateInput {
        return {
            ${this.formatObjectProperties(fixtures.minimal, 3)}
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.${modelName}CreateInput>): Prisma.${modelName}CreateInput {
        return {
            ${this.formatObjectProperties(fixtures.complete, 3)}
            ...overrides,
        };
    }

    protected getDefaultInclude(): Prisma.${modelName}Include {
        return {
            ${this.generateDefaultInclude(modelName)}
        };
    }

    protected async applyRelationships(
        baseData: Prisma.${modelName}CreateInput,
        config: RelationConfig,
        tx: any
    ): Promise<Prisma.${modelName}CreateInput> {
        let data = { ...baseData };

        ${this.generateRelationshipHandlers(modelName)}

        return data;
    }

    ${this.generateScenarioMethods(fixtures.edgeCases)}

    ${this.generateConstraintChecks(modelName)}
}

// Export singleton instance
export const ${modelName.toLowerCase()}DbFactory = (prisma: PrismaClient) => new ${modelName}DbFactory(prisma);
`;
    }

    /**
     * Format object properties for code generation
     */
    private static formatObjectProperties(obj: any, indentLevel: number): string {
        const indent = " ".repeat(indentLevel * 4);
        const entries = Object.entries(obj);
        
        return entries.map(([key, value]) => {
            if (typeof value === "string") {
                return `${indent}${key}: "${value}",`;
            } else if (typeof value === "object" && value !== null) {
                if (Array.isArray(value)) {
                    return `${indent}${key}: ${JSON.stringify(value, null, 4).replace(/\n/g, "\n" + indent)},`;
                } else {
                    return `${indent}${key}: {\n${this.formatObjectProperties(value, indentLevel + 1)}\n${indent}},`;
                }
            } else {
                return `${indent}${key}: ${value},`;
            }
        }).join("\n");
    }

    /**
     * Generate config imports based on fixture data
     */
    private static generateConfigImports(fixtures: DbTestFixtures<any>): string {
        const imports: Set<string> = new Set();
        
        // Check for config usage in fixtures
        const checkForConfigs = (obj: any) => {
            if (!obj || typeof obj !== "object") return;
            
            Object.entries(obj).forEach(([key, value]) => {
                if (key.endsWith("Settings") || key.endsWith("Config")) {
                    const configName = key.replace("Settings", "").replace("Config", "");
                    imports.add(`import { ${configName}ConfigFixtures } from "@vrooli/shared/test-fixtures";`);
                } else if (typeof value === "object") {
                    checkForConfigs(value);
                }
            });
        };
        
        checkForConfigs(fixtures);
        
        return Array.from(imports).join("\n");
    }

    /**
     * Generate default include based on model name
     */
    private static generateDefaultInclude(modelName: string): string {
        // Common relationships by model
        const includeMap: Record<string, string[]> = {
            User: ["emails", "auths", "translations", "teams { select: { id: true, team: { select: { id: true, name: true } } } }"],
            Team: ["members { select: { id: true, user: { select: { id: true, name: true } } } }", "translations"],
            Project: ["team", "versions", "translations", "owner"],
            Chat: ["participants", "messages { take: 10 }", "translations", "team"],
            // Add more as needed
        };
        
        const includes = includeMap[modelName] || [];
        return includes.join(",\n            ");
    }

    /**
     * Generate relationship handlers
     */
    private static generateRelationshipHandlers(modelName: string): string {
        const handlers: Record<string, string> = {
            User: `
        // Handle team memberships
        if (config.teams && Array.isArray(config.teams)) {
            data.teams = {
                create: config.teams.map((team: any) => ({
                    id: generatePK(),
                    teamId: team.teamId,
                    role: team.role || 'Member',
                }))
            };
        }

        // Handle authentication
        if (config.withAuth) {
            data.auths = {
                create: [{
                    id: generatePK(),
                    provider: 'Password',
                    hashed_password: '$2b$10$dummy.hashed.password.for.testing',
                }]
            };
            data.emails = {
                create: [{
                    id: generatePK(),
                    emailAddress: \`test_\${nanoid(6)}@example.com\`,
                    verifiedAt: new Date(),
                }]
            };
        }`,
            Team: `
        // Handle members
        if (config.members && Array.isArray(config.members)) {
            data.members = {
                create: config.members.map((member: any) => ({
                    id: generatePK(),
                    userId: member.userId,
                    role: member.role || 'Member',
                }))
            };
        }`,
            Project: `
        // Handle team connection
        if (config.teamId) {
            data.team = { connect: { id: config.teamId } };
        }

        // Handle owner
        if (config.ownerId) {
            data.owner = { connect: { id: config.ownerId } };
        }`,
            // Add more as needed
        };
        
        return handlers[modelName] || "// TODO: Add relationship handlers";
    }

    /**
     * Generate scenario methods
     */
    private static generateScenarioMethods(edgeCases: Record<string, any>): string {
        if (!edgeCases || Object.keys(edgeCases).length === 0) {
            return "";
        }
        
        const scenarios = Object.entries(edgeCases).map(([name, data]) => {
            return `
    async create${this.capitalize(name)}(): Promise<any> {
        return this.createMinimal({
            ${this.formatObjectProperties(data, 3)}
        });
    }`;
        }).join("\n");
        
        return `
    // Scenario creation methods
    ${scenarios}`;
    }

    /**
     * Generate constraint checks
     */
    private static generateConstraintChecks(modelName: string): string {
        return `
    protected async checkModelConstraints(record: Prisma.${modelName}): Promise<string[]> {
        const violations: string[] = [];
        
        // Add model-specific constraint checks here
        ${this.getModelSpecificConstraints(modelName)}
        
        return violations;
    }`;
    }

    /**
     * Get model-specific constraints
     */
    private static getModelSpecificConstraints(modelName: string): string {
        const constraints: Record<string, string> = {
            User: `
        // Check handle uniqueness
        if (record.handle) {
            const duplicate = await this.prisma.user.findFirst({
                where: { 
                    handle: record.handle,
                    id: { not: record.id }
                }
            });
            if (duplicate) {
                violations.push('Handle must be unique');
            }
        }`,
            Team: `
        // Check team name length
        if (record.name && record.name.length > 255) {
            violations.push('Team name too long');
        }`,
            // Add more as needed
        };
        
        return constraints[modelName] || "// No specific constraints";
    }

    /**
     * Capitalize first letter
     */
    private static capitalize(str: string): string {
        return str.charAt(0).toUpperCase() + str.slice(1);
    }

    /**
     * Generate migration instructions
     */
    static generateMigrationInstructions(modelName: string): string {
        return `
Migration Instructions for ${modelName} Fixtures:

1. Review the generated factory code
2. Add any missing relationship handlers
3. Implement scenario-specific methods
4. Add constraint validation logic
5. Test the factory with existing tests
6. Remove old static fixture exports
7. Update imports in test files

Example usage after migration:

\`\`\`typescript
import { ${modelName}DbFactory } from "../fixtures/db";
import { prisma } from "../setup";

const factory = new ${modelName}DbFactory(prisma);

// Create minimal record
const minimal = await factory.createMinimal();

// Create with relationships
const withRelations = await factory.createWithRelations({
    // Add relationship config
});

// Verify state
await factory.verifyState(minimal.id, { /* expected state */ });

// Cleanup
await factory.cleanup([minimal.id]);
\`\`\`
`;
    }
}
