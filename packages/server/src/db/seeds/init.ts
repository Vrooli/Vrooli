/**
 * Adds initial data to the database. (i.e. data that should be included in production). 
 * This is written so that it can be called multiple times without duplicating data.
 * 
 * AI_CHECK: TYPE_SAFETY=server-type-safety-maintenance-phase2 | LAST: 2025-07-04 - Added Promise<void> return type annotation to init function
 */
import { API_CREDITS_PREMIUM, AUTH_PROVIDERS, DEFAULT_LANGUAGE, generatePK, generatePublicId, SEEDED_PUBLIC_IDS, SEEDED_TAGS, STANDARD_RESOURCE_QUOTAS, TeamConfig } from "@vrooli/shared";
import type { Prisma } from "@prisma/client";
import pkg, { CreditEntryType, CreditSourceSystem } from "@prisma/client";
import { readManyHelper } from "../../actions/reads.js";
import { PasswordAuthService } from "../../auth/email.js";
import { importData } from "../../builders/importExport.js";
import type { PaginatedSearchResult } from "../../builders/types.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { RunProcessSelect } from "../../tasks/run/process.js";
import { data as resourceData } from "./data/resources.js";
import { data as tags } from "./data/tags.js";

const { PrismaClient } = pkg;

const vrooliHandle = "vrooli";
let adminId: bigint;
let vrooliId: bigint;

async function initTags(client: InstanceType<typeof PrismaClient>) {
    const tagInfo: { tag: string, description: string }[] = tags.map(([tag, description]) => ({ tag, description }));
    for (const tag of tagInfo) {
        await client.tag.upsert({
            where: { tag: tag.tag },
            update: {},
            create: {
                id: generatePK(),
                tag: tag.tag,
                ...(tag.description ? { translations: { create: [{ id: generatePK(), language: DEFAULT_LANGUAGE, description: tag.description }] } } : {}),
            },
        });
    }
}

async function initUsers(client: InstanceType<typeof PrismaClient>) {
    // Admin
    const admin = await client.user.upsert({
        where: {
            publicId: SEEDED_PUBLIC_IDS.Admin,
        },
        update: {
            handle: "matt",
            plan: {
                upsert: {
                    create: {
                        id: generatePK(),
                        enabledAt: new Date(),
                        expiresAt: new Date("2069-04-20"),
                    },
                    update: {
                        enabledAt: new Date(),
                        expiresAt: new Date("2069-04-20"),
                    },
                },
            },
            creditAccount: {
                upsert: {
                    create: {
                        id: generatePK(),
                        currentBalance: API_CREDITS_PREMIUM,
                        entries: {
                            create: {
                                id: generatePK(),
                                idempotencyKey: `admin-initial-credits-${SEEDED_PUBLIC_IDS.Admin}`,
                                amount: API_CREDITS_PREMIUM,
                                type: CreditEntryType.MigrationImport,
                                source: CreditSourceSystem.MigrationScript,
                            },
                        },
                    },
                    update: {
                        currentBalance: API_CREDITS_PREMIUM,
                    },
                },
            },
        },
        create: {
            id: generatePK(),
            publicId: SEEDED_PUBLIC_IDS.Admin,
            handle: "matt",
            name: "Matt Halloran",
            status: "Unlocked",
            auths: {
                create: {
                    id: generatePK(),
                    provider: AUTH_PROVIDERS.Password,
                    hashed_password: PasswordAuthService.hashPassword(process.env.ADMIN_PASSWORD ?? ""),
                },
            },
            emails: {
                create: [
                    {
                        id: generatePK(),
                        emailAddress: process.env.SITE_EMAIL_USERNAME ?? "",
                        verifiedAt: new Date(),
                    },
                ],
            },
            wallets: {
                create: [
                    {
                        id: generatePK(),
                        stakingAddress: process.env.ADMIN_WALLET ?? "",
                        verifiedAt: new Date(),
                    },
                ],
            },
            languages: [DEFAULT_LANGUAGE],
            awards: {
                create: [{
                    id: generatePK(),
                    tierCompletedAt: new Date(),
                    category: "AccountNew",
                    progress: 1,
                }],
            },
            plan: {
                create: {
                    id: generatePK(),
                    enabledAt: new Date(),
                    expiresAt: new Date("2069-04-20"),
                },
            },
            creditAccount: {
                create: {
                    id: generatePK(),
                    currentBalance: API_CREDITS_PREMIUM,
                    entries: {
                        create: {
                            id: generatePK(),
                            idempotencyKey: `admin-initial-credits-${SEEDED_PUBLIC_IDS.Admin}`,
                            amount: API_CREDITS_PREMIUM,
                            type: CreditEntryType.MigrationImport,
                            source: CreditSourceSystem.MigrationScript,
                        },
                    },
                },
            },
        },
    });
    adminId = admin.id;

    // Default AI assistant
    await client.user.upsert({
        where: {
            publicId: SEEDED_PUBLIC_IDS.Valyxa,
        },
        update: {
            handle: "valyxa",
            invitedByUser: { connect: { publicId: SEEDED_PUBLIC_IDS.Admin } },
        },
        create: {
            id: generatePK(),
            publicId: SEEDED_PUBLIC_IDS.Valyxa,
            handle: "valyxa",
            isBot: true,
            name: "Valyxa",
            status: "Unlocked",
            invitedByUser: { connect: { publicId: SEEDED_PUBLIC_IDS.Admin } },
            languages: [DEFAULT_LANGUAGE],
            translations: {
                create: [{
                    id: generatePK(),
                    language: DEFAULT_LANGUAGE,
                    bio: "The official AI assistant for Vrooli. Ask me anything!",
                }],
            },
            auths: {
                create: {
                    id: generatePK(),
                    provider: AUTH_PROVIDERS.Password,
                    hashed_password: PasswordAuthService.hashPassword(process.env.VALYXA_PASSWORD ?? ""),
                },
            },
            awards: {
                create: [{
                    id: generatePK(),
                    tierCompletedAt: new Date(),
                    category: "AccountNew",
                    progress: 1,
                }],
            },
            plan: {
                create: {
                    id: generatePK(),
                    enabledAt: new Date(),
                    expiresAt: new Date("9999-12-31"),
                },
            },
            creditAccount: {
                create: {
                    id: generatePK(),
                    currentBalance: API_CREDITS_PREMIUM,
                    entries: {
                        create: {
                            id: generatePK(),
                            idempotencyKey: `valyxa-initial-credits-${SEEDED_PUBLIC_IDS.Valyxa}`,
                            amount: API_CREDITS_PREMIUM,
                            type: CreditEntryType.MigrationImport,
                            source: CreditSourceSystem.MigrationScript,
                        },
                    },
                },
            },
        },
    });
}

async function initTeams(client: InstanceType<typeof PrismaClient>) {
    let vrooli = await client.team.findFirst({
        where: {
            publicId: SEEDED_PUBLIC_IDS.Vrooli,
        },
    });
    if (!vrooli) {
        logger.info("🏗 Creating Vrooli team");
        vrooli = await client.team.create({
            data: {
                id: generatePK(),
                publicId: SEEDED_PUBLIC_IDS.Vrooli,
                config: (new TeamConfig({
                    config: {
                        __version: "1.0",
                        deploymentType: "saas",
                        goal: "Official Vrooli platform team - manage platform development, provide support, and foster community growth",
                        businessPrompt: "You are the official Vrooli team. Focus on platform stability, user support, community building, and sustainable growth. Prioritize open-source development and user empowerment.",
                        resourceQuota: STANDARD_RESOURCE_QUOTAS.standard,
                        targetProfitPerMonth: "0", // Non-profit platform team
                        stats: {
                            totalInstances: 0,
                            totalProfit: "0",
                            totalCosts: "0",
                            averageKPIs: {},
                            activeInstances: 0,
                            lastUpdated: Date.now(),
                        },
                        resources: [
                            {
                                link: "https://vrooli.com",
                                usedFor: "OfficialWebsite",
                                translations: [{
                                    language: DEFAULT_LANGUAGE,
                                    name: "Website",
                                    description: "Vrooli's official website",
                                }],
                            },
                            {
                                link: "https://x.com/VrooliOfficial",
                                usedFor: "Social",
                                translations: [{
                                    language: DEFAULT_LANGUAGE,
                                    name: "X",
                                    description: "Follow us on X",
                                }],
                            },
                        ],
                    },
                })).export() as unknown as Prisma.InputJsonValue,
                handle: vrooliHandle,
                createdBy: { connect: { publicId: SEEDED_PUBLIC_IDS.Admin } },
                translations: {
                    create: [
                        {
                            id: generatePK(),
                            language: DEFAULT_LANGUAGE,
                            name: "Vrooli",
                            bio: "Building an automated, self-improving productivity assistant",
                        },
                    ],
                },
                permissions: JSON.stringify({}),
                members: {
                    create: [
                        {
                            id: generatePK(),
                            isAdmin: true,
                            permissions: JSON.stringify({}),
                            publicId: generatePublicId(),
                            user: { connect: { publicId: SEEDED_PUBLIC_IDS.Admin } },
                        },
                    ],
                },
                tags: {
                    create: [
                        { id: generatePK(), tag: { connect: { tag: SEEDED_TAGS.Vrooli } } },
                        { id: generatePK(), tag: { connect: { tag: SEEDED_TAGS.Ai } } },
                        { id: generatePK(), tag: { connect: { tag: SEEDED_TAGS.Automation } } },
                        { id: generatePK(), tag: { connect: { tag: SEEDED_TAGS.Collaboration } } },
                    ],
                },
            },
        });
    }
    vrooliId = vrooli.id;
}

export async function init(client: InstanceType<typeof PrismaClient>): Promise<void> {
    logger.info("🌱 Starting database initial seed...");
    // Check for required .env variables
    if (["ADMIN_WALLET", "ADMIN_PASSWORD", "SITE_EMAIL_USERNAME", "VALYXA_PASSWORD"].some(name => !process.env[name])) {
        logger.error("🚨 Missing required .env variables. Not seeding database.", { trace: "0006" });
        return;
    }

    const importDataBase = {
        __exportedAt: new Date().toISOString(),
        __signature: "",
        __source: "vrooli",
        __version: "1.0.0",
    } as const;

    // Order matters here. Some objects depend on others.
    await initTags(client);
    await initUsers(client);
    await initTeams(client);

    // Debug: show seeded IDs from init Users/Teams
    logger.info(`Seed IDs: adminId=${adminId}, vrooliId=${vrooliId}`);
    // Ensure IDs are defined before string conversion
    if (adminId == null || vrooliId == null) {
        logger.error("Cannot build userData/importConfig: adminId or vrooliId is null or undefined", { adminId, vrooliId });
        throw new Error(`Seed init failed: adminId=${adminId}, vrooliId=${vrooliId}`);
    }

    // Build userData for import
    const userData = {
        id: adminId.toString(),
        publicId: SEEDED_PUBLIC_IDS.Admin,
        languages: [DEFAULT_LANGUAGE],
    };
    // Build import configuration for initial data
    const importConfig = {
        allowForeignData: true, // Skip foreign data checks
        assignObjectsTo: { __typename: "Team" as const, id: vrooliId.toString() }, // Assign to Vrooli team
        isSeeding: true,
        onConflict: "overwrite" as const, //TODO need update option
        skipPermissions: true, // Skip permission checks
        userData, // Set user data
    };

    await importData({
        ...importDataBase,
        data: [
            ...resourceData,
        ],
    }, importConfig);

    // Cache seeded resource versions for performance using CacheService
    {
        const cacheService = CacheService.get();
        const info = RunProcessSelect.ResourceVersion;
        const req = {
            session: {
                apiToken: undefined,
                fromSafeOrigin: true,
                isLoggedIn: true,
                languages: [DEFAULT_LANGUAGE],
                users: [userData],
            },
        };
        const baseInputFiltered = { isLatestPublic: true, ownedByUserIdRoot: userData.id };
        let hasNextPage: boolean;
        let afterCursor: string | undefined;
        do {
            const input = afterCursor ? { ...baseInputFiltered, after: afterCursor } : baseInputFiltered;
            const page: PaginatedSearchResult = await readManyHelper({ info, input, objectType: "ResourceVersion", req });
            for (const { node: rv } of page.edges) {
                await cacheService.set(`seeded:resourceVersion:${rv.id}`, JSON.stringify(rv));
            }
            hasNextPage = page.pageInfo.hasNextPage;
            afterCursor = page.pageInfo.endCursor ?? undefined;
        } while (hasNextPage);
    }

    logger.info("✅ Database seeding complete.");
}

