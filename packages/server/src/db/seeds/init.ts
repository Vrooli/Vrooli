/**
 * Adds initial data to the database. (i.e. data that should be included in production). 
 * This is written so that it can be called multiple times without duplicating data.
 */
import { InputType, uuid } from "@local/shared";
import { hashPassword } from "../../auth";
import { logger } from "../../events/logger";
import { PrismaType } from "../../types";

export async function init(prisma: PrismaType) {
    //==============================================================
    /* #region Initialization */
    //==============================================================
    logger.info("🌱 Starting database initial seed...");
    // Check for required .env variables
    if (["ADMIN_WALLET", "ADMIN_PASSWORD", "SITE_EMAIL_USERNAME"].some(name => !process.env[name])) {
        logger.error("🚨 Missing required .env variables. Not seeding database.", { trace: "0006" });
        return;
    }

    const EN = "en";

    //==============================================================
    /* #endregion Initialization */
    //==============================================================

    //==============================================================
    /* #region Create Tags */
    //==============================================================
    const tagCardano = await prisma.tag.upsert({
        where: { tag: "Cardano" },
        update: {},
        create: { tag: "Cardano" },
    });
    const tagCip = await prisma.tag.upsert({
        where: { tag: "CIP" },
        update: {},
        create: { tag: "CIP" },
    });
    const tagEntrepreneurship = await prisma.tag.upsert({
        where: { tag: "Entrepreneurship" },
        update: {},
        create: { tag: "Entrepreneurship" },
    });
    const tagVrooli = await prisma.tag.upsert({
        where: { tag: "Vrooli" },
        update: {},
        create: { tag: "Vrooli" },
    });
    const tagIdeaValidation = await prisma.tag.upsert({
        where: { tag: "Idea Validation" },
        update: {},
        create: { tag: "Idea Validation" },
    });
    const tagLearn = await prisma.tag.upsert({
        where: { tag: "Learn" },
        update: {},
        create: { tag: "Learn" },
    });
    const tagResearch = await prisma.tag.upsert({
        where: { tag: "Research" },
        update: {},
        create: { tag: "Research" },
    });
    //==============================================================
    /* #endregion Create Tags */
    //==============================================================

    //==============================================================
    /* #region Create Admin */
    //==============================================================
    const adminId = "3f038f3b-f8f9-4f9b-8f9b-c8f4b8f9b8d2";
    const admin = await prisma.user.upsert({
        where: {
            id: adminId,
        },
        update: {},
        create: {
            id: adminId,
            name: "Matt Halloran",
            password: hashPassword(process.env.ADMIN_PASSWORD ?? ""),
            reputation: 1000000, // TODO temporary until community grows
            status: "Unlocked",
            emails: {
                create: [
                    { emailAddress: process.env.SITE_EMAIL_USERNAME ?? "", verified: true },
                ],
            },
            wallets: {
                create: [
                    { stakingAddress: process.env.ADMIN_WALLET ?? "", verified: true } as any,
                ],
            },
            languages: {
                create: [{ language: EN }],
            },
            focusModes: {
                create: [{
                    name: "Work",
                    description: "This is an auto-generated focus mode. You can edit or delete it.",
                    reminderList: { create: {} },
                    resourceList: { create: {} },
                }, {
                    name: "Study",
                    description: "This is an auto-generated focus mode. You can edit or delete it.",
                    reminderList: { create: {} },
                    resourceList: { create: {} },
                }],
            },
            awards: {
                create: [{
                    timeCurrentTierCompleted: new Date(),
                    category: "AccountNew",
                    progress: 1,
                }],
            },
        },
    });
    //==============================================================
    /* #endregion Create Admin */
    //==============================================================

    //==============================================================
    /* #endregion Create Resources */
    //==============================================================

    //==============================================================
    /* #region Create Organizations */
    //==============================================================
    let vrooli = await prisma.organization.findFirst({
        where: {
            AND: [
                { translations: { some: { language: EN, name: "Vrooli" } } },
                { members: { some: { userId: admin.id } } },
            ],
        },
    });
    if (!vrooli) {
        logger.info("🏗 Creating Vrooli organization");
        const organizationId = uuid();
        vrooli = await prisma.organization.create({
            data: {
                id: organizationId,
                createdBy: { connect: { id: admin.id } },
                translations: {
                    create: [
                        {
                            language: EN,
                            name: "Vrooli",
                            bio: "Entrepreneurship is not accessible to all unless it can be accomplished using little money, time, and prior knowledge. Let's make that possible.",
                        },
                    ],
                },
                permissions: JSON.stringify({}),
                roles: {
                    create: {
                        name: "Admin",
                        permissions: JSON.stringify({}),
                        members: {
                            create: [
                                {
                                    isAdmin: true,
                                    permissions: JSON.stringify({}),
                                    user: { connect: { id: admin.id } },
                                    organization: { connect: { id: organizationId } },
                                },
                            ],
                        },
                    },
                },
                tags: {
                    create: [
                        { tagTag: tagVrooli.tag },
                        { tagTag: tagEntrepreneurship.tag },
                        { tagTag: tagCardano.tag },
                    ],
                },
                resourceList: {
                    create: {
                        resources: {
                            create: [
                                {
                                    usedFor: "OfficialWebsite",
                                    index: 0,
                                    link: "https://vrooli.com",
                                    translations: {
                                        create: [{ language: EN, name: "Website", description: "Vrooli's official website" }],
                                    },
                                },
                                {
                                    usedFor: "Social",
                                    index: 1,
                                    link: "https://twitter.com/VrooliOfficial",
                                    translations: {
                                        create: [{ language: EN, name: "Twitter", description: "Follow us on Twitter" }],
                                    },
                                },
                            ],
                        },
                    },
                },
            },
        });
    }
    //==============================================================
    /* #endregion Create Organizations */
    //==============================================================

    //==============================================================
    /* #region Create Projects */
    //==============================================================
    let projectEntrepreneur = await prisma.project_version.findFirst({
        where: {
            AND: [
                { root: { ownedByOrganizationId: vrooli.id } },
                { translations: { some: { language: EN, name: "Project Catalyst Entrepreneur Guide" } } },
            ],
        },
    });
    if (!projectEntrepreneur) {
        logger.info("📚 Creating Project Catalyst Guide project");
        projectEntrepreneur = await prisma.project_version.create({
            data: {
                translations: {
                    create: [
                        {
                            language: EN,
                            description: "A guide to the best practices and tools for building a successful project on Project Catalyst.",
                            name: "Project Catalyst Entrepreneur Guide",
                        },
                    ],
                },
                root: {
                    create: {
                        permissions: JSON.stringify({}),
                        createdBy: { connect: { id: admin.id } },
                        ownedByOrganization: { connect: { id: vrooli.id } },
                    },
                },
            },
        });
    }

    // TODO temporary
    // Add 100 dummy projects
    if (process.env.NODE_ENV === "development") {
        const dummy1 = await prisma.project.findFirst({
            where: {
                AND: [
                    { ownedByOrganizationId: vrooli.id },
                    { versions: { some: { translations: { some: { language: EN, name: "DUMMY 1" } } } } },
                ],
            },
        });
        if (!dummy1) {
            for (let i = 0; i < 100; i++) {
                logger.info("📚 Creating DUMMY project" + i);
                await prisma.project.create({
                    data: {
                        permissions: JSON.stringify({}),
                        createdBy: { connect: { id: admin.id } },
                        ownedByOrganization: { connect: { id: vrooli.id } },
                        versions: {
                            create: [{
                                isComplete: true,
                                isLatest: true,
                                versionIndex: 0,
                                versionLabel: "1.0.0",
                                translations: {
                                    create: [
                                        {
                                            language: EN,
                                            description: `This is the first description for DUMMY ${i}`,
                                            name: `DUMMY ${i}`,
                                        },
                                    ],
                                },
                            }, {
                                isComplete: false,
                                versionIndex: 1,
                                versionLabel: "1.0.1",
                                translations: {
                                    create: [
                                        {
                                            language: EN,
                                            description: `This is the second description for DUMMY ${i}`,
                                            name: `DUMMY ${i}`,
                                        },
                                    ],
                                },
                            }],
                        },
                    },
                });
            }
        }
    }

    //==============================================================
    /* #endregion Create Projects */
    //==============================================================

    //==============================================================
    /* #region Create Standards */
    //==============================================================
    const standardCip0025Id = "3a038a3b-f8a9-4fab-8fab-c8a4baaab8d2";
    let standardCip0025 = await prisma.standard_version.findFirst({
        where: {
            id: standardCip0025Id,
        },
    });
    if (!standardCip0025) {
        logger.info("📚 Creating CIP-0025 standard");
        standardCip0025 = await prisma.standard_version.create({
            data: {
                root: {
                    create: {
                        id: uuid(),
                        permissions: JSON.stringify({}),
                        createdById: admin.id,
                        tags: {
                            create: [
                                { tag: { connect: { id: tagCardano.id } } },
                                { tag: { connect: { id: tagCip.id } } },
                            ],
                        },
                    },
                },
                translations: {
                    create: [
                        {
                            language: EN,
                            name: "CIP-0025 - NFT Metadata Standard",
                            description: "A metadata standard for Native Token NFTs on Cardano.",
                        },
                    ],
                },
                versionLabel: "1.0.0",
                versionIndex: 0,
                standardType: InputType.JSON,
                props: "{\"format\":{\"<721>\":{\"<policy_id>\":{\"<asset_name>\":{\"name\":\"<asset_name>\",\"image\":\"<ipfs_link>\",\"?mediaType\":\"<mime_type>\",\"?description\":\"<description>\",\"?files\":[{\"name\":\"<asset_name>\",\"mediaType\":\"<mime_type>\",\"src\":\"<ipfs_link>\"}],\"[x]\":\"[any]\"}},\"version\":\"1.0\"}},\"defaults\":[]}",
            },
        });
    }
    //==============================================================
    /* #endregion Create Standards */
    //==============================================================

    //==============================================================
    /* #region Create Routines */
    //==============================================================
    const mintTokenId = "3f038f3b-f8f9-4f9b-8f9b-f8f9b8f9b8f9"; // <- DO NOT CHANGE. This is used as a reference routine
    let mintToken: any = await prisma.routine.findFirst({
        where: { id: mintTokenId },
    });
    if (!mintToken) {
        logger.info("📚 Creating Native Token Minting routine");
        mintToken = await prisma.routine_version.create({
            data: {
                root: {
                    create: {
                        id: mintTokenId, // Set ID so we can know ahead of time this routine's URL, and link to it as an example/introductory routine
                        permissions: JSON.stringify({}),
                        isInternal: false,
                        createdBy: { connect: { id: admin.id } },
                        ownedByOrganization: { connect: { id: vrooli.id } },
                    },
                },
                translations: {
                    create: [
                        {
                            language: EN,
                            description: "Mint a fungible token on the Cardano blockchain.",
                            instructions: "To mint through a web interface, select the online resource and follow the instructions.\nTo mint through the command line, select the developer resource and follow the instructions.",
                            name: "Mint Native Token",
                        },
                    ],
                },
                complexity: 1,
                simplicity: 1,
                isAutomatable: false,
                versionLabel: "1.0.0",
                versionIndex: 0,
                resourceList: {
                    create: {
                        resources: {
                            create: [
                                {
                                    usedFor: "ExternalService",
                                    link: "https://minterr.io/mint-cardano-tokens/",
                                    translations: {
                                        create: [{ language: EN, name: "minterr.io" }],
                                    },
                                },
                                {
                                    usedFor: "Developer",
                                    link: "https://developers.cardano.org/docs/native-tokens/minting/",
                                    translations: {
                                        create: [{ language: EN, name: "cardano.org guide" }],
                                    },
                                },
                            ],
                        },
                    },
                },
            },
        });
    }

    const mintNftId = "4e038f3b-f8f9-4f9b-8f9b-f8f9b8f9b8f9"; // <- DO NOT CHANGE. This is used as a reference routine
    let mintNft: any = await prisma.routine.findFirst({
        where: { id: mintNftId },
    });
    if (!mintNft) {
        logger.info("📚 Creating NFT Minting routine");
        mintNft = await prisma.routine_version.create({
            data: {
                root: {
                    create: {
                        id: mintNftId, // Set ID so we can know ahead of time this routine's URL, and link to it as an example/introductory routine
                        permissions: JSON.stringify({}),
                        isInternal: false,
                        createdBy: { connect: { id: admin.id } },
                        ownedByOrganization: { connect: { id: vrooli.id } },
                    },
                },
                translations: {
                    create: [
                        {
                            language: EN,
                            description: "Mint a non-fungible token (NFT) on the Cardano blockchain.",
                            instructions: "To mint through a web interface, select one of the online resources and follow the instructions.\nTo mint through the command line, select the developer resource and follow the instructions.",
                            name: "Mint NFT",
                        },
                    ],
                },
                complexity: 1,
                simplicity: 1,
                isAutomatable: false,
                versionLabel: "1.0.0",
                versionIndex: 0,
                resourceList: {
                    create: {
                        resources: {
                            create: [
                                {
                                    usedFor: "ExternalService",
                                    link: "https://minterr.io/mint-cardano-tokens/",
                                    translations: {
                                        create: [{ language: EN, name: "minterr.io" }],
                                    },
                                },
                                {
                                    usedFor: "ExternalService",
                                    link: "https://cardano-tools.io/mint",
                                    translations: {
                                        create: [{ language: EN, name: "cardano-tools.io" }],
                                    },
                                },
                                {
                                    usedFor: "Developer",
                                    link: "https://developers.cardano.org/docs/native-tokens/minting-nfts",
                                    translations: {
                                        create: [{ language: EN, name: "cardano.org guide" }],
                                    },
                                },
                            ],
                        },
                    },
                },
            },
        });
    }

    //==============================================================
    /* #endregion Create Routines */
    //==============================================================

    logger.info("✅ Database seeding complete.");
}

