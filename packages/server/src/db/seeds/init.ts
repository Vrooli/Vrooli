/**
 * Adds initial data to the database. (i.e. data that should be included in production). 
 * This is written so that it can be called multiple times without duplicating data.
 */
import { InputType, uuid, VALYXA_ID } from "@local/shared";
import { Prisma } from "@prisma/client";
import { hashPassword } from "../../auth";
import { logger } from "../../events/logger";
import { PrismaType } from "../../types";

export async function init(prisma: PrismaType) {
    //==============================================================
    /* #region Initialization */
    //==============================================================
    logger.info("🌱 Starting database initial seed...");
    // Check for required .env variables
    if (["ADMIN_WALLET", "ADMIN_PASSWORD", "SITE_EMAIL_USERNAME", "VALYXA_PASSWORD"].some(name => !process.env[name])) {
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
    async function createTag(name: string, description?: string) {
        const tagData: Prisma.tagCreateInput = { tag: name };
        if (description) {
            tagData.translations = {
                create: {
                    language: "en",
                    description,
                },
            };
        }
        return prisma.tag.upsert({
            where: { tag: name },
            update: {},
            create: tagData,
        });
    }
    const tags = await Promise.all([
        // Tags we'll use to seed other objects go first
        createTag("Cardano", "Open-source, decentralized blockchain platform for building and deploying smart contracts and decentralized applications"),
        createTag("CIP", "Cardano Improvement Proposals: Formalized suggestions for changes or improvements to the Cardano protocol"),
        createTag("Entrepreneurship", "Process of starting, organizing, and managing a new business venture"),
        createTag("Vrooli", "Vrooli-related content, including the platform's features, updates, and community activities"),
        createTag("Artificial Intelligence (AI)", "Development of computer systems capable of performing tasks that typically require human intelligence"),
        createTag("Automation", "Use of technology to perform tasks with minimal or no human intervention"),
        createTag("Collaboration", "Working together with others to achieve a common goal or complete a task"),
        // Other tags to cover a wide range of topics
        createTag("Idea Validation", "Process of evaluating the feasibility, market potential, and demand for a new product or business idea"),
        createTag("Learn", "Educational content focused on teaching new skills, concepts, or information"),
        createTag("Research", "Research-related content, including studies, findings, and analysis"),
        createTag("Discover", "Informative content that introduces new products, technologies, or ideas"),
        createTag("Cryptocurrency", "Digital or virtual currency using cryptography for security, enabling decentralized financial transactions"),
        createTag("Cryptography", "Practice of securing communication and data through the use of codes and encryption"),
        createTag("Decentralization", "Distribution of power, authority, or resources away from a central control or location"),
        createTag("Prototyping", "Creating a preliminary model or sample version of a product to test and refine before full-scale production"),
        createTag("No-Code", "Development of applications or software without the need for traditional coding or programming"),
        createTag("Low-Code", "Development approach that minimizes the amount of manual coding required through the use of visual interfaces and pre-built components"),
        createTag("Product Development", "Process of designing, creating, and introducing new products or services to the market"),
        createTag("Idea Generation", "Process of generating, developing, and communicating new ideas or concepts"),
        createTag("Security", "Protection of information, systems, and assets against unauthorized access, use, or harm"),
        createTag("Software Development", "Process of designing, programming, testing, and maintaining computer software"),
        createTag("Open Source", "Software development model where the source code is publicly available and can be modified or redistributed by anyone"),
        createTag("Blockchain", "Decentralized, distributed ledger technology that securely records transactions across multiple computers"),
        createTag("Oracle", "Service that provides external data to smart contracts on a blockchain"),
        createTag("Compliance", "Adherence to laws, regulations, guidelines, and specifications relevant to a business or organization"),
        createTag("Analytics", "Systematic analysis of data to gain insights, make decisions, and solve problems"),
        createTag("Web3", "Next-generation internet model emphasizing decentralization, security, and user control through blockchain and other technologies"),
        createTag("Decentralized Finance (DeFi)", "Financial services built on decentralized platforms, primarily using blockchain technology"),
        createTag("Governance", "Processes, rules, and decision-making structures that determine how an organization, network, or system is managed"),
        createTag("Project Management", "Planning, organizing, and controlling resources to achieve specific goals within a project"),
        createTag("Productivity", "Efficiency in completing tasks and achieving goals"),
        createTag("Documentation", "Creation and organization of written resources and guides"),
        createTag("Deployment", "Process of releasing and distributing software or applications"),
        createTag("Community", "Group of people with shared interests or goals"),
        createTag("Education", "Process of acquiring knowledge, skills, and understanding"),
        createTag("Marketing", "Promoting and selling products or services"),
        createTag("Supply Chain", "Network of organizations involved in production, distribution, and delivery of goods"),
        createTag("Robotics", "Design, construction, and operation of robots and automation"),
        createTag("Healthcare", "Industry focused on providing medical services and improving health outcomes"),
        createTag("Accounting", "Recording, summarizing, and analyzing financial transactions"),
        createTag("Social Media", "Online platforms for sharing content and connecting with others"),
        createTag("Agriculture", "Cultivation of plants and animals for food, fiber, and other products"),
        createTag("Manufacturing", "Production of goods using labor, machines, and tools"),
        createTag("E-Commerce", "Buying and selling of goods and services over the internet"),
        createTag("Finance", "Management of money, investments, and other financial resources"),
        createTag("Insurance", "Industry providing financial protection against risks and losses"),
        createTag("Transportation", "Movement of goods and people from one place to another"),
        createTag("Logistics", "Planning and managing the flow of goods and services"),
        createTag("Real Estate", "Ownership, management, and development of land and property"),
        createTag("Energy", "Production, distribution, and consumption of power sources"),
        createTag("Utilities", "Companies providing essential services such as water, electricity, and gas"),
        createTag("Telecommunications", "Transmission of information through wired or wireless communication systems"),
        createTag("Retail", "Businesses selling products and services directly to consumers"),
        createTag("Hospitality", "Industry providing services related to accommodation, food, and entertainment"),
        createTag("Tourism", "Travel and services related to leisure and business trips"),
        createTag("Automotive", "Design, production, and sale of motor vehicles"),
        createTag("Aerospace", "Development and production of aircraft, spacecraft, and related technologies"),
        createTag("Defense", "Industry focused on national security and military equipment"),
        createTag("Pharmaceuticals", "Development, production, and distribution of drugs and medications"),
        createTag("Biotechnology", "Application of biological systems and living organisms in technology and products"),
        createTag("Construction", "Design and building of infrastructure and structures"),
        createTag("Mining", "Extraction and processing of valuable minerals and resources"),
        createTag("Chemicals", "Production and distribution of chemical substances and products"),
        createTag("Textiles", "Design, production, and distribution of fabrics and fibers"),
        createTag("Food Processing", "Transformation of raw ingredients into consumable food products"),
        createTag("Consumer Goods", "Products intended for everyday use by consumers"),
        createTag("Electronics", "Design and production of electronic devices and systems"),
        createTag("Entertainment", "Creation and distribution of content for amusement and leisure"),
        createTag("Media", "Channels of communication for conveying information and entertainment"),
        createTag("Publishing", "Production and dissemination of literature, music, or other content"),
        createTag("Gaming", "Design, development, and distribution of video games and related products"),
        createTag("Sports", "Physical activities involving competition and recreation"),
        createTag("Fashion", "Trends and styles in clothing, accessories, and personal appearance"),
        createTag("Environment", "Natural surroundings and the impact of human activities on ecosystems"),
        createTag("Waste Management", "Collection, transportation, and disposal of waste materials"),
        createTag("Recycling", "Process of converting waste materials into new products"),
        createTag("Human Resources", "Management of employee relations, recruitment, and development"),
        createTag("Public Relations", "Management of communication between an organization and its publics"),
        createTag("Advertising", "Creation and placement of promotional content to reach target audiences"),
        createTag("Legal Services", "Provision of legal advice and representation"),
        createTag("Consulting", "Professional advice and assistance in various fields and industries"),
        createTag("Information Technology", "Development and management of computer systems and networks"),
        createTag("Cybersecurity", "Protection of computer systems and data from digital threats"),
        createTag("Machine Learning", "Development of algorithms that enable computers to learn from data"),
        createTag("Data Science", "Extraction of knowledge and insights from structured and unstructured data"),
        createTag("Internet of Things", "Interconnected network of devices and sensors that communicate and share data"),
        createTag("Virtual Reality", "Immersive technology that simulates a three-dimensional environment"),
        createTag("Augmented Reality", "Technology that overlays digital content onto the real world"),
        createTag("Digital Twins", "Virtual replicas of physical assets or systems for simulation and analysis"),
        createTag("Big Data", "Large and complex datasets that require advanced processing and analysis"),
        createTag("Cloud Computing", "Delivery of computing services and resources over the internet"),
        createTag("Edge Computing", "Decentralized processing of data closer to the source or edge of a network"),
        createTag("Fintech", "Innovative financial services and technologies"),
        createTag("Regtech", "Technologies that facilitate regulatory compliance and risk management"),
        createTag("Cleantech", "Technologies that promote sustainable use of resources and reduce environmental impact"),
        createTag("Healthtech", "Technologies that improve healthcare delivery and patient outcomes"),
        createTag("Proptech", "Technologies that transform the real estate industry"),
        createTag("Nanotechnology", "Manipulation of matter at the atomic and molecular scale"),
        createTag("Quantum Computing", "Advanced computing systems based on principles of quantum mechanics"),
        createTag("Smart Cities", "Urban areas using technology and data to improve infrastructure and services"),
        createTag("Space Exploration", "Scientific study and discovery beyond Earth's atmosphere"),
        createTag("Drones", "Unmanned aerial vehicles used for various applications"),
        createTag("Autonomous Vehicles", "Self-driving vehicles that operate without human intervention"),
        createTag("E-Government", "Delivery of government services and information through digital channels"),
        createTag("Digital Payments", "Electronic transfer of funds for goods and services"),
        createTag("Sharing Economy", "Economic model based on shared access to goods and services"),
        createTag("Gig Economy", "Labor market characterized by short-term contracts and freelance work"),
        createTag("Remote Work", "Employment that can be performed from any location outside the office"),
        createTag("Coworking Spaces", "Shared workspaces used by remote and independent professionals"),
        createTag("Social Entrepreneurship", "Businesses focused on creating social and environmental impact"),
        createTag("Sustainable Development", "Economic growth that meets present needs without compromising future resources"),
        createTag("Climate Change", "Long-term shifts in global weather patterns due to human activities"),
        createTag("Renewable Energy", "Energy sources that are replenished naturally and have a lower environmental impact"),
        createTag("Circular Economy", "Economic model focused on minimizing waste and maximizing resource efficiency"),
        createTag("Wearable Technology", "Electronics and devices worn on the body for various functions"),
        createTag("Genomics", "Study of an organism's complete set of genetic information"),
        createTag("Precision Medicine", "Tailored medical treatments based on individual genetic, environmental, and lifestyle factors"),
        createTag("Telemedicine", "Delivery of healthcare services through remote communication technologies"),
        createTag("Mental Health", "State of emotional, psychological, and social well-being"),
        createTag("Fitness", "Physical activities and exercises for maintaining health and well-being"),
        createTag("Nutrition", "Science of food and its relationship to health and disease"),
        createTag("Elderly Care", "Services and support for the needs of older adults"),
        createTag("Childcare", "Supervision and care of children in the absence of their parents"),
        createTag("Pet Care", "Services and products for the health and well-being of pets"),
        createTag("Travel", "Exploration and movement between different geographic locations"),
        createTag("Language Learning", "Process of acquiring proficiency in a new language"),
        createTag("Arts", "Creative expression through visual, performing, and literary forms"),
        createTag("Photography", "Art and practice of capturing images with a camera"),
        createTag("Music", "Art of creating and performing sounds with rhythm, melody, and harmony"),
        createTag("Film", "Creation and production of motion pictures as an art form and industry"),
        createTag("Housing", "Provision and development of residential buildings and neighborhoods"),
        createTag("Urban Planning", "Design and regulation of land use and development in urban areas"),
        createTag("Infrastructure", "Basic physical and organizational structures needed for a society to function"),
        createTag("Marine Biology", "Study of organisms and ecosystems in the ocean and other marine environments"),
        createTag("Oceanography", "Scientific study of the physical, chemical, and biological aspects of the ocean"),
        createTag("Forestry", "Management and conservation of forests and related ecosystems"),
        createTag("Water Management", "Control and distribution of water resources for human and environmental needs"),
        createTag("Air Quality", "Measurement and management of pollutants in the atmosphere"),
        createTag("Soil Conservation", "Preservation and improvement of soil quality for agriculture and the environment"),
        createTag("Wildlife Conservation", "Protection and management of animal species and their habitats"),
        createTag("Pollution Control", "Measures taken to reduce and prevent the release of harmful substances"),
        createTag("Natural Disasters", "Sudden and severe events resulting from natural processes of the Earth"),
        createTag("Meteorology", "Study of atmospheric phenomena and weather patterns"),
        createTag("Geology", "Science of the Earth's physical structure and substance, and its history"),
        createTag("Astronomy", "Study of celestial objects, space, and the physical universe"),
        createTag("Astrophysics", "Branch of astronomy focused on the physical and chemical properties of celestial objects"),
        createTag("Particle Physics", "Study of fundamental particles and forces that govern the universe"),
        createTag("Material Science", "Study of the properties, applications, and synthesis of materials"),
        createTag("Bioinformatics", "Application of computational methods to analyze and interpret biological data"),
        createTag("Biomedical Engineering", "Development and application of engineering principles to medicine and biology"),
        createTag("Chemical Engineering", "Design and operation of processes that transform raw materials into valuable products"),
        createTag("Civil Engineering", "Design, construction, and maintenance of the built environment"),
        createTag("Electrical Engineering", "Design, development, and maintenance of electrical systems and equipment"),
        createTag("Mechanical Engineering", "Design, analysis, and manufacturing of mechanical systems and devices"),
        createTag("Industrial Engineering", "Optimization of complex processes, systems, and organizations"),
        createTag("Systems Engineering", "Interdisciplinary field focused on designing and managing complex systems"),
        createTag("Environmental Engineering", "Design and application of engineering principles to protect and improve the environment"),
        createTag("Aeronautical Engineering", "Design, development, and testing of aircraft and aerospace technologies"),
        createTag("Computer Engineering", "Integration of computer hardware and software systems"),
        createTag("Network Engineering", "Design, implementation, and management of communication networks"),
        createTag("Software Engineering", "Development, maintenance, and testing of computer software"),
        createTag("Archaeology", "Study of human history and prehistory through excavation and analysis of artifacts"),
        createTag("Anthropology", "Study of human societies, cultures, and their development"),
        createTag("Sociology", "Study of social relationships, institutions, and human behavior"),
        createTag("Psychology", "Scientific study of the human mind and its functions"),
        createTag("Cognitive Science", "Interdisciplinary study of the mind and its processes"),
        createTag("Philosophy", "Study of fundamental questions about existence, knowledge, values, and reason"),
        createTag("Ethics", "Study of moral principles and values that govern human behavior"),
        createTag("Political Science", "Study of political systems, behavior, and processes"),
        createTag("Economics", "Study of production, distribution, and consumption of goods and services"),
        createTag("International Relations", "Study of political and economic relationships among nations"),
        createTag("Linguistics", "Study of language and its structure, development, and use"),
        createTag("Literature", "Study of written works as a form of artistic expression"),
        createTag("History", "Study of past events and their impact on societies and cultures"),
        createTag("Religious Studies", "Study of religious beliefs, practices, and institutions"),
        createTag("Gender Studies", "Study of gender identities, roles, and their impact on society"),
        createTag("Cultural Studies", "Interdisciplinary study of culture and its relationship with society and politics"),
        createTag("Ethnic Studies", "Study of the social, political, and historical aspects of ethnic groups"),
        createTag("Disability Studies", "Study of disabilities, their impact on individuals, and societal attitudes"),
        createTag("Human Geography", "Study of human populations, their activities, and their impact on the Earth"),
        createTag("Physical Geography", "Study of natural features and processes of the Earth's surface"),
        createTag("Geospatial Technologies", "Tools and methods for analyzing and representing spatial data"),
        createTag("GIS", "Geographic Information Systems: software for storing, analyzing, and visualizing spatial data"),
        createTag("Remote Sensing", "Acquisition of information about an object or phenomenon without direct contact"),
        createTag("Cartography", "Art and science of creating maps and visual representations of geographic data"),
        createTag("Demography", "Study of population characteristics, including size, growth, and distribution"),
        createTag("Criminology", "Study of crime, its causes, and prevention"),
        createTag("Forensic Science", "Application of scientific methods to investigate and solve crimes"),
        createTag("Emergency Management", "Planning, response, and recovery from natural and man-made disasters"),
        createTag("Public Health", "Promotion and protection of health and well-being of communities and populations"),
        createTag("Social Work", "Assisting individuals, families, and communities to enhance their well-being"),
        createTag("Library Science", "Study of organizing, preserving, and providing access to information resources"),
        createTag("Museology", "Study of the operation, management, and development of museums"),
        createTag("Archival Science", "Preservation, organization, and accessibility of historical documents and records"),
        createTag("Conservation", "Preservation and protection of natural and cultural resources"),
        createTag("Restoration", "Process of returning art, architecture, or artifacts to their original condition"),
        createTag("Curatorship", "Management and acquisition of artworks and artifacts for museums and galleries"),
        createTag("Art History", "Study of visual art and its historical development"),
        createTag("Theater", "Creation and performance of live drama, comedy, or musical productions"),
        createTag("Dance", "Art and practice of creating and performing rhythmic movements"),
        createTag("Graphic Design", "Visual communication through the use of typography, images, and illustrations"),
        createTag("Industrial Design", "Design of products for mass production"),
        createTag("Interior Design", "Design of indoor spaces to create functional and aesthetically pleasing environments"),
        createTag("Landscape Architecture", "Design of outdoor spaces, including parks, gardens, and public spaces"),
        createTag("Game Design", "Creation of rules, mechanics, and aesthetics for digital and analog games"),
        createTag("Animation", "Art of creating the illusion of movement through a series of images"),
        createTag("Visual Effects", "Integration of live-action footage and computer-generated imagery"),
        createTag("Illustration", "Art of creating images for books, magazines, and other media"),
        createTag("Typography", "Design and arrangement of text and fonts"),
        createTag("Branding", "Development and management of a company's visual identity and reputation"),
        createTag("Packaging Design", "Creation of functional and visually appealing packaging for products"),
        createTag("Fashion Design", "Design of clothing, accessories, and footwear"),
        createTag("Textile Design", "Design and creation of patterns, textures, and fabrics for various applications"),
        createTag("Culinary Arts", "Art and technique of preparing and cooking food"),
        createTag("Parenting", "Raising and nurturing children through various stages of development"),
        createTag("Relationships", "Interactions and connections between individuals, including romantic and social"),
        createTag("Personal Finance", "Management of individual and family income, expenses, and investments"),
        createTag("Investing", "Allocation of resources to assets with the expectation of generating returns"),
        createTag("Minimalism", "Lifestyle focused on simplifying and decluttering to achieve greater focus and fulfillment"),
        createTag("Personal Development", "Process of self-improvement and self-discovery to reach one's full potential"),
        createTag("Time Management", "Organizing and prioritizing tasks to optimize productivity and efficiency"),
        createTag("Goal Setting", "Establishing and planning for specific, measurable, and achievable objectives"),
        createTag("Stress Management", "Techniques for coping with and reducing stress in daily life"),
        createTag("Emotional Intelligence", "Ability to recognize, understand, and manage one's own emotions and those of others"),
        createTag("Communication", "Exchange of information, thoughts, and ideas through verbal and nonverbal means"),
        createTag("Communication Skills", "Effective verbal, nonverbal, and written techniques for conveying messages"),
        createTag("Leadership", "Ability to guide, inspire, and influence others towards a common goal"),
        createTag("Team Building", "Creating and maintaining strong, cohesive groups through activities and exercises"),
        createTag("Sales", "Process of selling goods and services to customers"),
        createTag("Customer Service", "Support and assistance provided to customers before, during, and after a purchase"),
        createTag("Training", "Development of skills and knowledge through instruction and practice"),
        createTag("Knowledge Management", "Systematic process of creating, sharing, and utilizing organizational information"),
        createTag("Risk Management", "Identification, assessment, and mitigation of potential threats to an organization"),
        createTag("Quality Management", "Monitoring and improving the performance and consistency of products and services"),
        createTag("Business Intelligence", "Use of data analytics and visualization tools to inform strategic decision-making"),
        createTag("Business Analysis", "Process of identifying and solving problems to improve organizational performance"),
        createTag("Market Research", "Gathering and analyzing data on customer preferences, competitors, and market trends"),
        createTag("Digital Marketing", "Promotion of products and services through digital channels and platforms"),
        createTag("Social Media Marketing", "Leveraging social media platforms to engage and attract customers"),
        createTag("Search Engine Optimization", "Improving website visibility and traffic through search engine rankings"),
        createTag("Pay-Per-Click Advertising", "Online advertising model where advertisers pay for each click on their ads"),
        createTag("Email Marketing", "Using targeted email campaigns to communicate with and engage customers"),
        createTag("Affiliate Marketing", "Earning commissions by promoting others' products and services"),
        createTag("Influencer Marketing", "Partnering with influential individuals to promote products and services"),
        createTag("Conversion Rate Optimization", "Improving the percentage of visitors who complete desired actions on a website"),
        createTag("User Acquisition", "Strategies and techniques for attracting and converting new customers or users"),
        createTag("Customer Retention", "Efforts to maintain and strengthen relationships with existing customers"),
        createTag("Customer Success", "Ensuring customers achieve their desired outcomes using a product or service"),
        createTag("User Onboarding", "Process of helping new users become familiar and comfortable with a product or service"),
        createTag("Data Visualization", "Representation of data through graphical elements such as charts and graphs"),
        createTag("Web Design", "Planning, creation, and maintenance of websites and web applications"),
        createTag("Web Development", "Building, coding, and implementing functional elements of websites and web applications"),
        createTag("App Development", "Process of designing, coding, and deploying mobile or desktop applications"),
        createTag("Inventory Management", "Tracking and controlling the quantity, location, and status of products in a supply chain"),
        createTag("Distribution", "Process of delivering goods from manufacturers to customers, including logistics and transportation"),
        createTag("Warehousing", "Storage and handling of goods in a facility to maintain their condition until they are sold or shipped"),
        createTag("Sustainability", "Meeting present needs without compromising the ability of future generations to meet their own needs"),
    ]);
    const tagCardano = tags[0];
    const tagCip = tags[1];
    const tagEntrepreneurship = tags[2];
    const tagVrooli = tags[3];
    const tagAi = tags[4];
    const tagAutomation = tags[5];
    const tagCollaboration = tags[6];
    //==============================================================
    /* #endregion Create Tags */
    //==============================================================

    //==============================================================
    /* #region Create Users */
    //==============================================================
    // Admin
    const adminId = "3f038f3b-f8f9-4f9b-8f9b-c8f4b8f9b8d2";
    const admin = await prisma.user.upsert({
        where: {
            id: adminId,
        },
        update: {
            premium: {
                upsert: {
                    create: {
                        enabledAt: new Date(),
                        expiresAt: new Date("2069-04-20"),
                        isActive: true,
                    },
                    update: {
                        enabledAt: new Date(),
                        expiresAt: new Date("2069-04-20"),
                        isActive: true,
                    },
                },
            },
        },
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
            premium: {
                create: {
                    enabledAt: new Date(),
                    expiresAt: new Date("2069-04-20"),
                    isActive: true,
                },
            },
        },
    });
    // AI assistant
    const valyxaId = VALYXA_ID;
    const valyxa = await prisma.user.upsert({
        where: {
            id: valyxaId,
        },
        update: {
            invitedByUser: { connect: { id: adminId } },
        },
        create: {
            id: valyxaId,
            isBot: true,
            name: "Valyxa",
            password: hashPassword(process.env.VALYXA_PASSWORD ?? ""),
            reputation: 1000000, // TODO temporary until community grows
            status: "Unlocked",
            invitedByUser: { connect: { id: adminId } },
            languages: {
                create: [{ language: EN }],
            },
            translations: {
                create: [{
                    language: EN,
                    bio: "The official AI assistant for Vrooli. Ask me anything!",
                }],
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
            premium: {
                create: {
                    enabledAt: new Date(),
                    expiresAt: new Date("9999-12-31"),
                    isActive: true,
                },
            },
        },
    });

    //==============================================================
    /* #endregion Create Admin */
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
                        { tagTag: tagAi.tag },
                        { tagTag: tagAutomation.tag },
                        { tagTag: tagCollaboration.tag },
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
                                {
                                    usedFor: "Tutorial",
                                    index: 1,
                                    link: "https://docs.vrooli.com",
                                    translations: {
                                        create: [{ language: EN, name: "Vrooli Docs", description: "Interested in how Vrooli works? Want to become a developer? Check out our docs!" }],
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
    const mintTokenId = "3f038f3b-f8f9-4f9b-8f9b-f8f9b8f9b8f9";
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

