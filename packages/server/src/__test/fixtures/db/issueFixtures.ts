/* eslint-disable no-magic-numbers */
import { type Prisma } from "@prisma/client";
import { generatePK, generatePublicId } from "@vrooli/shared";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { BulkSeedResult, DbErrorScenarios, DbTestFixtures } from "./types.js";

/**
 * Database fixtures for Issue model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Cached IDs for consistent testing - lazy initialization pattern
let _issueDbIds: Record<string, bigint> | null = null;
export function getIssueDbIds() {
    if (!_issueDbIds) {
        _issueDbIds = {
            issue1: generatePK(),
            issue2: generatePK(),
            issue3: generatePK(),
            user1: generatePK(),
            user2: generatePK(),
            project1: generatePK(),
            routine1: generatePK(),
            label1: generatePK(),
            label2: generatePK(),
        };
    }
    return _issueDbIds;
}

/**
 * Enhanced test fixtures for Issue model following standard structure
 */
export const issueDbFixtures: DbTestFixtures<Prisma.issueCreateInput> = {
    minimal: {
        id: generatePK(),
        publicId: generatePublicId(),
        createdBy: { connect: { id: getIssueDbIds().user1 } },
    },
    complete: {
        id: generatePK(),
        publicId: generatePublicId(),
        createdBy: { connect: { id: getIssueDbIds().user1 } },
        closedBy: { connect: { id: getIssueDbIds().user2 } },
        project: { connect: { id: getIssueDbIds().project1 } },
        labels: {
            connect: [
                { id: getIssueDbIds().label1 },
                { id: getIssueDbIds().label2 },
            ],
        },
        translations: {
            create: [
                {
                    id: generatePK(),
                    language: "en",
                    name: "Complete Issue Report",
                    description: "This is a comprehensive issue report with detailed information about the problem, steps to reproduce, and expected behavior.",
                },
                {
                    id: generatePK(),
                    language: "es",
                    name: "Reporte Completo de Problema",
                    description: "Este es un reporte comprensivo de problema con información detallada sobre el problema, pasos para reproducir, y comportamiento esperado.",
                },
            ],
        },
    },
    invalid: {
        missingRequired: {
            // Missing required createdBy
            publicId: generatePublicId(),
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            publicId: 123, // Should be string
            createdBy: "invalid-user-reference", // Should be connect object
            closedBy: "invalid-user-reference", // Should be connect object
        },
        noTargetObject: {
            id: generatePK(),
            publicId: generatePublicId(),
            createdBy: { connect: { id: getIssueDbIds().user1 } },
            // No target object connected (business logic issue)
        },
        multipleTargets: {
            id: generatePK(),
            publicId: generatePublicId(),
            createdBy: { connect: { id: getIssueDbIds().user1 } },
            project: { connect: { id: getIssueDbIds().project1 } },
            routine: { connect: { id: getIssueDbIds().routine1 } },
            // Multiple targets (business logic violation)
        },
    },
    edgeCases: {
        projectIssue: {
            id: generatePK(),
            publicId: generatePublicId(),
            createdBy: { connect: { id: getIssueDbIds().user1 } },
            project: { connect: { id: getIssueDbIds().project1 } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Project Issue",
                    description: "Issue specific to a project",
                }],
            },
        },
        routineIssue: {
            id: generatePK(),
            publicId: generatePublicId(),
            createdBy: { connect: { id: getIssueDbIds().user1 } },
            routine: { connect: { id: getIssueDbIds().routine1 } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Routine Issue",
                    description: "Issue specific to a routine",
                }],
            },
        },
        closedIssue: {
            id: generatePK(),
            publicId: generatePublicId(),
            createdBy: { connect: { id: getIssueDbIds().user1 } },
            closedBy: { connect: { id: getIssueDbIds().user2 } },
            project: { connect: { id: getIssueDbIds().project1 } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Closed Issue",
                    description: "This issue has been resolved",
                }],
            },
        },
        labeledIssue: {
            id: generatePK(),
            publicId: generatePublicId(),
            createdBy: { connect: { id: getIssueDbIds().user1 } },
            project: { connect: { id: getIssueDbIds().project1 } },
            labels: {
                connect: [
                    { id: getIssueDbIds().label1 },
                    { id: getIssueDbIds().label2 },
                ],
            },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Labeled Issue",
                    description: "Issue with multiple labels",
                }],
            },
        },
        multiLanguageIssue: {
            id: generatePK(),
            publicId: generatePublicId(),
            createdBy: { connect: { id: getIssueDbIds().user1 } },
            project: { connect: { id: getIssueDbIds().project1 } },
            translations: {
                create: [
                    { id: generatePK(), language: "en", name: "Multi-language Issue", description: "English description" },
                    { id: generatePK(), language: "es", name: "Problema Multi-idioma", description: "Descripción en español" },
                    { id: generatePK(), language: "fr", name: "Problème Multi-langue", description: "Description française" },
                ],
            },
        },
        selfClosedIssue: {
            id: generatePK(),
            publicId: generatePublicId(),
            createdBy: { connect: { id: getIssueDbIds().user1 } },
            closedBy: { connect: { id: getIssueDbIds().user1 } }, // Same user created and closed
            project: { connect: { id: getIssueDbIds().project1 } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Self-closed Issue",
                    description: "Issue closed by the same user who created it",
                }],
            },
        },
    },
};

/**
 * Enhanced factory for creating issue database fixtures
 */
export class IssueDbFactory extends EnhancedDbFactory<Prisma.issueCreateInput> {

    /**
     * Get the test fixtures for Issue model
     */
    protected getFixtures(): DbTestFixtures<Prisma.issueCreateInput> {
        return issueDbFixtures;
    }

    /**
     * Get Issue-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: getIssueDbIds().issue1, // Duplicate ID
                    publicId: generatePublicId(),
                    createdBy: { connect: { id: getIssueDbIds().user1 } },
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    createdBy: { connect: { id: "non-existent-user-id" } },
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    publicId: "", // Empty publicId
                    createdBy: { connect: { id: getIssueDbIds().user1 } },
                },
            },
            validation: {
                requiredFieldMissing: issueDbFixtures.invalid.missingRequired,
                invalidDataType: issueDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    publicId: "a".repeat(500), // PublicId too long
                    createdBy: { connect: { id: getIssueDbIds().user1 } },
                },
            },
            businessLogic: {
                noTargetObject: issueDbFixtures.invalid.noTargetObject,
                multipleTargets: issueDbFixtures.invalid.multipleTargets,
                invalidClosure: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    createdBy: { connect: { id: getIssueDbIds().user1 } },
                    closedBy: { connect: { id: "non-existent-user-id" } }, // Invalid closer
                    project: { connect: { id: getIssueDbIds().project1 } },
                },
            },
        };
    }

    /**
     * Issue-specific validation
     */
    protected validateSpecific(data: Prisma.issueCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to Issue
        if (!data.createdBy) errors.push("Issue createdBy is required");
        if (!data.publicId) errors.push("Issue publicId is required");

        // Check business logic - should have exactly one target object
        const targetFields = ["api", "code", "note", "project", "routine", "standard", "team"];
        const connectedTargets = targetFields.filter(field => data[field as keyof Prisma.issueCreateInput]);

        if (connectedTargets.length === 0) {
            warnings.push("Issue should reference a target object");
        } else if (connectedTargets.length > 1) {
            errors.push("Issue cannot reference multiple target objects");
        }

        // Check closure logic
        if (data.closedBy && data.createdBy &&
            typeof data.closedBy === "object" && "connect" in data.closedBy &&
            typeof data.createdBy === "object" && "connect" in data.createdBy &&
            data.closedBy.connect.id === data.createdBy.connect.id) {
            warnings.push("Issue closed by the same user who created it");
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        createdById: string,
        overrides?: Partial<Prisma.issueCreateInput>,
    ): Prisma.issueCreateInput {
        const factory = new IssueDbFactory();
        return factory.createMinimal({
            createdBy: { connect: { id: createdById } },
            ...overrides,
        });
    }

    static createForObject(
        createdById: string,
        objectId: string,
        objectType: string,
        overrides?: Partial<Prisma.issueCreateInput>,
    ): Prisma.issueCreateInput {
        const base = this.createMinimal(createdById, overrides);

        // Add the appropriate connection based on object type
        const connections: Record<string, any> = {
            Api: { api: { connect: { id: objectId } } },
            Code: { code: { connect: { id: objectId } } },
            Note: { note: { connect: { id: objectId } } },
            Project: { project: { connect: { id: objectId } } },
            Routine: { routine: { connect: { id: objectId } } },
            Standard: { standard: { connect: { id: objectId } } },
            Team: { team: { connect: { id: objectId } } },
        };

        return {
            ...base,
            ...(connections[objectType] || {}),
        };
    }

    static createWithTranslations(
        createdById: string,
        translations: Array<{ language: string; name: string; description: string }>,
        overrides?: Partial<Prisma.issueCreateInput>,
    ): Prisma.issueCreateInput {
        return {
            ...this.createMinimal(createdById, overrides),
            translations: {
                create: translations.map(t => ({
                    id: generatePK(),
                    language: t.language,
                    name: t.name,
                    description: t.description,
                })),
            },
        };
    }

    static createWithLabels(
        createdById: string,
        labelIds: string[],
        overrides?: Partial<Prisma.issueCreateInput>,
    ): Prisma.issueCreateInput {
        return {
            ...this.createMinimal(createdById, overrides),
            labels: {
                connect: labelIds.map(id => ({ id })),
            },
        };
    }
}

/**
 * Enhanced helper to seed multiple test issues with comprehensive options
 */
export async function seedIssues(
    prisma: any,
    options: {
        createdById: string;
        count?: number;
        forObject?: { id: string; type: string };
        withTranslations?: boolean;
        withLabels?: string[];
    },
): Promise<BulkSeedResult<any>> {
    const issues = [];
    const count = options.count || 1;
    let withTranslationsCount = 0;
    let withLabelsCount = 0;
    let targetedCount = 0;

    for (let i = 0; i < count; i++) {
        let issueData: Prisma.issueCreateInput;

        if (options.forObject) {
            issueData = IssueDbFactory.createForObject(
                options.createdById,
                options.forObject.id,
                options.forObject.type,
                {
                    ...(options.withTranslations && {
                        translations: {
                            create: [{
                                id: generatePK(),
                                language: "en",
                                name: `Issue ${i + 1}`,
                                description: `Description for issue ${i + 1}`,
                            }],
                        },
                    }),
                    ...(options.withLabels && {
                        labels: {
                            connect: options.withLabels.map(id => ({ id })),
                        },
                    }),
                },
            );
            targetedCount++;
        } else {
            issueData = options.withTranslations
                ? IssueDbFactory.createWithTranslations(
                    options.createdById,
                    [{
                        language: "en",
                        name: `Issue ${i + 1}`,
                        description: `Description for issue ${i + 1}`,
                    }],
                    {
                        ...(options.withLabels && {
                            labels: {
                                connect: options.withLabels.map(id => ({ id })),
                            },
                        }),
                    },
                )
                : IssueDbFactory.createMinimal(options.createdById, {
                    ...(options.withLabels && {
                        labels: {
                            connect: options.withLabels.map(id => ({ id })),
                        },
                    }),
                });
        }

        if (options.withTranslations) withTranslationsCount++;
        if (options.withLabels && options.withLabels.length > 0) withLabelsCount++;

        const issue = await prisma.issue.create({
            data: issueData,
            include: { translations: true, labels: true },
        });
        issues.push(issue);
    }

    return {
        records: issues,
        summary: {
            total: issues.length,
            withAuth: 0, // Issues don't have auth
            bots: 0, // Issues don't have bots
            teams: 0, // Issues don't have teams
            withTranslations: withTranslationsCount,
            withLabels: withLabelsCount,
            targeted: targetedCount,
        },
    };
}
