/**
 * Adds initial data to the database. (i.e. data that should be included in production). 
 * This is written so that it can be called multiple times without duplicating data.
 */
import { ROLES, StandardType } from '@local/shared';
import { ProfileModel } from '../../models';
import { envVariableExists } from '../../utils';
import { PrismaType } from '../../types';
import pkg from '@prisma/client';
const { AccountStatus, NodeType, ResourceUsedFor } = pkg;

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
            }
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
                { name: 'Vrooli' },
                { members: { some: { userId: admin.id } } },
            ]
        }
    })
    if (!vrooli) {
        console.info('🏗 Creating Vrooli organization');
        vrooli = await prisma.organization.create({
            data: {
                name: "Vrooli",
                bio: "Entrepreneurship is not accessible to all unless it can be accomplished using little money, time, and prior knowledge. Let's make that possible.",
                members: {
                    create: [
                        { userId: admin.id },
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
                        { title: 'Twitter', usedFor: ResourceUsedFor.Social as any, link: 'https://twitter.com/VrooliOfficial' },
                        { title: 'Website', usedFor: ResourceUsedFor.OfficialWebsite as any, link: 'https://vrooli.com' },
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
                { name: 'Project Catalyst Entrepreneur Guide' },
            ]
        }
    })
    if (!projectEntrepreneur) {
        console.info('📚 Creating Project Catalyst Guide project');
        projectEntrepreneur = await prisma.project.create({
            data: {
                name: "Project Catalyst Entrepreneur Guide",
                description: "A guide to the best practices and tools for building a successful project on Project Catalyst.",
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
                description: "A metadata standard for Native Token NFTs on Cardano.",
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
    let frameworkBusinessIdea: any = await prisma.routine.findFirst({
        where: {
            AND: [
                { organizationId: vrooli.id },
                { title: 'Starting New Business Frameworks' },
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
                title: 'Starting New Business Frameworks',
                description: 'Hash out your new business idea.',
                instructions: 'Fill out the following forms to collect your thoughts and decide if your business idea is worth pursuing.',
                isAutomatable: true,
                isInternal: false,
                version: '1.0.0',
                createdByOrganization: { connect: { id: vrooli.id } },
                organization: { connect: { id: vrooli.id } },
                inputs: {}, //TODO
                outputs: {}, //TODO
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
                            title: 'Start',
                            type: NodeType.Start,
                        },
                        // Collect thoughts
                        {
                            id: explainId,
                            title: 'Explain',
                            type: NodeType.RoutineList,
                            nodeRoutineList: {
                                create: {
                                    isOrdered: false,
                                    isOptional: false,
                                    routines: {
                                        create: [
                                            {
                                                routine: {
                                                    create: {
                                                        title: 'Overview',
                                                        description: 'Hash out your new business idea.',
                                                        instructions: 'Fill out the form below',
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
                                                    }
                                                }
                                            },
                                            {
                                                routine: {
                                                    create: {
                                                        title: 'Roadmap',
                                                        description: 'Develop a roadmap for your new business',
                                                        instructions: 'Fill out the form below',
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
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
                            title: 'Business',
                            type: NodeType.RoutineList,
                            nodeRoutineList: {
                                create: {
                                    isOrdered: false,
                                    isOptional: true,
                                    routines: {
                                        create: [
                                            {
                                                routine: {
                                                    create: {
                                                        title: 'Company',
                                                        description: "Define your company's structure",
                                                        instructions: 'Fill out the form below',
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
                            title: 'Validate',
                            type: NodeType.RoutineList,
                            nodeRoutineList: {
                                create: {
                                    isOrdered: false,
                                    isOptional: true,
                                    routines: {
                                        create: [
                                            {
                                                routine: {
                                                    create: {
                                                        title: 'Validation',
                                                        description: 'Reflect on your new business idea and decide if it is worth pursuing',
                                                        instructions: 'Fill out the form below',
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
                            title: 'End',
                            type: NodeType.End
                        },
                        // Presentation
                        {
                            id: presentationId,
                            title: 'Presentation',
                            type: NodeType.RoutineList,
                            nodeRoutineList: {
                                create: {
                                    isOrdered: false,
                                    isOptional: true,
                                    routines: {
                                        create: [
                                            {
                                                routine: {
                                                    create: {
                                                        title: 'Marketing and Sales',
                                                        description: 'Define a marketing strategy for your business',
                                                        instructions: 'Fill out the form below',
                                                        //TODO
                                                    }
                                                }
                                            },
                                            {
                                                routine: {
                                                    create: {
                                                        title: 'Pitch Deck',
                                                        description: 'Determine key plans and metrics to measure success',
                                                        instructions: 'Fill out the form below',
                                                        //TODO
                                                    }
                                                }
                                            },
                                        ]
                                    }
                                }
                            }
                        },
                        // Team
                        {
                            id: teamId,
                            title: 'Team',
                            type: NodeType.RoutineList,
                            nodeRoutineList: {
                                create: {
                                    isOrdered: false,
                                    isOptional: true,
                                    routines: {
                                        create: [
                                            {
                                                routine: {
                                                    create: {
                                                        title: 'Culture',
                                                        description: "Define your team's culture",
                                                        instructions: 'Fill out the form below',
                                                        //TODO
                                                    }
                                                }
                                            },
                                            {
                                                routine: {
                                                    create: {
                                                        title: 'Build Team',
                                                        description: 'Build a team to run your business',
                                                        instructions: 'Fill out the form below',
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
                            title: 'Venture',
                            type: NodeType.RoutineList,
                            nodeRoutineList: {
                                create: {
                                    isOrdered: false,
                                    isOptional: true,
                                    routines: {
                                        create: [
                                            {
                                                routine: {
                                                    create: {
                                                        title: 'Sales Playbook',
                                                        description: 'Further defines your sales strategy',
                                                        instructions: 'Fill out the form below',
                                                        //TODO
                                                    }
                                                }
                                            },
                                            {
                                                routine: {
                                                    create: {
                                                        title: 'Product Market Fit',
                                                        description: "Determine your product's market fit",
                                                        instructions: 'Fill out the form below',
                                                        //TODO
                                                    }
                                                }
                                            },
                                            {
                                                routine: {
                                                    create: {
                                                        title: 'Growth State Machine',
                                                        description: 'Define a plan to scale your business',
                                                        instructions: 'Fill out the form below',
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
                            title: 'Finances',
                            type: NodeType.RoutineList,
                            nodeRoutineList: {
                                create: {
                                    isOrdered: false,
                                    isOptional: true,
                                    routines: {
                                        create: [
                                            {
                                                routine: {
                                                    create: {
                                                        title: 'Monetization',
                                                        description: 'Determine how your for-profit business will make money',
                                                        instructions: 'Fill out the form below',
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
                                                    }
                                                }
                                            },
                                            {
                                                routine: {
                                                    create: {
                                                        title: 'Fundraising',
                                                        description: 'Set up fundraising',
                                                        instructions: 'Fill out the form below',
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
                                                    }
                                                }
                                            },
                                            {
                                                routine: {
                                                    create: {
                                                        title: 'Investing',
                                                        description: 'Set up investing',
                                                        instructions: 'Fill out the form below',
                                                        isAutomatable: true,
                                                        isInternal: true,
                                                        version: '1.0.0',
                                                        createdByOrganization: { connect: { id: vrooli.id } },
                                                        organization: { connect: { id: vrooli.id } },
                                                        inputs: {}, //TODO
                                                        outputs: {}, //TODO
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
                            title: 'Scale',
                            type: NodeType.RoutineList,
                            nodeRoutineList: {
                                create: {
                                    isOrdered: false,
                                    isOptional: true,
                                    routines: {
                                        create: [
                                            {
                                                routine: {
                                                    create: {
                                                        title: 'Scale Playbook',
                                                        description: 'Summarize and reformulate key insights you have gained',
                                                        instructions: 'Fill out the form below',
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
                            title: 'End',
                            type: NodeType.End
                        },
                    ]
                }
            }
        })
        // Create node links
        console.log('frameworkBusinessIdea', frameworkBusinessIdea)
    }
    //==============================================================
    /* #endregion Create Routines */
    //==============================================================

    console.info(`✅ Database seeding complete.`);
}

