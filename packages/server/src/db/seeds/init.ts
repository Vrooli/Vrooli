/**
 * Adds initial data to the database. (i.e. data that should be included in production). 
 * This is written so that it can be called multiple times without duplicating data.
 */
import { InputType } from '@local/shared';
import { ProfileModel } from '../../models';
import { PrismaType } from '../../types';
import pkg from '@prisma/client';
import { genErrorCode, logger, LogLevel } from '../../logger';
import { v4 as uuid } from 'uuid';
const { AccountStatus, NodeType, ResourceUsedFor, ResourceListUsedFor } = pkg;

export async function init(prisma: PrismaType) {
    console.log('a')
    // TODO currently stick migration code here. Once the production database is updated, this stuff
    // can be removed
    // 1. internalizes standards that start with one of the input types
    const boops = ['Checkbox ', 'Dropzone ', 'JSON ', 'LanguageInput ', 'Markdown ', 'Radio ', 'Selector ', 'Slider ', 'Switch ', 'TagSelector ', 'TextField ', 'QuantityBox '];
    for (let i = 0; i < boops.length; i++) {
        const boop = boops[i];
        await prisma.standard.updateMany({
            where: { name: { startsWith: boop } },
            data: { isInternal: true },
        })
    }
    // 2. For every organization that has no roles, add a new 'Admin' role
    // and apply it to every member
    const orgs = await prisma.organization.findMany({
        select: {
            id: true,
            members: {
                select: {
                    id: true,
                }
            },
            _count: {
                select: {
                    roles: true
                }
            }
        }
    });
    for (let i = 0; i < orgs.length; i++) {
        const org = orgs[i];
        // Create new role titled 'Admin'
        const role = await prisma.role.upsert({
            where: {
                role_organizationId_title_unique: {
                    title: 'Admin',
                    organizationId: org.id,
                }
            },
            update: {},
            create: {
                title: 'Admin',
                organizationId: org.id,
            }
        })
        // Apply role to every member
        for (let j = 0; j < org.members.length; j++) {
            const member = org.members[j];
            const exists = prisma.user_roles.findUnique({
                where: {
                    user_roles_userid_roleid_unique: {
                        userId: member.id,
                        roleId: role.id,
                    }
                }
            })
            if (!Boolean(exists)) {
                await prisma.user_roles.create({
                    data: {
                        userId: member.id,
                        roleId: role.id,
                    }
                })
            }
        }
    }
    // For every routine that has a null versionGroupId, set it to a new ID
    const routinesWithoutVersionGroupId = await prisma.routine.findMany({
        where: {
            versionGroupId: null
        },
        select: {
            id: true,
        }
    })
    for (let i = 0; i < routinesWithoutVersionGroupId.length; i++) {
        await prisma.routine.update({
            where: {
                id: routinesWithoutVersionGroupId[i].id,
            },
            data: {
                versionGroupId: uuid(),
            }
        })
    }
    // For every standard that has a null versionGroupId, set it to a new ID
    const standardsWithoutVersionGroupId = await prisma.standard.findMany({
        where: {
            versionGroupId: null
        },
        select: {
            id: true,
        }
    })
    for (let i = 0; i < standardsWithoutVersionGroupId.length; i++) {
        await prisma.standard.update({
            where: {
                id: standardsWithoutVersionGroupId[i].id,
            },
            data: {
                versionGroupId: uuid(),
            }
        })
    }

    //==============================================================
    /* #region Initialization */
    //==============================================================
    logger.log(LogLevel.info, '🌱 Starting database intial seed...');

    // Check for required .env variables
    if (['ADMIN_WALLET', 'ADMIN_PASSWORD', 'SITE_EMAIL_USERNAME'].some(name => !process.env[name])) {
        logger.log(LogLevel.error, '🚨 Missing required .env variables. Not seeding database.', { code: genErrorCode('0006') });
        return;
    };

    const EN = 'en';

    //==============================================================
    /* #endregion Initialization */
    //==============================================================

    //==============================================================
    /* #region Create Tags */
    //==============================================================
    const tagCardano = await prisma.tag.upsert({
        where: { tag: 'Cardano' },
        update: {},
        create: { tag: 'Cardano' },
    })
    const tagCip = await prisma.tag.upsert({
        where: { tag: 'CIP' },
        update: {},
        create: { tag: 'CIP' },
    })
    const tagEntrepreneurship = await prisma.tag.upsert({
        where: { tag: 'Entrepreneurship' },
        update: {},
        create: { tag: 'Entrepreneurship' },
    })
    const tagVrooli = await prisma.tag.upsert({
        where: { tag: 'Vrooli' },
        update: {},
        create: { tag: 'Vrooli' },
    })
    const tagIdeaValidation = await prisma.tag.upsert({
        where: { tag: 'Idea Validation' },
        update: {},
        create: { tag: 'Idea Validation' },
    })
    const tagLearn = await prisma.tag.upsert({
        where: { tag: 'Learn' },
        update: {},
        create: { tag: 'Learn' },
    })
    const tagResearch = await prisma.tag.upsert({
        where: { tag: 'Research' },
        update: {},
        create: { tag: 'Research' },
    })
    //==============================================================
    /* #endregion Create Tags */
    //==============================================================

    //==============================================================
    /* #region Create Admin */
    //==============================================================
    console.log('h');
    const adminId = '3f038f3b-f8f9-4f9b-8f9b-c8f4b8f9b8d2'
    const admin = await prisma.user.upsert({
        where: {
            id: adminId,
        },
        update: {},
        create: {
            id: adminId,
            name: 'Matt Halloran',
            password: ProfileModel.verify.hashPassword(process.env.ADMIN_PASSWORD ?? ''),
            status: AccountStatus.Unlocked,
            emails: {
                create: [
                    { emailAddress: process.env.SITE_EMAIL_USERNAME ?? '', verified: true },
                ]
            },
            wallets: {
                create: [
                    { stakingAddress: process.env.ADMIN_WALLET ?? '', verified: true } as any,
                ]
            },
            languages: {
                create: [{ language: EN }],
            },
            resourceLists: {
                create: [
                    {
                        usedFor: ResourceListUsedFor.Learn,
                    },
                    {
                        usedFor: ResourceListUsedFor.Research,
                    },
                    {
                        usedFor: ResourceListUsedFor.Develop,
                    },
                    {
                        usedFor: ResourceListUsedFor.Display,
                    }
                ]
            }
        },
    })
    console.log('i');
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
                { translations: { some: { language: EN, name: 'Vrooli' } } },
                { members: { some: { userId: admin.id } } },
            ]
        }
    })
    console.log('j');
    if (!vrooli) {
        logger.log(LogLevel.info, '🏗 Creating Vrooli organization');
        vrooli = await prisma.organization.create({
            data: {
                translations: {
                    create: [
                        {
                            language: EN,
                            name: "Vrooli",
                            bio: "Entrepreneurship is not accessible to all unless it can be accomplished using little money, time, and prior knowledge. Let's make that possible."
                        },
                    ]
                },
                members: {
                    create: [
                        { userId: admin.id },
                    ]
                },
                tags: {
                    create: [
                        { tagTag: tagVrooli.tag },
                        { tagTag: tagEntrepreneurship.tag },
                        { tagTag: tagCardano.tag },
                    ]
                },
                resourceLists: {
                    create: [
                        {
                            usedFor: ResourceListUsedFor.Display,
                            resources: {
                                create: [
                                    {
                                        usedFor: ResourceUsedFor.OfficialWebsite as any,
                                        index: 0,
                                        link: 'https://vrooli.com',
                                        translations: {
                                            create: [{ language: EN, title: "Website", description: "Vrooli's official website" }]
                                        },
                                    },
                                    {
                                        usedFor: ResourceUsedFor.Social as any,
                                        index: 1,
                                        link: 'https://twitter.com/VrooliOfficial',
                                        translations: {
                                            create: [{ language: EN, title: "Twitter", description: "Follow us on Twitter" }]
                                        },
                                    },
                                ]
                            }
                        }
                    ]
                }
            }
        })
        console.log('k');
        const vrooliAdminRole = await prisma.role.upsert({
            where: {
                role_organizationId_title_unique: {
                    title: 'Admin',
                    organizationId: vrooli.id,
                }
            },
            update: {},
            create: {
                title: 'Admin',
                organization: {
                    connect: {
                        id: vrooli.id
                    }
                }
            }
        })
        console.log('l');
        // Assign admin role to admin
        await prisma.user.update({
            where: {
                id: admin.id
            },
            data: {
                roles: {
                    connect: {
                        id: vrooliAdminRole.id
                    }
                }
            }
        })
        console.log('m');
    }
    console.log('n');
    //==============================================================
    /* #endregion Create Organizations */
    //==============================================================

    //==============================================================
    /* #region Create Projects */
    //==============================================================
    let projectEntrepreneur = await prisma.project.findFirst({
        where: {
            AND: [
                { organizationId: vrooli.id },
                { translations: { some: { language: EN, name: 'Project Catalyst Entrepreneur Guide' } } },
            ]
        }
    })
    if (!projectEntrepreneur) {
        logger.log(LogLevel.info, '📚 Creating Project Catalyst Guide project');
        projectEntrepreneur = await prisma.project.create({
            data: {
                translations: {
                    create: [
                        {
                            language: EN,
                            description: "A guide to the best practices and tools for building a successful project on Project Catalyst.",
                            name: "Project Catalyst Entrepreneur Guide",
                        }
                    ]
                },
                createdByOrganizationId: vrooli.id,
                organizationId: vrooli.id,
            }
        })
    }
    //==============================================================
    /* #endregion Create Projects */
    //==============================================================

    //==============================================================
    /* #region Create Standards */
    //==============================================================
    let standardCip0025 = await prisma.standard.findFirst({
        where: {
            AND: [
                { createdByUserId: admin.id },
                { name: 'CIP-0025 - NFT Metadata Standard' },
            ]
        }
    })
    if (!standardCip0025) {
        logger.log(LogLevel.info, '📚 Creating CIP-0025 standard');
        standardCip0025 = await prisma.standard.create({
            data: {
                name: "CIP-0025 - NFT Metadata Standard",
                translations: {
                    create: [
                        {
                            language: EN,
                            description: "A metadata standard for Native Token NFTs on Cardano.",
                        }
                    ]
                },
                createdByUserId: admin.id,
                tags: {
                    create: [
                        { tag: { connect: { id: tagCardano.id } } },
                        { tag: { connect: { id: tagCip.id } } },
                    ]
                },
                type: InputType.JSON,
                props: `{"format":{"<721>":{"<policy_id>":{"<asset_name>":{"name":"<asset_name>","image":"<ipfs_link>","?mediaType":"<mime_type>","?description":"<description>","?files":[{"name":"<asset_name>","mediaType":"<mime_type>","src":"<ipfs_link>"}],"[x]":"[any]"}},"version":"1.0"}},"defaults":[]}`,
            }
        })
    }
    //==============================================================
    /* #endregion Create Standards */
    //==============================================================

    //==============================================================
    /* #region Create Routines */
    //==============================================================
    const mintTokenId = '3f038f3b-f8f9-4f9b-8f9b-f8f9b8f9b8f9'; // <- DO NOT CHANGE. This is used as a reference routine
    let mintToken: any = await prisma.routine.findFirst({
        where: { id: mintTokenId }
    })
    if (!mintToken) {
        logger.log(LogLevel.info, '📚 Creating Native Token Minting routine');
        mintToken = await prisma.routine.create({
            data: {
                id: mintTokenId, // Set ID so we can know ahead of time this routine's URL, and link to it as an example/introductory routine
                translations: {
                    create: [
                        {
                            language: EN,
                            description: 'Mint a fungible token on the Cardano blockchain.',
                            instructions: `To mint through a web interface, select the online resource and follow the instructions.\nTo mint through the command line, select the developer resource and follow the instructions.`,
                            title: 'Mint Native Token',
                        }
                    ]
                },
                complexity: 1,
                simplicity: 1,
                isAutomatable: false,
                isInternal: false,
                version: '1.0.0',
                createdByOrganization: { connect: { id: vrooli.id } },
                organization: { connect: { id: vrooli.id } },
                inputs: {}, //TODO
                outputs: {}, //TODO
                resourceLists: {
                    create: [
                        {
                            usedFor: ResourceListUsedFor.Display,
                            resources: {
                                create: [
                                    {
                                        usedFor: ResourceUsedFor.ExternalService as any,
                                        link: 'https://minterr.io/mint-cardano-tokens/',
                                        translations: {
                                            create: [{ language: EN, title: "minterr.io" }]
                                        },
                                    },
                                    {
                                        usedFor: ResourceUsedFor.Developer as any,
                                        link: 'https://developers.cardano.org/docs/native-tokens/minting/',
                                        translations: {
                                            create: [{ language: EN, title: "cardano.org guide" }]
                                        },
                                    },
                                ]
                            }
                        }
                    ]
                }
            }
        })
    }

    const mintNftId = '4e038f3b-f8f9-4f9b-8f9b-f8f9b8f9b8f9'; // <- DO NOT CHANGE. This is used as a reference routine
    let mintNft: any = await prisma.routine.findFirst({
        where: { id: mintNftId }
    })
    if (!mintNft) {
        logger.log(LogLevel.info, '📚 Creating NFT Minting routine');
        mintNft = await prisma.routine.create({
            data: {
                id: mintNftId, // Set ID so we can know ahead of time this routine's URL, and link to it as an example/introductory routine
                translations: {
                    create: [
                        {
                            language: EN,
                            description: 'Mint a non-fungible token (NFT) on the Cardano blockchain.',
                            instructions: `To mint through a web interface, select one of the online resources and follow the instructions.\nTo mint through the command line, select the developer resource and follow the instructions.`,
                            title: 'Mint NFT',
                        }
                    ]
                },
                complexity: 1,
                simplicity: 1,
                isAutomatable: false,
                isInternal: false,
                version: '1.0.0',
                createdByOrganization: { connect: { id: vrooli.id } },
                organization: { connect: { id: vrooli.id } },
                inputs: {}, //TODO
                outputs: {}, //TODO
                resourceLists: {
                    create: [
                        {
                            usedFor: ResourceListUsedFor.Display,
                            resources: {
                                create: [
                                    {
                                        usedFor: ResourceUsedFor.ExternalService as any,
                                        link: 'https://minterr.io/mint-cardano-tokens/',
                                        translations: {
                                            create: [{ language: EN, title: "minterr.io" }]
                                        },
                                    },
                                    {
                                        usedFor: ResourceUsedFor.ExternalService as any,
                                        link: 'https://cardano-tools.io/mint',
                                        translations: {
                                            create: [{ language: EN, title: "cardano-tools.io" }]
                                        },
                                    },
                                    {
                                        usedFor: ResourceUsedFor.Developer as any,
                                        link: 'https://developers.cardano.org/docs/native-tokens/minting-nfts',
                                        translations: {
                                            create: [{ language: EN, title: "cardano.org guide" }]
                                        },
                                    },
                                ]
                            }
                        }
                    ]
                }
            }
        })
    }

    const frameworkBusinessIdeaId = '5f0f8f9b-f8f9-4f9b-8f9b-f8f9b8f9b8f9'; // <- DO NOT CHANGE. This is used as a reference routine, which is linked in ui
    let frameworkBusinessIdea: any = await prisma.routine.findFirst({
        where: { id: frameworkBusinessIdeaId }
    })
    if (!frameworkBusinessIdea) {
        logger.log(LogLevel.info, '📚 Creating Starting New Business Frameworks');
        const startId = '01234569-7890-1234-5678-901234567890';
        const explainId = '01234569-7890-1234-5678-901234567891';
        const describeId = '01234569-7890-1234-5678-901234567892';
        const validateId = '01234569-7890-1234-5678-901234567893';
        const notWorthItId = '01234569-7890-1234-5678-901234567894';
        const presentationId = '01234569-7890-1234-5678-901234567895';
        const teamId = '01234569-7890-1234-5678-901234567896';
        const ventureId = '01234569-7890-1234-5678-901234567897';
        const financesId = '01234569-7890-1234-5678-901234567898';
        const scaleId = '01234569-7890-1234-5678-901234567899';
        const successEndId = '01234569-7890-1234-5678-901234567900';
        frameworkBusinessIdea = await prisma.routine.create({
            data: {
                id: frameworkBusinessIdeaId, // Set ID so we can know ahead of time this routine's URL, and link to it as an example/introductory routine
                complexity: 16, // 1 + 0 + 2 + 1 + 1 + 2 + 0 + 2 + 3 + 3 + 1 + 0 = 16
                simplicity: 11, // 16 - 2 - 3
                isAutomatable: true,
                isInternal: false,
                version: '1.0.0',
                createdByOrganization: { connect: { id: vrooli.id } },
                organization: { connect: { id: vrooli.id } },
                inputs: {}, //TODO
                outputs: {}, //TODO
                translations: {
                    create: [{
                        language: EN,
                        title: 'Starting New Business Frameworks',
                        description: 'Hash out your new business idea.',
                        instructions: 'Fill out the following forms to collect your thoughts and decide if your business idea is worth pursuing.',
                    }]
                },
                nodeLinks: {
                    create: [
                        {
                            from: { connect: { id: startId } },
                            to: { connect: { id: explainId } },
                        },
                        {
                            from: { connect: { id: explainId } },
                            to: { connect: { id: describeId } },
                        },
                        {
                            from: { connect: { id: describeId } },
                            to: { connect: { id: validateId } },
                        },
                        {
                            from: { connect: { id: validateId } },
                            to: { connect: { id: notWorthItId } },
                        },
                        {
                            from: { connect: { id: validateId } },
                            to: { connect: { id: presentationId } },
                        },
                        {
                            from: { connect: { id: presentationId } },
                            to: { connect: { id: teamId } },
                        },
                        {
                            from: { connect: { id: presentationId } },
                            to: { connect: { id: ventureId } },
                        },
                        {
                            from: { connect: { id: teamId } },
                            to: { connect: { id: ventureId } },
                        },
                        {
                            from: { connect: { id: ventureId } },
                            to: { connect: { id: financesId } },
                        },
                        {
                            from: { connect: { id: ventureId } },
                            to: { connect: { id: scaleId } },
                        },
                        {
                            from: { connect: { id: financesId } },
                            to: { connect: { id: scaleId } },
                        },
                        {
                            from: { connect: { id: scaleId } },
                            to: { connect: { id: successEndId } },
                        },
                    ]
                },
                nodes: {
                    create: [
                        // Start node - complexity = 0, since it has no routines
                        {
                            id: startId,
                            columnIndex: 0,
                            rowIndex: 0,
                            type: NodeType.Start,
                            translations: {
                                create: [{ language: EN, title: 'Start' }]
                            },
                        },
                        // Collect thoughts - complexity = 2, since it has 2 routines
                        {
                            id: explainId,
                            columnIndex: 1,
                            rowIndex: 0,
                            type: NodeType.RoutineList,
                            translations: {
                                create: [{ language: EN, title: 'Explain' }]
                            },
                            nodeRoutineList: {
                                create: {
                                    isOrdered: false,
                                    isOptional: false,
                                    routines: {
                                        create: [
                                            {
                                                index: 0,
                                                routine: {
                                                    create: {
                                                        complexity: 1,
                                                        simplicity: 1,
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
                                                        translations: {
                                                            create: [{
                                                                language: EN,
                                                                title: 'Overview',
                                                                description: 'Hash out your new business idea.',
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        },
                                                    }
                                                }
                                            },
                                            {
                                                index: 1,
                                                routine: {
                                                    create: {
                                                        complexity: 1,
                                                        simplicity: 1,
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
                                                        translations: {
                                                            create: [{
                                                                language: EN,
                                                                title: 'Roadmap',
                                                                description: 'Develop a roadmap for your new business.',
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        },
                                                    }
                                                }
                                            },
                                        ]
                                    }
                                }
                            }
                        },
                        // Describe business - complexity = 1
                        {
                            id: describeId,
                            columnIndex: 2,
                            rowIndex: 0,
                            type: NodeType.RoutineList,
                            translations: {
                                create: [{ language: EN, title: 'Business' }]
                            },
                            nodeRoutineList: {
                                create: {
                                    isOrdered: false,
                                    isOptional: true,
                                    routines: {
                                        create: [
                                            {
                                                index: 0,
                                                routine: {
                                                    create: {
                                                        complexity: 1,
                                                        simplicity: 1,
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
                                                        translations: {
                                                            create: [{
                                                                language: EN,
                                                                title: 'Company',
                                                                description: "Define your company's structure.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        },
                                                    }
                                                }
                                            },
                                        ]
                                    }
                                }
                            }
                        },
                        // Validate idea - complexity = 1
                        {
                            id: validateId,
                            columnIndex: 3,
                            rowIndex: 0,
                            type: NodeType.RoutineList,
                            translations: {
                                create: [{ language: EN, title: 'Validate' }]
                            },
                            nodeRoutineList: {
                                create: {
                                    isOrdered: false,
                                    isOptional: true,
                                    routines: {
                                        create: [
                                            {
                                                index: 0,
                                                routine: {
                                                    create: {
                                                        complexity: 1,
                                                        simplicity: 1,
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
                                                        translations: {
                                                            create: [{
                                                                language: EN,
                                                                title: 'Validation',
                                                                description: "Reflect on your new business idea and decide if it is worth pursuing.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        },
                                                    }
                                                }
                                            },
                                        ]
                                    }
                                }
                            }
                        },
                        // Presentation - complexity = 2
                        {
                            id: presentationId,
                            columnIndex: 4,
                            rowIndex: 0,
                            type: NodeType.RoutineList,
                            translations: {
                                create: [{ language: EN, title: 'Presentation' }]
                            },
                            nodeRoutineList: {
                                create: {
                                    isOrdered: false,
                                    isOptional: true,
                                    routines: {
                                        create: [
                                            {
                                                index: 0,
                                                routine: {
                                                    create: {
                                                        complexity: 1,
                                                        simplicity: 1,
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
                                                        translations: {
                                                            create: [{
                                                                language: EN,
                                                                title: 'Marketing & Sales',
                                                                description: "Define a marketing strategy for your business.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        },
                                                    }
                                                }
                                            },
                                            {
                                                index: 0,
                                                routine: {
                                                    create: {
                                                        complexity: 1,
                                                        simplicity: 1,
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
                                                        translations: {
                                                            create: [{
                                                                language: EN,
                                                                title: 'Pitch Deck',
                                                                description: "Determine key plans and metrics to measure success.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        },
                                                    }
                                                }
                                            },
                                        ]
                                    }
                                }
                            }
                        },
                        // Not worth pursuing end node - complexity = 0
                        {
                            id: notWorthItId,
                            columnIndex: 4,
                            rowIndex: 1,
                            type: NodeType.End,
                            translations: {
                                create: [{ language: EN, title: 'End' }]
                            },
                        },
                        // Team - complexity = 2, but isn't on shortest path
                        {
                            id: teamId,
                            columnIndex: 5,
                            rowIndex: 0,
                            type: NodeType.RoutineList,
                            translations: {
                                create: [{ language: EN, title: 'Team' }]
                            },
                            nodeRoutineList: {
                                create: {
                                    isOrdered: false,
                                    isOptional: true,
                                    routines: {
                                        create: [
                                            {
                                                index: 0,
                                                routine: {
                                                    create: {
                                                        complexity: 1,
                                                        simplicity: 1,
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
                                                        translations: {
                                                            create: [{
                                                                language: EN,
                                                                title: 'Culture',
                                                                description: "Define your team's culture.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        },
                                                    }
                                                }
                                            },
                                            {
                                                index: 1,
                                                routine: {
                                                    create: {
                                                        complexity: 1,
                                                        simplicity: 1,
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
                                                        translations: {
                                                            create: [{
                                                                language: EN,
                                                                title: 'Build Team',
                                                                description: "Build a team to run your business.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        },
                                                    }
                                                }
                                            }
                                        ]
                                    }
                                }
                            }
                        },
                        // Venture - complexity = 3
                        {
                            id: ventureId,
                            columnIndex: 5,
                            rowIndex: 1,
                            type: NodeType.RoutineList,
                            translations: {
                                create: [{ language: EN, title: 'Venture' }]
                            },
                            nodeRoutineList: {
                                create: {
                                    isOrdered: false,
                                    isOptional: true,
                                    routines: {
                                        create: [
                                            {
                                                index: 0,
                                                routine: {
                                                    create: {
                                                        complexity: 1,
                                                        simplicity: 1,
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
                                                        translations: {
                                                            create: [{
                                                                language: EN,
                                                                title: 'Sales Playbook',
                                                                description: "Further defines your sales strategy.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        },
                                                    }
                                                }
                                            },
                                            {
                                                index: 1,
                                                routine: {
                                                    create: {
                                                        complexity: 1,
                                                        simplicity: 1,
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
                                                        translations: {
                                                            create: [{
                                                                language: EN,
                                                                title: 'Product Market Fit',
                                                                description: "Determine your product's market fit.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        },
                                                    }
                                                }
                                            },
                                            {
                                                index: 2,
                                                routine: {
                                                    create: {
                                                        complexity: 1,
                                                        simplicity: 1,
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
                                                        translations: {
                                                            create: [{
                                                                language: EN,
                                                                title: 'Growth State Machine',
                                                                description: "Define a plan to scale your business.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        }
                                                    }
                                                }
                                            },
                                        ]
                                    }
                                }
                            }
                        },
                        // Finances - complexity = 3, but isn't on shortest path
                        {
                            id: financesId,
                            columnIndex: 6,
                            rowIndex: 0,
                            type: NodeType.RoutineList,
                            translations: {
                                create: [{ language: EN, title: 'Finances' }]
                            },
                            nodeRoutineList: {
                                create: {
                                    isOrdered: false,
                                    isOptional: true,
                                    routines: {
                                        create: [
                                            {
                                                index: 0,
                                                routine: {
                                                    create: {
                                                        complexity: 1,
                                                        simplicity: 1,
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
                                                        translations: {
                                                            create: [{
                                                                language: EN,
                                                                title: 'Monetization',
                                                                description: "Determine how your for-profit business will make money.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        }
                                                    }
                                                }
                                            },
                                            {
                                                index: 1,
                                                routine: {
                                                    create: {
                                                        complexity: 1,
                                                        simplicity: 1,
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
                                                        translations: {
                                                            create: [{
                                                                language: EN,
                                                                title: 'Fundraising',
                                                                description: "Set up fundraising.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        }
                                                    }
                                                }
                                            },
                                            {
                                                index: 2,
                                                routine: {
                                                    create: {
                                                        complexity: 1,
                                                        simplicity: 1,
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
                                                        translations: {
                                                            create: [{
                                                                language: EN,
                                                                title: 'Investing',
                                                                description: "Set up investment opportunities.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        }
                                                    }
                                                }
                                            }
                                        ]
                                    }
                                }
                            }
                        },
                        // Scale - complexity = 1
                        {
                            id: scaleId,
                            columnIndex: 6,
                            rowIndex: 1,
                            type: NodeType.RoutineList,
                            translations: {
                                create: [{ language: EN, title: 'Scale' }]
                            },
                            nodeRoutineList: {
                                create: {
                                    isOrdered: false,
                                    isOptional: true,
                                    routines: {
                                        create: [
                                            {
                                                index: 0,
                                                routine: {
                                                    create: {
                                                        complexity: 1,
                                                        simplicity: 1,
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
                                                        translations: {
                                                            create: [{
                                                                language: EN,
                                                                title: 'Scale Playbook',
                                                                description: "Summarize and reformulate key insights you have gained.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        }
                                                    }
                                                }
                                            },
                                        ]
                                    }
                                }
                            }
                        },
                        // Successful end node - complexity = 0
                        {
                            id: successEndId,
                            columnIndex: 7,
                            rowIndex: 0,
                            type: NodeType.End,
                            translations: {
                                create: [{ language: EN, title: 'End' }]
                            },
                        },
                    ]
                }
            }
        })
    }
    //==============================================================
    /* #endregion Create Routines */
    //==============================================================

    logger.log(LogLevel.info, '✅ Database seeding complete.');
}

