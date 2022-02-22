/**
 * Adds initial data to the database. (i.e. data that should be included in production). 
 * This is written so that it can be called multiple times without duplicating data.
 */

import { AccountStatus, NodeType, ROLES, StandardType } from '@local/shared';
import { UserModel } from '../../models';
import { envVariableExists } from '../../utils';
import { PrismaType } from '../../types';

export async function init(prisma: PrismaType) {
    console.info('🌱 Starting database intial seed...');

    // Check for required .env variables
    if (['ADMIN_WALLET', 'SITE_EMAIL_USERNAME'].some(name => !envVariableExists(name))) {
        console.error('🚨 Missing required .env variables. Not seeding database.');
        return;
    };

    // Initialize models
    const userModel = UserModel(prisma);

    // Create roles
    const actorRole = await prisma.role.upsert({
        where: { title: ROLES.Actor },
        update: {},
        create: { title: ROLES.Actor, description: 'This role allows a user to create routines and save their progress.' },
    })

    // Create tags
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

    // Create admin
    const admin = await prisma.user.upsert({
        where: {
            username: 'admin',
        },
        update: {},
        create: {
            username: 'admin',
            password: userModel.hashPassword('admin'),
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

    // Create Vrooli organization
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
                }
            }
        })
    }

    // Create projects
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

    // Create standards
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

    //TODO temp section
    await prisma.routine.create({
        data: {
            title: 'Test Routine',
            description: 'Hash out your new business idea.',
            instructions: 'Fill out the following forms to collect your thoughts and decide if your business idea is worth pursuing.',
            isAutomatable: true,
            version: '1.0.0',
            createdByOrganization: { connect: { id: vrooli.id } },
            organization: { connect: { id: vrooli.id } },
            inputs: {}, //TODO
            outputs: {}, //TODO
            nodes: {
                create: [
                    {
                        title: 'Explain Idea',
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
                                                    title: 'Overview - Starting New Business Frameworks',
                                                    description: 'Hash out your new business idea.',
                                                    instructions: 'Fill out the form below',
                                                    isAutomatable: true,
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
                    }
                ]
            }
        }
    })
    console.log('successful temp')

    // Create routines
    // let frameworkBusinessIdea = await prisma.routine.findFirst({
    //     where: {
    //         AND: [
    //             { organizationId: vrooli.id },
    //             { title: 'Starting New Business Frameworks' },
    //         ]
    //     }
    // })
    // if (!frameworkBusinessIdea) {
    //     console.info('📚 Creating Starting New Business Frameworks');
    //     frameworkBusinessIdea = await prisma.routine.create({
    //         data: {
    //             title: 'Starting New Business Frameworks',
    //             description: 'Hash out your new business idea.',
    //             instructions: 'Fill out the following forms to collect your thoughts and decide if your business idea is worth pursuing.',
    //             isAutomatable: true,
    //             version: '1.0.0',
    //             createdByOrganization: { connect: { id: vrooli.id } },
    //             organization: { connect: { id: vrooli.id } },
    //             inputs: {}, //TODO
    //             outputs: {}, //TODO
    //             nodes: {
    //                 create: [
    //                     // Start node
    //                     {
    //                         title: 'Start',
    //                         type: NodeType.Start
    //                     },
    //                     // Idea routine list TODO
    //                     {
    //                         title: 'Explain Idea',
    //                         type: NodeType.RoutineList,
    //                         nodeRoutineList: {
    //                             create: {
    //                                 isOrdered: false,
    //                                 isOptional: true,
    //                                 routines: {
    //                                     create: [
    //                                         {
    //                                             title: 'Overview - Starting New Business Frameworks',
    //                                             description: 'Hash out your new business idea.',
    //                                             instructions: 'Fill out the form below',
    //                                             isAutomatable: true,
    //                                             version: '1.0.0',
    //                                             createdByOrganization: { connect: { id: vrooli.id } },
    //                                             organization: { connect: { id: vrooli.id } },
    //                                             inputs: {}, //TODO
    //                                             outputs: {}, //TODO
    //                                         },
    //                                         {
    //                                             title: 'Roadmap - Starting New Business Frameworks',
    //                                             description: 'Develop a roadmap for your new business',
    //                                             instructions: 'Fill out the form below',
    //                                             isAutomatable: true,
    //                                             version: '1.0.0',
    //                                             createdByOrganization: { connect: { id: vrooli.id } },
    //                                             organization: { connect: { id: vrooli.id } },
    //                                             inputs: {}, //TODO
    //                                             outputs: {}, //TODO
    //                                         },
    //                                         // {
    //                                         //     title: 'Decide Profitability - Starting New Business Frameworks',
    //                                         //     description: "Decide what type of business you'd like to create",
    //                                         //     instructions: 'What type of bueinss would you like to ',
    //                                         //     isAutomatable: true,
    //                                         //     version: '1.0.0',
    //                                         //     createdByOrganization: { connect: { id: vrooli.id } },
    //                                         //     organization: { connect: { id: vrooli.id } },
    //                                         //     inputs: {}, //TODO
    //                                         //     outputs: {}, //TODO
    //                                         // },
    //                                         {
    //                                             title: 'Monetization - Starting New Business Frameworks',
    //                                             description: 'Determine how your for-profit business will make money',
    //                                             instructions: 'Fill out the form below',
    //                                             isAutomatable: true,
    //                                             version: '1.0.0',
    //                                             createdByOrganization: { connect: { id: vrooli.id } },
    //                                             organization: { connect: { id: vrooli.id } },
    //                                             inputs: {}, //TODO
    //                                             outputs: {}, //TODO
    //                                         },
    //                                         {
    //                                             title: 'Marketing and Sales - Starting New Business Frameworks',
    //                                             //TODO
    //                                         },
    //                                         {
    //                                             title: 'Company - Starting New Business Frameworks',
    //                                             //TODO
    //                                         },
    //                                         {
    //                                             title: 'Validation - Starting New Business Frameworks',
    //                                             //TODO
    //                                         },
    //                                         {
    //                                             title: 'Pitch Deck - Starting New Business Frameworks',
    //                                         },
    //                                     ]
    //                                 }
    //                             }
    //                         }
    //                     },
    //                     // Venture routine list TODO
    //                     {
    //                         title: 'Explain Business',
    //                         type: NodeType.RoutineList,
    //                         nodeRoutineList: {
    //                             create: {
    //                                 isOrdered: false,
    //                                 isOptional: true,
    //                                 routines: {
    //                                     create: [
    //                                         {
    //                                             title: 'Culture - Starting New Business Frameworks',
    //                                         },
    //                                         {
    //                                             title: 'Sales Playbook - Starting New Business Frameworks',
    //                                         },
    //                                         {
    //                                             title: 'Product Market Fit - Starting New Business Frameworks',
    //                                         },
    //                                         {
    //                                             title: 'Growth State Machine - Starting New Business Frameworks',
    //                                         },
    //                                     ]
    //                                 }
    //                             }
    //                         }
    //                     },
    //                     // Scale routine list TODO
    //                     {
    //                         title: 'Scale Business',
    //                         type: NodeType.RoutineList,
    //                         nodeRoutineList: {
    //                             create: {
    //                                 isOrdered: false,
    //                                 isOptional: true,
    //                                 routines: {
    //                                     create: [
    //                                         {
    //                                             title: 'Scale Playbook - Starting New Business Frameworks',
    //                                         },
    //                                     ]
    //                                 }
    //                             }
    //                         }
    //                     },
    //                     {
    //                         title: 'End',
    //                         type: NodeType.End
    //                     },
    //                 ]
    //             }
    //         }
    //     })
    // }

    console.info(`✅ Database seeding complete.`);
}

