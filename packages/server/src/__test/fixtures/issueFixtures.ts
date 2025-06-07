import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Issue model - used for seeding test data
 */

export class IssueDbFactory {
    static createMinimal(
        createdById: string,
        overrides?: Partial<Prisma.IssueCreateInput>
    ): Prisma.IssueCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            createdBy: { connect: { id: createdById } },
            ...overrides,
        };
    }

    static createForObject(
        createdById: string,
        objectId: string,
        objectType: string,
        overrides?: Partial<Prisma.IssueCreateInput>
    ): Prisma.IssueCreateInput {
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
        overrides?: Partial<Prisma.IssueCreateInput>
    ): Prisma.IssueCreateInput {
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
        overrides?: Partial<Prisma.IssueCreateInput>
    ): Prisma.IssueCreateInput {
        return {
            ...this.createMinimal(createdById, overrides),
            labels: {
                connect: labelIds.map(id => ({ id })),
            },
        };
    }
}

/**
 * Helper to seed issues for testing
 */
export async function seedIssues(
    prisma: any,
    options: {
        createdById: string;
        count?: number;
        forObject?: { id: string; type: string };
        withTranslations?: boolean;
        withLabels?: string[];
    }
) {
    const issues = [];
    const count = options.count || 1;

    for (let i = 0; i < count; i++) {
        let issueData: Prisma.IssueCreateInput;

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
                }
            );
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
                    }
                )
                : IssueDbFactory.createMinimal(options.createdById, {
                    ...(options.withLabels && {
                        labels: {
                            connect: options.withLabels.map(id => ({ id })),
                        },
                    }),
                });
        }

        const issue = await prisma.issue.create({
            data: issueData,
            include: { translations: true, labels: true },
        });
        issues.push(issue);
    }

    return issues;
}