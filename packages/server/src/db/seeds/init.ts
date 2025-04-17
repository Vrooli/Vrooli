/**
 * Adds initial data to the database. (i.e. data that should be included in production). 
 * This is written so that it can be called multiple times without duplicating data.
 */
import { AUTH_PROVIDERS, CodeLanguage, DEFAULT_LANGUAGE, FormElement, FormStructureType, InputType, ResourceUsedFor, RoutineType, RoutineVersionConfig, SEEDED_IDS, SEEDED_TAGS, StandardType } from "@local/shared";
import pkg from "@prisma/client";
import { PasswordAuthService } from "../../auth/email.js";
import { importData } from "../../builders/importExport.js";
import { logger } from "../../events/logger.js";
import { withRedis } from "../../redisConn.js";
import { RunProcessSelect } from "../../tasks/run/process.js";
import { data as codes } from "./data/codes.js";
import { data as routines } from "./data/routines.js";
import { data as standards } from "./data/standards.js";
import { data as tags } from "./data/tags.js";

const { PrismaClient } = pkg;

const vrooliHandle = "vrooli";

async function initTags(client: InstanceType<typeof PrismaClient>) {
    const tagInfo: { tag: string, description: string }[] = tags.map(([tag, description]) => ({ tag, description }));
    for (const tag of tagInfo) {
        await client.tag.upsert({
            where: { tag: tag.tag },
            update: {},
            create: {
                tag: tag.tag,
                ...(tag.description ? { translations: { create: [{ language: DEFAULT_LANGUAGE, description: tag.description }] } } : {}),
            },
        });
    }
}

async function initUsers(client: InstanceType<typeof PrismaClient>) {
    // Admin
    await client.user.upsert({
        where: {
            id: SEEDED_IDS.User.Admin,
        },
        update: {
            handle: "matt",
            premium: {
                upsert: {
                    create: {
                        enabledAt: new Date(),
                        expiresAt: new Date("2069-04-20"),
                        isActive: true,
                        // eslint-disable-next-line no-magic-numbers
                        credits: BigInt(10_000_000_000),
                    },
                    update: {
                        enabledAt: new Date(),
                        expiresAt: new Date("2069-04-20"),
                        isActive: true,
                        // eslint-disable-next-line no-magic-numbers
                        credits: BigInt(10_000_000_000),
                    },
                },
            },
        },
        create: {
            id: SEEDED_IDS.User.Admin,
            handle: "matt",
            name: "Matt Halloran",
            reputation: 1000000, // TODO temporary until community grows
            status: "Unlocked",
            auths: {
                create: {
                    provider: AUTH_PROVIDERS.Password,
                    hashed_password: PasswordAuthService.hashPassword(process.env.ADMIN_PASSWORD ?? ""),
                },
            },
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
                create: [{ language: DEFAULT_LANGUAGE }],
            },
            focusModes: {
                create: [{
                    name: "Work",
                    description: "This is an auto-generated focus mode. You can edit or delete it.",
                }, {
                    name: "Study",
                    description: "This is an auto-generated focus mode. You can edit or delete it.",
                }],
            },
            awards: {
                create: [{
                    timeCurrentTierCompleted: new Date(),
                    category: "AccountNew",
                    progress: 1,
                }],
            },
            premium: {
                create: {
                    enabledAt: new Date(),
                    expiresAt: new Date("2069-04-20"),
                    isActive: true,
                    // eslint-disable-next-line no-magic-numbers
                    credits: BigInt(10_000_000_000),
                },
            },
        },
    });
    // users[admin.id] = admin as unknown as User;

    // Default AI assistant
    const valyxa = await client.user.upsert({
        where: {
            id: SEEDED_IDS.User.Valyxa,
        },
        update: {
            handle: "valyxa",
            invitedByUser: { connect: { id: SEEDED_IDS.User.Admin } },
        },
        create: {
            id: SEEDED_IDS.User.Valyxa,
            handle: "valyxa",
            isBot: true,
            name: "Valyxa",
            reputation: 1000000, // TODO temporary until community grows
            status: "Unlocked",
            invitedByUser: { connect: { id: SEEDED_IDS.User.Admin } },
            languages: {
                create: [{ language: DEFAULT_LANGUAGE }],
            },
            translations: {
                create: [{
                    language: DEFAULT_LANGUAGE,
                    bio: "The official AI assistant for Vrooli. Ask me anything!",
                }],
            },
            auths: {
                create: {
                    provider: AUTH_PROVIDERS.Password,
                    hashed_password: PasswordAuthService.hashPassword(process.env.VALYXA_PASSWORD ?? ""),
                },
            },
            focusModes: {
                create: [{
                    name: "Work",
                    description: "This is an auto-generated focus mode. You can edit or delete it.",
                }, {
                    name: "Study",
                    description: "This is an auto-generated focus mode. You can edit or delete it.",
                }],
            },
            awards: {
                create: [{
                    timeCurrentTierCompleted: new Date(),
                    category: "AccountNew",
                    progress: 1,
                }],
            },
            premium: {
                create: {
                    enabledAt: new Date(),
                    expiresAt: new Date("9999-12-31"),
                    isActive: true,
                },
            },
        },
    });
    // users[valyxa.id] = valyxa as unknown as User;
}

async function initTeams(client: InstanceType<typeof PrismaClient>) {
    let vrooli = await client.team.findFirst({
        where: {
            AND: [
                { translations: { some: { language: DEFAULT_LANGUAGE, name: "Vrooli" } } },
                { members: { some: { userId: SEEDED_IDS.User.Admin } } },
            ],
        },
    });
    if (!vrooli) {
        logger.info("🏗 Creating Vrooli team");
        vrooli = await client.team.create({
            data: {
                id: SEEDED_IDS.Team.Vrooli,
                handle: vrooliHandle,
                createdBy: { connect: { id: SEEDED_IDS.User.Admin } },
                translations: {
                    create: [
                        {
                            language: DEFAULT_LANGUAGE,
                            name: "Vrooli",
                            bio: "Building an automated, self-improving productivity assistant",
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
                                    team: { connect: { id: SEEDED_IDS.Team.Vrooli } },
                                    user: { connect: { id: SEEDED_IDS.User.Admin } },
                                },
                            ],
                        },
                    },
                },
                tags: {
                    create: [
                        { tagTag: SEEDED_TAGS.Vrooli },
                        { tagTag: SEEDED_TAGS.Ai },
                        { tagTag: SEEDED_TAGS.Automation },
                        { tagTag: SEEDED_TAGS.Collaboration },
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
                                        create: [{ language: DEFAULT_LANGUAGE, name: "Website", description: "Vrooli's official website" }],
                                    },
                                },
                                {
                                    usedFor: "Social",
                                    index: 1,
                                    link: "https://x.com/VrooliOfficial",
                                    translations: {
                                        create: [{ language: DEFAULT_LANGUAGE, name: "X", description: "Follow us on X" }],
                                    },
                                },
                            ],
                        },
                    },
                },
            },
        });
    }
    else {
        vrooli = await client.team.update({
            where: { id: vrooli.id },
            data: {
                handle: vrooliHandle,
            },
        });
    }
    // teams[vrooli.handle as string] = vrooli as unknown as Team;
}

async function initProjects(client: InstanceType<typeof PrismaClient>) {
    let projectEntrepreneur = await client.project_version.findFirst({
        where: {
            AND: [
                { root: { ownedByTeamId: SEEDED_IDS.Team.Vrooli } },
                { translations: { some: { language: DEFAULT_LANGUAGE, name: "Project Catalyst Entrepreneur Guide" } } },
            ],
        },
    });
    if (!projectEntrepreneur) {
        logger.info("📚 Creating Project Catalyst Guide project");
        projectEntrepreneur = await client.project_version.create({
            data: {
                translations: {
                    create: [
                        {
                            language: DEFAULT_LANGUAGE,
                            description: "A guide to the best practices and tools for building a successful project on Project Catalyst.",
                            name: "Project Catalyst Entrepreneur Guide",
                        },
                    ],
                },
                root: {
                    create: {
                        permissions: JSON.stringify({}),
                        createdBy: { connect: { id: SEEDED_IDS.User.Admin } },
                        ownedByTeam: { connect: { id: SEEDED_IDS.Team.Vrooli } },
                    },
                },
            },
        });
    }
    // projects[projectEntrepreneur.id] = projectEntrepreneur as unknown as Project;
}

async function initStandards(client: InstanceType<typeof PrismaClient>) {
    let standardCip0025 = await client.standard_version.findFirst({
        where: {
            id: SEEDED_IDS.Standard.Cip0025,
        },
    });
    if (!standardCip0025) {
        logger.info("📚 Creating CIP-0025 standard");
        standardCip0025 = await client.standard_version.create({
            data: {
                id: SEEDED_IDS.Standard.Cip0025,
                root: {
                    create: {
                        id: SEEDED_IDS.Standard.Cip0025,
                        permissions: JSON.stringify({}),
                        createdById: SEEDED_IDS.User.Admin,
                        tags: {
                            create: [
                                { tag: { connect: { id: SEEDED_IDS.Tag.Cardano } } },
                                { tag: { connect: { id: SEEDED_IDS.Tag.Cip } } },
                            ],
                        },
                    },
                },
                translations: {
                    create: [
                        {
                            language: DEFAULT_LANGUAGE,
                            name: "CIP-0025 - NFT Metadata Standard",
                            description: "A metadata standard for Native Token NFTs on Cardano.",
                        },
                    ],
                },
                versionLabel: "1.0.0",
                versionIndex: 0,
                codeLanguage: CodeLanguage.Json,
                variant: StandardType.DataStructure,
                props: "{\"format\":{\"<721>\":{\"<policy_id>\":{\"<asset_name>\":{\"name\":\"<asset_name>\",\"image\":\"<ipfs_link>\",\"?mediaType\":\"<mime_type>\",\"?description\":\"<description>\",\"?files\":[{\"name\":\"<asset_name>\",\"mediaType\":\"<mime_type>\",\"src\":\"<ipfs_link>\"}],\"[x]\":\"[any]\"}},\"version\":\"1.0\"}},\"defaults\":[]}",
            },
        });
    }
    // standards[standardCip0025Id] = standardCip0025 as unknown as Standard;
}

async function initRoutines(client: InstanceType<typeof PrismaClient>) {
    let mintToken: any = await client.routine.findFirst({
        where: { id: SEEDED_IDS.Routine.MintToken },
    });
    if (!mintToken) {
        logger.info("📚 Creating Native Token Minting routine");
        mintToken = await client.routine_version.create({
            data: {
                root: {
                    create: {
                        id: SEEDED_IDS.Routine.MintToken,
                        permissions: JSON.stringify({}),
                        isInternal: false,
                        createdBy: { connect: { id: SEEDED_IDS.User.Admin } },
                        ownedByTeam: { connect: { id: SEEDED_IDS.Team.Vrooli } },
                    },
                },
                translations: {
                    create: [
                        {
                            language: DEFAULT_LANGUAGE,
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
                                        create: [{ language: DEFAULT_LANGUAGE, name: "minterr.io" }],
                                    },
                                },
                                {
                                    usedFor: "Developer",
                                    link: "https://developers.cardano.org/docs/native-tokens/minting/",
                                    translations: {
                                        create: [{ language: DEFAULT_LANGUAGE, name: "cardano.org guide" }],
                                    },
                                },
                            ],
                        },
                    },
                },
            },
        });
    }
    // routines[mintTokenId] = mintToken as unknown as Routine;

    let mintNft: any = await client.routine.findFirst({
        where: { id: SEEDED_IDS.Routine.MintNft },
    });
    if (!mintNft) {
        logger.info("📚 Creating NFT Minting routine");
        mintNft = await client.routine_version.create({
            data: {
                root: {
                    create: {
                        id: SEEDED_IDS.Routine.MintNft,
                        permissions: JSON.stringify({}),
                        isInternal: false,
                        createdBy: { connect: { id: SEEDED_IDS.User.Admin } },
                        ownedByTeam: { connect: { id: SEEDED_IDS.Team.Vrooli } },
                    },
                },
                translations: {
                    create: [
                        {
                            language: DEFAULT_LANGUAGE,
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
                                    usedFor: ResourceUsedFor.ExternalService,
                                    link: "https://minterr.io/mint-cardano-tokens/",
                                    translations: {
                                        create: [{ language: DEFAULT_LANGUAGE, name: "minterr.io" }],
                                    },
                                },
                                {
                                    usedFor: ResourceUsedFor.ExternalService,
                                    link: "https://cardano-tools.io/mint",
                                    translations: {
                                        create: [{ language: DEFAULT_LANGUAGE, name: "cardano-tools.io" }],
                                    },
                                },
                                {
                                    usedFor: ResourceUsedFor.Developer,
                                    link: "https://developers.cardano.org/docs/native-tokens/minting-nfts",
                                    translations: {
                                        create: [{ language: DEFAULT_LANGUAGE, name: "cardano.org guide" }],
                                    },
                                },
                            ],
                        },
                    },
                },
                routineType: RoutineType.Informational,
            },
        });
    }
    // routines[mintNftId] = mintNft as unknown as Routine;

    await client.routine.deleteMany({ where: { id: SEEDED_IDS.Routine.ProjectKickoffChecklist } }); //TODO temp
    let projectKickoffChecklist: any = await client.routine.findFirst({
        where: { id: SEEDED_IDS.Routine.ProjectKickoffChecklist },
    });
    if (!projectKickoffChecklist) {
        logger.info("📚 Creating Project Kickoff Checklist routine");
        const configFormInputElements: readonly FormElement[] = [
            {
                type: FormStructureType.Header,
                color: "primary",
                id: "intro-header",
                label: "Overview",
                description: "Starting a new project can be both exciting and challenging. This guide will walk you through the essential steps to ensure a successful project kickoff.",
                tag: "h2",
            },
            {
                type: FormStructureType.Header,
                color: "secondary",
                id: "intro-header",
                label: "Starting a new project can be both exciting and challenging. This guide will walk you through the essential steps to ensure a successful project kickoff.",
                tag: "body1",
            },
            {
                type: FormStructureType.Tip,
                icon: "Info",
                id: "intro-tip",
                label: "Remember, thorough planning at the start can save a lot of time and resources down the line.",
            },
            {
                type: FormStructureType.Divider,
                id: "intro-divider",
                label: "",
            },
            // Step 1: Define Project Objectives and Scope
            {
                type: FormStructureType.Header,
                id: "step1-header",
                label: "Step 1: Define Project Objectives and Scope",
                description: `**Purpose**: Clearly articulating your project's objectives and scope is crucial for guiding the team and aligning stakeholder expectations.

**Description**: Outline the primary goals your project aims to achieve and the boundaries within which it will operate. This includes deliverables, timelines, and any constraints or limitations.

**Example**: If you're developing a mobile app, your objective might be "To create a user-friendly mobile application for online shopping that increases customer engagement by 20% within the first year."

**Tips**:
- Be specific and measurable.
- Align objectives with organizational goals.
- Consider both short-term and long-term impacts.`,
                tag: "h3",
            },
            {
                type: InputType.Text,
                id: "projectObjectives",
                fieldName: "projectObjectives",
                isRequired: true,
                label: "Project Objectives",
                props: {
                    isMarkdown: true,
                    placeholder: "Example: Increase customer engagement by 20% within the first year",
                    minRows: 4,
                    maxRows: 6,
                },
                yup: {
                    required: true,
                    checks: [],
                },
            },
            // Step 2: Identify Key Stakeholders
            {
                type: FormStructureType.Header,
                id: "step2-header",
                label: "Step 2: Identify Key Stakeholders",
                description: `**Purpose**: Identifying stakeholders ensures that all parties affected by the project are considered, which aids in gaining support and preventing obstacles.

**Description**: Select individuals, teams, or organizations with a vested interest in the project.

**Tips**:
- Consider stakeholders at all levels.
- Understand their expectations and how they define success.
- Plan how to communicate with each stakeholder group.`,
                tag: "h3",
            },
            {
                type: InputType.Checkbox,
                id: "keyStakeholders",
                fieldName: "keyStakeholders",
                isRequired: false,
                label: "Key Stakeholders",
                props: {
                    options: [
                        { label: "Project Team", value: "project_team" },
                        { label: "Management", value: "management" },
                        { label: "Clients", value: "clients" },
                        { label: "Suppliers", value: "suppliers" },
                        { label: "Regulatory Agencies", value: "regulatory_agencies" },
                    ],
                    row: false,
                    defaultValue: [],
                    allowCustomValues: true,
                    maxCustomValues: 3, // Allows up to 3 custom entries
                },
                yup: {
                    required: false,
                    checks: [],
                },
            },
            // Step 3: Assemble Your Project Team
            {
                type: FormStructureType.Header,
                id: "step3-header",
                label: "Step 3: Assemble Your Project Team",
                description: `**Purpose**: Building the right team is essential for project success, ensuring that all necessary skills and expertise are represented.

**Description**: Determine the roles required for the project and assign team members accordingly. Consider technical skills, experience, and interpersonal dynamics.

**Tips**:
- Balance expertise and workload among team members.
- Clarify roles and responsibilities to avoid overlap.
- Encourage collaboration and open communication.`,
                tag: "h3",
            },
            {
                type: InputType.Checkbox,
                id: "teamRoles",
                fieldName: "teamRoles",
                isRequired: false,
                label: "Team Roles",
                props: {
                    options: [
                        { label: "Project Manager", value: "project_manager" },
                        { label: "Lead Developer", value: "lead_developer" },
                        { label: "Quality Assurance Specialist", value: "qa_specialist" },
                        { label: "UI/UX Designer", value: "ui_ux_designer" },
                        { label: "Business Analyst", value: "business_analyst" },
                    ],
                    row: false,
                    defaultValue: [],
                    allowCustomValues: true,
                    maxCustomValues: 5, // Allows up to 5 custom roles
                },
                yup: {
                    required: false,
                    checks: [],
                },
            },
            // Step 4: Establish Communication Channels
            {
                type: FormStructureType.Header,
                id: "step4-header",
                label: "Step 4: Establish Communication Channels",
                description: `**Purpose**: Effective communication is vital for coordination and timely issue resolution throughout the project.

**Description**: Define how information will be shared within the team and with stakeholders. This includes meetings, reporting methods, and tools used for collaboration.

**Tips**:
- Choose communication methods suitable for your team size and structure.
- Set clear expectations for response times and availability.
- Utilize tools that integrate well with your workflows.`,
                tag: "h3",
            },
            {
                type: InputType.Checkbox,
                id: "communicationChannels",
                fieldName: "communicationChannels",
                isRequired: false,
                label: "Communication Channels",
                props: {
                    options: [
                        { label: "Email", value: "email" },
                        { label: "Slack", value: "slack" },
                        { label: "Microsoft Teams", value: "teams" },
                        { label: "Zoom Meetings", value: "zoom" },
                        { label: "Asana", value: "asana" },
                    ],
                    row: false,
                    defaultValue: [],
                    allowCustomValues: true,
                    maxCustomValues: 3, // Allows up to 3 custom channels
                },
                yup: {
                    required: false,
                    checks: [],
                },
            },
            // Step 5: Establish Timelines and Deliverables
            {
                type: FormStructureType.Header,
                id: "step5-header",
                label: "Step 5: Establish Timelines",
                description: `**Purpose**: Setting a timeline with milestones helps track progress and keeps the project on schedule.

**Description**: Develop a project schedule outlining key deliverables and deadlines. Use project management tools to visualize timelines.

**Tips**:
- Be realistic with time estimates.
- Factor in buffer time for unexpected delays.
- Regularly review and adjust the schedule as needed.`,
                tag: "h3",
            },
            {
                type: InputType.Text,
                id: "projectTimeline",
                fieldName: "projectTimeline",
                isRequired: false,
                label: "Project Timeline",
                props: {
                    isMarkdown: true,
                    placeholder: "Example: Week 1 - Project kickoff meeting, Week 2 - UI/UX design mockups, Week 3 - Frontend development",
                    minRows: 4,
                    maxRows: 6,
                },
                yup: {
                    required: false,
                    checks: [],
                },
            },
            // Step 6: Define Deliverables
            {
                type: FormStructureType.Header,
                id: "step6-header",
                label: "Step 6: Define Deliverables",
                description: `**Purpose**: Clearly defining deliverables ensures that all team members understand what needs to be produced and can work towards common goals.

**Description**: List all tangible and intangible outputs the project is expected to produce, including specifications and acceptance criteria.

**Tips**:
- Make deliverables specific and measurable.
- Align deliverables with project objectives.
- Include quality criteria to define acceptable standards.`,
                tag: "h3",
            },
            {
                type: InputType.Text,
                id: "deliverables",
                fieldName: "deliverables",
                isRequired: true,
                label: "Deliverables",
                props: {
                    isMarkdown: true,
                    placeholder: "Example: Wireframes, User Stories, Code Repository",
                    minRows: 4,
                    maxRows: 6,
                },
                yup: {
                    required: true,
                    checks: [],
                },
            },
            // Step 7: Set Project Budget
            {
                type: FormStructureType.Header,
                id: "step7-header",
                label: "Step 7: Set Project Budget",
                description: `**Purpose**: Estimating the budget ensures that the project has sufficient resources.

**Description**: Define the total budget allocated for the project, including all expenses.

**Tips**:
- Consider all potential costs.
- Include a contingency fund for unexpected expenses.`,
                tag: "h3",
            },
            {
                type: InputType.IntegerInput,
                id: "projectBudget",
                fieldName: "projectBudget",
                isRequired: false,
                label: "Project Budget",
                props: {
                    min: 0,
                    step: 1000,
                },
                yup: {
                    required: false,
                    checks: [],
                },
            },
            // Step 8: Assess Project Risks
            {
                type: FormStructureType.Header,
                id: "step8-header",
                label: "Step 8: Assess Project Risks",
                description: `**Purpose**: Identifying potential risks allows you to plan mitigation strategies.

**Description**: Evaluate the potential risks in terms of likelihood and impact.

**Tips**:
- Consider technical, financial, and operational risks.
- Engage the team in brainstorming potential risks.`,
                tag: "h3",
            },
            {
                type: InputType.Slider,
                id: "riskLevel",
                fieldName: "riskLevel",
                isRequired: false,
                label: "Overall Risk Level",
                props: {
                    min: 0,
                    max: 10,
                    step: 1,
                    valueLabelDisplay: "on",
                },
                yup: {
                    required: false,
                    checks: [],
                },
            },
        ] as const;
        const config = new RoutineVersionConfig({
            __version: "1.0",
            formInput: {
                __version: "1.0",
                schema: {
                    layout: {
                        title: "Project Kickoff Checklist",
                        description: "Follow the steps below to ensure a successful project kickoff.",
                    },
                    containers: [
                        {
                            title: "",
                            description: "",
                            totalItems: configFormInputElements.length,
                        },
                    ],
                    elements: configFormInputElements,
                },
            },
        });
        projectKickoffChecklist = await client.routine_version.create({
            data: {
                root: {
                    create: {
                        id: SEEDED_IDS.Routine.ProjectKickoffChecklist,
                        permissions: JSON.stringify({}),
                        isInternal: false,
                        createdBy: { connect: { id: SEEDED_IDS.User.Admin } },
                        ownedByTeam: { connect: { id: SEEDED_IDS.Team.Vrooli } },
                    },
                },
                translations: {
                    create: [
                        {
                            language: DEFAULT_LANGUAGE,
                            name: "Project Kickoff Checklist",
                            description: "A comprehensive guide to effectively initiate a new project.",
                            instructions: "Fill out the form.",
                        },
                    ],
                },
                complexity: 1,
                simplicity: 1,
                isAutomatable: false,
                versionLabel: "1.0.0",
                versionIndex: 0,
                routineType: RoutineType.Informational,
                resourceList: {
                    create: {
                        resources: {
                            create: [
                                {
                                    usedFor: ResourceUsedFor.Tutorial,
                                    link: "https://www.projectmanagementdocs.com/template/project-charter",
                                    translations: {
                                        create: [{ language: DEFAULT_LANGUAGE, name: "Project Charter Template" }],
                                    },
                                },
                                {
                                    usedFor: ResourceUsedFor.Learning,
                                    link: "https://www.pmi.org/about/learn-about-pmi/what-is-project-management",
                                    translations: {
                                        create: [{ language: DEFAULT_LANGUAGE, name: "Project Management Best Practices" }],
                                    },
                                },
                                {
                                    usedFor: ResourceUsedFor.ExternalService,
                                    link: "https://asana.com",
                                    translations: {
                                        create: [{ language: DEFAULT_LANGUAGE, name: "Asana - Project Management Tool" }],
                                    },
                                },
                            ],
                        },
                    },
                },
                config: config.serialize("json"),
                inputs: {
                    create: [
                        {
                            index: 0,
                            isRequired: true,
                            name: "projectObjectives",
                        },
                        {
                            index: 1,
                            isRequired: true,
                            name: "keyStakeholders",
                        },
                        {
                            index: 2,
                            isRequired: true,
                            name: "teamRoles",
                        },
                        {
                            index: 3,
                            isRequired: true,
                            name: "communicationChannels",
                        },
                        {
                            index: 4,
                            isRequired: true,
                            name: "projectTimeline",
                        },
                        {
                            index: 5,
                            isRequired: true,
                            name: "deliverables",
                        },
                        {
                            index: 6,
                            isRequired: true,
                            name: "projectBudget",
                        },
                        {
                            index: 7,
                            isRequired: true,
                            name: "riskLevel",
                        },
                    ],
                },

            },
        });
    }
    // routines[projectKickoffChecklistId] = projectKickoffChecklist as unknown as Routine;

    await client.routine.deleteMany({ where: { id: SEEDED_IDS.Routine.WorkoutPlanGenerator } }); //TODO temp
    let workoutPlanGenerator: any = await client.routine.findFirst({
        where: { id: SEEDED_IDS.Routine.WorkoutPlanGenerator },
    });
    if (!workoutPlanGenerator) {
        logger.info("📚 Creating Workout Plan Generator routine");

        const configFormInputElements: readonly FormElement[] = [
            {
                type: FormStructureType.Header,
                id: "intro-header",
                label: "Workout Plan Generator",
                description: "Provide some information to receive a personalized workout plan.",
                tag: "h2",
            },
            {
                type: FormStructureType.Header,
                id: "intro-body",
                label: "Please fill out the form below to help us create a workout plan tailored to your needs.",
                tag: "body1",
            },
            {
                type: FormStructureType.Divider,
                id: "intro-divider",
                label: "",
            },
            // Step 1: Choose Your Fitness Goal
            {
                type: FormStructureType.Header,
                id: "fitness-goal-header",
                label: "Step 1: Choose Your Fitness Goal",
                description: `**Purpose**: Selecting a primary fitness goal helps us tailor the workout plan to meet your objectives.
    
**Tips**:
- Choose the goal that best aligns with your current aspirations.
- You can focus on multiple goals, but selecting one primary goal helps with specificity.`,
                tag: "h3",
            },
            {
                type: InputType.Radio,
                id: "fitnessGoal",
                fieldName: "fitnessGoal",
                isRequired: true,
                label: "Fitness Goal",
                props: {
                    options: [
                        { label: "Lose weight", value: "lose_weight" },
                        { label: "Build muscle", value: "build_muscle" },
                        { label: "Improve endurance", value: "improve_endurance" },
                        { label: "Increase flexibility", value: "increase_flexibility" },
                        { label: "General fitness", value: "general_fitness" },
                    ],
                    row: false,
                },
                yup: {
                    required: true,
                    checks: [],
                },
            },
            // Step 2: Indicate Your Current Fitness Level
            {
                type: FormStructureType.Header,
                id: "fitness-level-header",
                label: "Step 2: Indicate Your Current Fitness Level",
                description: `**Purpose**: Understanding your fitness level ensures that the exercises are appropriate and safe for you.
    
**Tips**:
- Be honest about your current fitness level.
- If you're unsure, select the level that feels most accurate.`,
                tag: "h3",
            },
            {
                type: InputType.Radio,
                id: "fitnessLevel",
                fieldName: "fitnessLevel",
                isRequired: true,
                label: "Current Fitness Level",
                props: {
                    options: [
                        { label: "Beginner", value: "beginner" },
                        { label: "Intermediate", value: "intermediate" },
                        { label: "Advanced", value: "advanced" },
                    ],
                    row: false,
                },
                yup: {
                    required: true,
                    checks: [],
                },
            },
            // Step 3: Select Available Equipment
            {
                type: FormStructureType.Header,
                id: "equipment-header",
                label: "Step 3: Select Available Equipment",
                description: `**Purpose**: Knowing what equipment you have helps us design a plan that utilizes your resources.
    
**Tips**:
- Select all that apply.
- If you have other equipment, use the custom option.`,
                tag: "h3",
            },
            {
                type: InputType.Checkbox,
                id: "availableEquipment",
                fieldName: "availableEquipment",
                isRequired: false,
                label: "Available Equipment",
                props: {
                    options: [
                        { label: "None (bodyweight exercises)", value: "none" },
                        { label: "Dumbbells", value: "dumbbells" },
                        { label: "Resistance bands", value: "resistance_bands" },
                        { label: "Barbell and plates", value: "barbell" },
                        { label: "Cardio machines", value: "cardio_machines" },
                        { label: "Yoga mat", value: "yoga_mat" },
                    ],
                    row: false,
                    defaultValue: [],
                    allowCustomValues: true,
                    maxCustomValues: 3,
                },
                yup: {
                    required: false,
                    checks: [],
                },
            },
            // Step 4: Choose Your Preferred Workout Days
            {
                type: FormStructureType.Header,
                id: "workout-days-header",
                label: "Step 4: Choose Your Preferred Workout Days",
                description: `**Purpose**: Scheduling workouts on days that suit you increases consistency and adherence.
    
**Tips**:
- Select all days you're available to work out.
- Be realistic about your schedule.`,
                tag: "h3",
            },
            {
                type: InputType.Checkbox,
                id: "workoutDays",
                fieldName: "workoutDays",
                isRequired: true,
                label: "Preferred Workout Days",
                props: {
                    options: [
                        { label: "Monday", value: "monday" },
                        { label: "Tuesday", value: "tuesday" },
                        { label: "Wednesday", value: "wednesday" },
                        { label: "Thursday", value: "thursday" },
                        { label: "Friday", value: "friday" },
                        { label: "Saturday", value: "saturday" },
                        { label: "Sunday", value: "sunday" },
                    ],
                    row: true,
                    defaultValue: [],
                    allowCustomValues: false,
                },
                yup: {
                    required: true,
                    checks: [],
                },
            },
            // Step 5: List Any Injuries or Physical Limitations
            {
                type: FormStructureType.Header,
                id: "injuries-header",
                label: "Step 5: List Any Injuries or Physical Limitations",
                description: `**Purpose**: Identifying any physical limitations helps us avoid exercises that may cause harm.
    
**Tips**:
- Mention any chronic pain, past injuries, or medical conditions.
- If none, you can leave this blank.`,
                tag: "h3",
            },
            {
                type: InputType.Text,
                id: "injuries",
                fieldName: "injuries",
                isRequired: false,
                label: "Injuries or Physical Limitations",
                props: {
                    isMarkdown: false,
                    placeholder: "Example: Lower back pain, knee injury",
                    minRows: 2,
                    maxRows: 4,
                },
                yup: {
                    required: false,
                    checks: [],
                },
            },
            // Step 6: Specify Desired Workout Duration
            {
                type: FormStructureType.Header,
                id: "time-header",
                label: "Step 6: Specify Desired Workout Duration",
                description: `**Purpose**: Setting workout duration ensures the plan fits into your schedule.
    
**Tips**:
- Be realistic about the time you can commit.
- Consistency is more important than duration.`,
                tag: "h3",
            },
            {
                type: InputType.Slider,
                id: "workoutDuration",
                fieldName: "workoutDuration",
                isRequired: true,
                label: "Time per Workout (minutes)",
                props: {
                    min: 15,
                    max: 120,
                    step: 5,
                    valueLabelDisplay: "on",
                },
                yup: {
                    required: true,
                    checks: [],
                },
            },
            // Step 7: Choose Your Preferred Workout Location
            {
                type: FormStructureType.Header,
                id: "location-header",
                label: "Step 7: Choose Your Preferred Workout Location",
                description: `**Purpose**: Knowing your preferred location helps tailor exercises suitable for that environment.
    
**Tips**:
- Select the location where you're most comfortable exercising.
- If 'Other', please specify.`,
                tag: "h3",
            },
            {
                type: InputType.Radio,
                id: "workoutLocation",
                fieldName: "workoutLocation",
                isRequired: true,
                label: "Preferred Workout Location",
                props: {
                    options: [
                        { label: "Gym", value: "gym" },
                        { label: "Home", value: "home" },
                        { label: "Outdoor", value: "outdoor" },
                        { label: "Other", value: "other" },
                    ],
                    row: false,
                },
                yup: {
                    required: true,
                    checks: [],
                },
            },
            {
                type: InputType.Text,
                id: "otherLocation",
                fieldName: "otherLocation",
                isRequired: false,
                label: "If 'Other', please specify",
                props: {
                    isMarkdown: false,
                    placeholder: "Specify your preferred location",
                    minRows: 1,
                    maxRows: 2,
                },
                yup: {
                    required: false,
                    checks: [],
                },
            },
        ] as const;
        const config = new RoutineVersionConfig({
            __version: "1.0",
            callDataGenerate: {
                __version: "1.0",
                schema: {
                    prompt: `You are a personal trainer creating a customized workout plan.
    
The user has provided the following information:

Fitness Goal: {fitnessGoal}
Fitness Level: {fitnessLevel}
Available Equipment: {availableEquipment}
Preferred Workout Days: {workoutDays}
Injuries or Limitations: {injuries}
Time per Workout: {workoutDuration} minutes
Preferred Workout Location: {workoutLocation}

Generate a detailed weekly workout plan that aligns with the user's goals and constraints.

Make sure to:

- Include exercises appropriate for the user's fitness level.
- Utilize the available equipment.
- Schedule workouts on the preferred days.
- Consider any injuries or limitations.
- Keep each workout within the specified duration.
- Provide tips or modifications if necessary.

Format the plan in a clear and organized manner, using markdown bullet points or tables where appropriate.`,
                },
            },
            formInput: {
                __version: "1.0",
                schema: {
                    layout: {
                        title: "Workout Plan Generator",
                        description: "Provide some information to receive a personalized workout plan.",
                    },
                    containers: [
                        {
                            title: "",
                            description: "",
                            totalItems: configFormInputElements.length,
                        },
                    ],
                    elements: configFormInputElements,
                },
            },
        });
        workoutPlanGenerator = await client.routine_version.create({
            data: {
                root: {
                    create: {
                        id: SEEDED_IDS.Routine.WorkoutPlanGenerator,
                        permissions: JSON.stringify({}),
                        isInternal: false,
                        createdBy: { connect: { id: SEEDED_IDS.User.Admin } },
                        ownedByTeam: { connect: { id: SEEDED_IDS.Team.Vrooli } },
                    },
                },
                translations: {
                    create: [
                        {
                            language: DEFAULT_LANGUAGE,
                            name: "Workout Plan Generator",
                            description: "Generates a personalized workout plan based on your inputs.",
                            instructions: "Fill out the form to receive your workout plan.",
                        },
                    ],
                },
                complexity: 1,
                simplicity: 1,
                isAutomatable: true,
                versionLabel: "1.0.0",
                versionIndex: 0,
                routineType: RoutineType.Generate,
                config: config.serialize("json"),
                inputs: {
                    create: [
                        {
                            index: 0,
                            isRequired: true,
                            name: "fitnessGoal",
                        },
                        {
                            index: 1,
                            isRequired: true,
                            name: "fitnessLevel",
                        },
                        {
                            index: 2,
                            isRequired: false,
                            name: "availableEquipment",
                        },
                        {
                            index: 3,
                            isRequired: true,
                            name: "workoutDays",
                        },
                        {
                            index: 4,
                            isRequired: false,
                            name: "injuries",
                        },
                        {
                            index: 5,
                            isRequired: true,
                            name: "workoutDuration",
                        },
                        {
                            index: 6,
                            isRequired: true,
                            name: "workoutLocation",
                        },
                        {
                            index: 7,
                            isRequired: false,
                            name: "otherLocation",
                        },
                    ],
                },
            },
        });
    }
    // routines[workoutPlanGeneratorId] = workoutPlanGenerator as unknown as Routine;
}

export async function init(client: InstanceType<typeof PrismaClient>) {
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

    const importConfig = {
        allowForeignData: true, // Skip foreign data checks
        assignObjectsTo: { __typename: "Team" as const, id: SEEDED_IDS.Team.Vrooli }, // Assign to Vrooli team
        onConflict: "overwrite" as const, //TODO need update option
        skipPermissions: true, // Skip permission checks
        userData: { id: SEEDED_IDS.User.Admin, languages: [DEFAULT_LANGUAGE] }, // Set user data
    };

    // Order matters here. Some objects depend on others.
    await initTags(client);
    await initUsers(client);
    await initTeams(client);
    await initProjects(client);
    await importData({
        ...importDataBase,
        data: [
            ...standards,
            ...codes,
            ...routines,
        ],
    }, importConfig);

    //TODO
    // For codes, routines, and standards, we seeded them for use in basic routines. 
    // To improve performance, let's cache them in memory
    await withRedis({
        process: async (redisClient) => {
            if (!redisClient) return;
            // Grab the root IDs for every run-related object we want to cache
            const codeRootIds = codes.map((code) => code.shape.id);
            const routineRootIds = routines.map((routine) => routine.shape.id);
            const standardRootIds = standards.map((standard) => standard.shape.id);

            // Determine the fields to request from the database, based on what we use in the run process
            const codeVersionSelect = RunProcessSelect.RoutineVersion.codeVersion.select;
            const codeRootSelect = {
                ...codeVersionSelect.root.select,
                versions: {
                    select: codeVersionSelect,
                },
            };
            const routineVersionSelect = RunProcessSelect.RoutineVersion;
            const routineRootSelect = {
                ...routineVersionSelect.root.select,
                versions: {
                    select: routineVersionSelect,
                },
            };
            // const standardVersionSelect = asdfasdfasdf;
            // const standardRootSelect = asdfasdfasdf;


            // // Read the data, requesting the same fields as the run process uses
            // const codeRoots = await client.code.findMany({
            //     where: { id: { in: codeRootIds } },
            //     select: codeRootSelect,
            // });
            // const routineRoots = await client.routine.findMany({
            //     where: { id: { in: routineRootIds } },
            //     select: routineRootSelect,
            // });
            // const standardRoots = await client.standard.findMany({
            //     where: { id: { in: standardRootIds } },
            //     select: standardRootSelect,
            // });

            // Flip the data around so it's by version, not root
            //asdfasdfasfd

            // Cache the version data in Redis
            // for (const code of codes) {
            //     redisClient.set(`seeded:code:${code.id}`, JSON.stringify(code));
            // }
            // for (const routine of routines) {
            //     redisClient.set(`seeded:routine:${routine.id}`, JSON.stringify(routine));
            // }
            // for (const standard of standards) {
            //     redisClient.set(`seeded:standard:${standard.id}`, JSON.stringify(standard));
            // }
        },
        trace: "0705",
    });

    logger.info("✅ Database seeding complete.");
}

