/**
 * Adds initial data to the database. (i.e. data that should be included in production). 
 * This is written so that it can be called multiple times without duplicating data.
 */
import { ROLES, StandardType } from '@local/shared';
import { ProfileModel } from '../../models';
import { envVariableExists } from '../../utils';
import { PrismaType } from '../../types';
import pkg from '@prisma/client';
const { AccountStatus, MemberRole, NodeType, ResourceUsedFor } = pkg;

export async function init(prisma: PrismaType) {

    //==============================================================
    /* #region Initialization */
    //==============================================================
    console.info('🌱 Starting database intial seed...');

    // Check for required .env variables
    if (['ADMIN_WALLET', 'SITE_EMAIL_USERNAME'].some(name => !envVariableExists(name))) {
        console.error('🚨 Missing required .env variables. Not seeding database.');
        return;
    };

    const EN = 'en';

    // Initialize models
    const profileModel = ProfileModel(prisma);
    //==============================================================
    /* #endregion Initialization */
    //==============================================================

    //==============================================================
    /* #region Create Roles */
    //==============================================================
    const actorRole = await prisma.role.upsert({
        where: { title: ROLES.Actor },
        update: {},
        create: { title: ROLES.Actor, description: 'This role allows a user to create routines and save their progress.' },
    })
    //==============================================================
    /* #endregion Create Roles*/
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
    const admin = await prisma.user.upsert({
        where: {
            username: 'admin',
        },
        update: {},
        create: {
            username: 'admin',
            password: profileModel.hashPassword('admin'),
            status: AccountStatus.Unlocked,
            emails: {
                create: [
                    { emailAddress: process.env.SITE_EMAIL_USERNAME ?? '', verified: true },
                ]
            },
            wallets: {
                create: [
                    { publicAddress: process.env.ADMIN_WALLET ?? '', verified: true },
                ]
            },
            roles: {
                create: [{ roleId: actorRole.id }]
            },
            languages: {
                create: [{ language: EN }],
            },
        },
    })
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
    if (!vrooli) {
        console.info('🏗 Creating Vrooli organization');
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
                        { userId: admin.id, role: MemberRole.Owner },
                    ]
                },
                tags: {
                    create: [
                        { tagId: tagVrooli.id },
                        { tagId: tagEntrepreneurship.id },
                        { tagId: tagCardano.id },
                    ]
                },
                resources: {
                    create: [
                        {
                            usedFor: ResourceUsedFor.Social as any,
                            link: 'https://twitter.com/VrooliOfficial',
                            translations: {
                                create: [{ language: EN, description: "Twitter" }]
                            },
                        },
                        {
                            usedFor: ResourceUsedFor.OfficialWebsite as any,
                            link: 'https://vrooli.com',
                            translations: {
                                create: [{ language: EN, description: "Website" }]
                            },
                        },
                    ]
                }
            }
        })
    }
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
        console.info('📚 Creating Project Catalyst Guide project');
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
        console.info('📚 Creating CIP-0025 standard');
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
                type: StandardType.Object,
                schema: `
                    { TODO
                        "721": {
                            "<policy_id>": {
                                "<asset_name>": {
                                    "name": {
                                        "type": "string",
                                        "checks": [
                                            {
                                                "key": "minLength",
                                                "val": 1,
                                                "err": "is invalid"
                                            }
                                        ]
                                    },
                                    "image": {
                                        "type": "string",
                                        "checks": [
                                            {
                                                "key": "minLength",
                                                "val": 1,
                                                "err": "is invalid"
                                            }
                                        ]
                                    },
                                    "mediaType": "image/<mime_sub_type>",
                            
                                    "description": <string | array>,
                            
                                    "files": [{
                                        "name": <string>,
                                        "mediaType": <mime_type>,
                                        "src": <uri | array>,
                                        <other_properties>
                                    }],
                            
                                    <other properties>
                                }
                            },
                            "version": "1.0"
                        }
                    }
                `
            }
        })
    }
    //==============================================================
    /* #endregion Create Standards */
    //==============================================================

    //==============================================================
    /* #region Create Routines */
    //==============================================================
    let mintToken: any = await prisma.routine.findFirst({
        where: {
            AND: [
                { organizationId: vrooli.id },
                { translations: { some: { language: EN, title: 'Mint Native Token' } } },
            ]
        }
    })
    if (!mintToken) {
        console.info('📚 Creating Native Token Minting routine');
        const mintTokenId = '3f038f3b-f8f9-4f9b-8f9b-f8f9b8f9b8f9'; // <- DO NOT CHANGE. This is used as a reference routine
        mintToken = await prisma.routine.create({
            data: {
                id: mintTokenId, // Set ID so we can know ahead of time this routine's URL, and link to it as an example/introductory routine
                translations: {
                    create: [
                        {
                            language: EN,
                            description: 'Mint a fungible token on the Cardano blockchain.',
                            instructions: `To mint through a web interface, select the online resource and follow the instructions.\n
                            To mint through the command line, select the developer resource and follow the instructions.`,
                            title: 'Mint Native Token',
                        }
                    ]
                },
                isAutomatable: false,
                isInternal: false,
                version: '1.0.0',
                createdByOrganization: { connect: { id: vrooli.id } },
                organization: { connect: { id: vrooli.id } },
                inputs: {}, //TODO
                outputs: {}, //TODO
                resources: {
                    create: [
                        {
                            usedFor: ResourceUsedFor.ExternalService as any,
                            link: 'https://minterr.io/mint-cardano-tokens/',
                            translations: {
                                create: [{ language: EN, description: "minterr.io" }]
                            },
                        },
                        {
                            usedFor: ResourceUsedFor.Developer as any,
                            link: 'https://developers.cardano.org/docs/native-tokens/minting/',
                            translations: {
                                create: [{ language: EN, description: "cardano.org guide" }]
                            },
                        },
                    ]
                }
            }
        })
    }

    let mintNft: any = await prisma.routine.findFirst({
        where: {
            AND: [
                { organizationId: vrooli.id },
                { translations: { some: { language: EN, title: 'Mint NFT' } } },
            ]
        }
    })
    if (!mintNft) {
        console.info('📚 Creating NFT Minting routine');
        const mintNftId = '4e038f3b-f8f9-4f9b-8f9b-f8f9b8f9b8f9'; // <- DO NOT CHANGE. This is used as a reference routine
        mintNft = await prisma.routine.create({
            data: {
                id: mintNftId, // Set ID so we can know ahead of time this routine's URL, and link to it as an example/introductory routine
                translations: {
                    create: [
                        {
                            language: EN,
                            description: 'Mint a non-fungible token (NFT) on the Cardano blockchain.',
                            instructions: `To mint through a web interface, select one of the online resources and follow the instructions.\n
                            To mint through the command line, select the developer resource and follow the instructions.`,
                            title: 'Mint NFT',
                        }
                    ]
                },
                isAutomatable: false,
                isInternal: false,
                version: '1.0.0',
                createdByOrganization: { connect: { id: vrooli.id } },
                organization: { connect: { id: vrooli.id } },
                inputs: {}, //TODO
                outputs: {}, //TODO
                resources: {
                    create: [
                        {
                            usedFor: ResourceUsedFor.ExternalService as any,
                            link: 'https://minterr.io/mint-cardano-tokens/',
                            translations: {
                                create: [{ language: EN, description: "minterr.io" }]
                            },
                        },
                        {
                            usedFor: ResourceUsedFor.ExternalService as any,
                            link: 'https://cardano-tools.io/mint',
                            translations: {
                                create: [{ language: EN, description: "cardano-tools.io" }]
                            },
                        },
                        {
                            usedFor: ResourceUsedFor.Developer as any,
                            link: 'https://developers.cardano.org/docs/native-tokens/minting-nfts',
                            translations: {
                                create: [{ language: EN, description: "cardano.org guide" }]
                            },
                        },
                    ]
                }
            }
        })
    }

    let frameworkBusinessIdea: any = await prisma.routine.findFirst({
        where: {
            AND: [
                { organizationId: vrooli.id },
                { translations: { some: { language: EN, title: 'Starting New Business Frameworks' } } },
            ]
        }
    })
    if (!frameworkBusinessIdea) {
        console.info('📚 Creating Starting New Business Frameworks');
        const frameworkBusinessIdeaId = '5f0f8f9b-f8f9-4f9b-8f9b-f8f9b8f9b8f9'; // <- DO NOT CHANGE. This is used as a reference routine, which is linked in ui
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
                }, //TODO
                nodes: {
                    create: [
                        // Start node
                        {
                            id: startId,
                            columnIndex: 0,
                            rowIndex: 0,
                            type: NodeType.Start,
                            translations: {
                                create: [{ language: EN, title: 'Start' }]
                            },
                        },
                        // Collect thoughts
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
                                                routine: {
                                                    create: {
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
                                                routine: {
                                                    create: {
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
                        // Describe business
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
                                                routine: {
                                                    create: {
                                                        translations: {
                                                            create: [{ 
                                                                language: EN, 
                                                                title: 'Company',
                                                                description: "Define your company's structure.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        },
                                                        //TODO
                                                    }
                                                }
                                            },
                                        ]
                                    }
                                }
                            }
                        },
                        // Validate idea
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
                                                routine: {
                                                    create: {
                                                        translations: {
                                                            create: [{ 
                                                                language: EN, 
                                                                title: 'Validation',
                                                                description: "Reflect on your new business idea and decide if it is worth pursuing.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        },
                                                        //TODO
                                                    }
                                                }
                                            },
                                        ]
                                    }
                                }
                            }
                        },
                        // Presentation
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
                                                routine: {
                                                    create: {
                                                        translations: {
                                                            create: [{ 
                                                                language: EN, 
                                                                title: 'Marketing & Sales',
                                                                description: "Define a marketing strategy for your business.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        },
                                                        //TODO
                                                    }
                                                }
                                            },
                                            {
                                                routine: {
                                                    create: {
                                                        translations: {
                                                            create: [{ 
                                                                language: EN, 
                                                                title: 'Pitch Deck',
                                                                description: "Determine key plans and metrics to measure success.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        },
                                                        //TODO
                                                    }
                                                }
                                            },
                                        ]
                                    }
                                }
                            }
                        },
                        // Not worth pursuing end node
                        {
                            id: notWorthItId,
                            columnIndex: 4,
                            rowIndex: 1,
                            type: NodeType.End,
                            translations: {
                                create: [{ language: EN, title: 'End' }]
                            },
                        },
                        // Team
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
                                                routine: {
                                                    create: {
                                                        translations: {
                                                            create: [{ 
                                                                language: EN, 
                                                                title: 'Culture',
                                                                description: "Define your team's culture.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        },
                                                        //TODO
                                                    }
                                                }
                                            },
                                            {
                                                routine: {
                                                    create: {
                                                        translations: {
                                                            create: [{ 
                                                                language: EN, 
                                                                title: 'Build Team',
                                                                description: "Build a team to run your business.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        },
                                                        //TODO
                                                    }
                                                }
                                            }
                                        ]
                                    }
                                }
                            }
                        },
                        // Venture
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
                                                routine: {
                                                    create: {
                                                        translations: {
                                                            create: [{ 
                                                                language: EN, 
                                                                title: 'Sales Playbook',
                                                                description: "Further defines your sales strategy.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        },
                                                        //TODO
                                                    }
                                                }
                                            },
                                            {
                                                routine: {
                                                    create: {
                                                        translations: {
                                                            create: [{ 
                                                                language: EN, 
                                                                title: 'Product Market Fit',
                                                                description: "Determine your product's market fit.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        },
                                                        //TODO
                                                    }
                                                }
                                            },
                                            {
                                                routine: {
                                                    create: {
                                                        translations: {
                                                            create: [{ 
                                                                language: EN, 
                                                                title: 'Growth State Machine',
                                                                description: "Define a plan to scale your business.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        }
                                                        //TODO
                                                    }
                                                }
                                            },
                                        ]
                                    }
                                }
                            }
                        },
                        // Finances
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
                                                routine: {
                                                    create: {
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
                                                routine: {
                                                    create: {
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
                                                routine: {
                                                    create: {
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
                        // Scale
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
                                                routine: {
                                                    create: {
                                                        translations: {
                                                            create: [{ 
                                                                language: EN, 
                                                                title: 'Scale Playbook',
                                                                description: "Summarize and reformulate key insights you have gained.",
                                                                instructions: 'Fill out the form below.',
                                                            }]
                                                        }
                                                        //TODO
                                                    }
                                                }
                                            },
                                        ]
                                    }
                                }
                            }
                        },
                        // Successful end node
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
        console.log('frameworkBusinessIdea', frameworkBusinessIdea)
    }
    //==============================================================
    /* #endregion Create Routines */
    //==============================================================

    console.info(`✅ Database seeding complete.`);
}

