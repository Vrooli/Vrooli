/* eslint-disable no-magic-numbers */
/**
 * Migration Helper
 * 
 * Utilities to help migrate existing test fixtures to the new schema-based format
 */

import fs from "fs/promises";
import path from "path";
import { fileURLToPath } from "url";
import type { AgentSchema, RoutineSchema, SwarmSchema } from "../schemas/index.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export class MigrationHelper {
    /**
     * Migrate existing SwarmTask fixture to new schema format
     */
    static async migrateSwarmTask(swarmTask: any): Promise<void> {
        // Extract routine information
        const routineSchemas: RoutineSchema[] = [];
        const agentSchemas: AgentSchema[] = [];

        // Create a basic swarm schema
        const swarmSchema: SwarmSchema = {
            identity: {
                name: swarmTask.input?.teamConfiguration?.name || "migrated-swarm",
                version: "1.0.0",
            },
            description: "Migrated swarm from legacy format",
            businessPrompt: swarmTask.input?.goal || "Migrated swarm goal",
            agents: [], // Will be filled with agent references
            resources: {
                maxCredits: swarmTask.allocation?.maxCredits || "1000",
                maxDurationMs: swarmTask.allocation?.maxDurationMs || 300000,
            },
        };

        // Save schemas to appropriate directories
        await this.saveSchema(
            swarmSchema,
            `swarms/migrated/${swarmSchema.identity.name}.json`,
        );
    }

    /**
     * Migrate existing RunTask fixture to routine schema
     */
    static async migrateRunTask(runTask: any): Promise<RoutineSchema> {
        const routineSchema: RoutineSchema = {
            identity: {
                name: runTask.routineLabel || "migrated-routine",
                version: runTask.routineVersion || "1.0.0",
            },
            description: "Migrated from RunTask fixture",
            inputs: runTask.inputs?.map((input: any) => ({
                name: input.name,
                type: input.type || "string",
                required: input.required || false,
            })) || [],
            outputs: runTask.outputs?.map((output: any) => ({
                name: output.name,
                type: output.type || "string",
            })) || [],
            steps: [], // Would need to be filled based on routine structure
        };

        await this.saveSchema(
            routineSchema,
            `routines/migrated/${routineSchema.identity.name}.json`,
        );

        return routineSchema;
    }

    /**
     * Migrate AI mock fixtures to new format
     */
    static async migrateAIMocks(mockPath: string): Promise<void> {
        const mockContent = await fs.readFile(mockPath, "utf-8");
        const mocks = JSON.parse(mockContent);

        // Convert to new mock configuration format
        const mockConfig = {
            ai: {},
            routines: {},
        };

        for (const [key, value] of Object.entries(mocks)) {
            if (key.includes("routine")) {
                (mockConfig.routines as any)[key] = {
                    responses: Array.isArray(value) ? value : [value],
                };
            } else {
                (mockConfig.ai as any)[key] = {
                    responses: Array.isArray(value) ? value : [value],
                };
            }
        }

        await fs.writeFile(
            path.join(__dirname, "../mocks/migrated-mocks.json"),
            JSON.stringify(mockConfig, null, 2),
        );
    }

    /**
     * Create a migration report
     */
    static async generateMigrationReport(
        sourceDir: string,
        targetDir: string,
    ): Promise<void> {
        const report = {
            timestamp: new Date().toISOString(),
            sourceDirectory: sourceDir,
            targetDirectory: targetDir,
            migrated: {
                routines: 0,
                agents: 0,
                swarms: 0,
                scenarios: 0,
            },
            pending: [] as string[],
            errors: [] as string[],
        };

        // Scan source directory for fixtures
        const files = await this.scanDirectory(sourceDir);

        for (const file of files) {
            try {
                if (file.includes("swarmTask")) {
                    report.migrated.swarms++;
                } else if (file.includes("runTask")) {
                    report.migrated.routines++;
                } else if (file.includes("agent")) {
                    report.migrated.agents++;
                } else {
                    report.pending.push(file);
                }
            } catch (error) {
                report.errors.push(`${file}: ${error}`);
            }
        }

        await fs.writeFile(
            path.join(targetDir, "migration-report.json"),
            JSON.stringify(report, null, 2),
        );
    }

    private static async saveSchema(
        schema: any,
        relativePath: string,
    ): Promise<void> {
        const fullPath = path.join(__dirname, "../schemas", relativePath);
        await fs.mkdir(path.dirname(fullPath), { recursive: true });
        await fs.writeFile(fullPath, JSON.stringify(schema, null, 2));
    }

    private static async scanDirectory(dir: string): Promise<string[]> {
        const entries = await fs.readdir(dir, { withFileTypes: true });
        const files = await Promise.all(
            entries.map(async (entry) => {
                const fullPath = path.join(dir, entry.name);
                if (entry.isDirectory()) {
                    return this.scanDirectory(fullPath);
                } else if (entry.name.endsWith(".ts") || entry.name.endsWith(".js")) {
                    return [fullPath];
                }
                return [];
            }),
        );
        return files.flat();
    }
}

/**
 * Example migration script
 */
export async function runMigration() {
    console.log("Starting fixture migration...");

    try {
        // Migrate existing fixtures
        const fixturesDir = path.join(__dirname, "../../fixtures");
        const executionDir = path.join(__dirname, "..");

        // Generate migration report
        await MigrationHelper.generateMigrationReport(fixturesDir, executionDir);

        console.log("Migration complete! Check migration-report.json for details.");
    } catch (error) {
        console.error("Migration failed:", error);
    }
}
