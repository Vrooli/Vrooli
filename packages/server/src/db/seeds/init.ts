/**
 * Adds initial data to the database. (i.e. data that should be included in production). 
 * This is written so that it can be called multiple times without duplicating data.
 */
import { AUTH_PROVIDERS, CodeLanguage, CodeType, FormElement, FormStructureType, InputType, LIST_TO_NUMBERED_PLAINTEXT, LIST_TO_PLAINTEXT_ID, PARSE_RUN_IO_FROM_PLAINTEXT_ID, Project, ResourceUsedFor, Routine, RoutineType, RoutineVersionConfig, Standard, StandardType, TRANSFORM_SEARCH_TERMS_ID, Tag, Team, User, VALYXA_ID, uuid } from "@local/shared";
import { Prisma } from "@prisma/client";
import fs from "fs";
import { PasswordAuthService } from "../../auth/email";
import { prismaInstance } from "../../db/instance";
import { logger } from "../../events/logger";

type TagsByName = { [name: string]: Tag };
type UsersById = { [id: string]: User };
type TeamsByHandle = { [handle: string]: Team };
type ProjectsById = { [id: string]: Project };
type StandardsById = { [id: string]: Standard };
type RoutinesById = { [id: string]: Routine };

const tags: TagsByName = {};
const users: UsersById = {};
const teams: TeamsByHandle = {};
const projects: ProjectsById = {};
const standards: StandardsById = {};
const routines: RoutinesById = {};

const EN = "en";

const tagVrooli = "Vrooli";
const tagAi = "Artificial Intelligence (AI)";
const tagAutomation = "Automation";
const tagCollaboration = "Collaboration";
const tagCardano = "Cardano";
const tagCip = "Cardano Improvement Proposal (CIP)";
const adminId = "3f038f3b-f8f9-4f9b-8f9b-c8f4b8f9b8d2";
const valyxaId = VALYXA_ID;
const vrooliHandle = "vrooli";
const standardCip0025Id = "3a038a3b-f8a9-4fab-8fab-c8a4baaab8d2";
const mintTokenId = "3f038f3b-f8f9-4f9b-8f9b-f8f9b8f9b8f9";
const mintNftId = "4e038f3b-f8f9-4f9b-8f9b-f8f9b8f9b8f9"; // <- DO NOT CHANGE. This is used as a reference routine
const projectKickoffChecklistId = "9daf1edb-b98f-41f0-9d76-aab3539d671a"; // <- DO NOT CHANGE. This is used as a reference routine
const workoutPlanGeneratorId = "3daf1bdb-a98f-41f0-9d76-cab3539d671c"; // <- DO NOT CHANGE. This is used as a reference routine

// Built-in code version functions need to be loaded from a file. We have it this way 
// so that we can run and test them outside of the sandbox
const distCodeFile = `${process.env.PROJECT_DIR}/packages/server/dist/db/seeds/codes.js`;
const srcCodeFile = `${process.env.PROJECT_DIR}/packages/server/src/db/seeds/codes.ts`;

/** Built-in function names to IDs map */
const codeNameToId: { [name: string]: string } = {
    parseRunIOFromPlaintext: PARSE_RUN_IO_FROM_PLAINTEXT_ID,
    transformSearchTerms: TRANSFORM_SEARCH_TERMS_ID,
    listToPlaintext: LIST_TO_PLAINTEXT_ID,
    listToNumberedPlaintext: LIST_TO_NUMBERED_PLAINTEXT,
};

type CodeVersionInfo = {
    name: string;
    description: string;
    code: string;
};

const codes: { [id: string]: CodeVersionInfo } = {};

/**
 * Splits a single JavaScript file content into an array of stringified functions.
 * Each function is expected to start with "export function".
 * 
 * @param jsFileContent The entire JavaScript file content as a single string.
 * @returns An array of string representations of each function.
 */
export function splitFunctions(jsFileContent: string): string[] {
    const lines = jsFileContent.split("\n");
    const functions: string[] = [];
    let currentFunction: string[] = [];
    let isCollectingFunction = false;

    lines.forEach(line => {
        if (line.trim().startsWith("export function")) {
            if (currentFunction.length > 0 && isCollectingFunction) {
                functions.push(currentFunction.join("\n"));
                currentFunction = [];
            }
            isCollectingFunction = true; // Start collecting lines as a function
        }
        if (isCollectingFunction) {
            currentFunction.push(line);
        }
    });

    // Push the last function if exists and it's actually a function
    if (currentFunction.length > 0 && isCollectingFunction) {
        functions.push(currentFunction.join("\n"));
    }

    return functions;
}

/**
 * Parses docstrings from a TypeScript file content and extracts metadata.
 * Each extracted docstring results in an object with the function's name, description, and the function's name.
 * 
 * @param tsFileContent The TypeScript file content as a string.
 * @returns An array of objects containing the name and description from the docstring and the function name.
 */
export function parseDocstrings(tsFileContent: string): { name: string, description: string, functionName: string }[] {
    const docstringRegex = /\/\*\*([\s\S]*?)\*\//g;
    const metadata: { name: string, description: string, functionName: string }[] = [];

    let match;
    while ((match = docstringRegex.exec(tsFileContent)) !== null) {
        const docstringContent = match[1];

        // Extract name and description
        const nameMatch = docstringContent.match(/@name\s+(.*)/);
        const descriptionMatch = docstringContent.match(/@description\s+(.*)/);

        if (nameMatch && descriptionMatch) {
            // Find the function name in the code following this docstring
            const functionRegex = /export function (\w+)/;
            const functionMatch = tsFileContent.substring(match.index + match[0].length)
                .match(functionRegex);

            if (functionMatch) {
                metadata.push({
                    name: nameMatch[1].trim(),
                    description: descriptionMatch[1].trim(),
                    functionName: functionMatch[1],
                });
            }
        }
    }

    return metadata;
}

/**
 * Loads code functions from a file and creates a map of codeVersionID to code version info. 
 * Info includes name, description, and the code itself.
 */
export function loadCodes() {
    try {
        const distFileContent = fs.readFileSync(distCodeFile, "utf-8");
        const functions = splitFunctions(distFileContent);

        const srcFileContent = fs.readFileSync(srcCodeFile, "utf-8");
        const functionMetadata = parseDocstrings(srcFileContent);

        if (functions.length !== functionMetadata.length) {
            logger.error("Error loading built-in functions. Function count mismatch", { trace: "0629" });
            return;
        }

        // Populate the codes object with the function metadata
        functionMetadata.forEach(({ name, description, functionName }, index) => {
            const code = functions[index];
            const id = codeNameToId[functionName];
            if (!id) {
                logger.error("Error loading built-in functions. Function ID not found", { trace: "0630" });
                return;
            }
            codes[id] = { name, description, code };
        });

        console.log("Loaded codes:", JSON.stringify(codes, null, 2));
    } catch (error) {
        logger.error("Error loading built-in functions. They won't be upserted into the database", { trace: "0625" });
    }
}

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
    return prismaInstance.tag.upsert({
        where: { tag: name },
        update: {},
        create: tagData,
    });
}

async function initTags() {
    const tagsList = await Promise.all([
        // Tags we'll use to seed other objects go first
        createTag(tagCardano, "Open-source, decentralized blockchain platform for building and deploying smart contracts and decentralized applications"),
        createTag(tagCip, "Cardano Improvement Proposals: Formalized suggestions for changes or improvements to the Cardano protocol"),
        createTag("Entrepreneurship", "Process of starting, organizing, and managing a new business venture"),
        createTag(tagVrooli, "Vrooli-related content, including the platform's features, updates, and community activities"),
        createTag(tagAi, "Development of computer systems capable of performing tasks that typically require human intelligence"),
        createTag(tagAutomation, "Use of technology to perform tasks with minimal or no human intervention"),
        createTag(tagCollaboration, "Working together with others to achieve a common goal or complete a task"),
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
    tagsList.forEach(tag => tags[tag.tag] = tag as unknown as Tag);
}

async function initUsers() {
    // Admin
    const admin = await prismaInstance.user.upsert({
        where: {
            id: adminId,
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
            id: adminId,
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
                    // eslint-disable-next-line no-magic-numbers
                    credits: BigInt(10_000_000_000),
                },
            },
        },
    });
    users[admin.id] = admin as unknown as User;

    // Default AI assistant
    const valyxa = await prismaInstance.user.upsert({
        where: {
            id: valyxaId,
        },
        update: {
            handle: "valyxa",
            invitedByUser: { connect: { id: adminId } },
        },
        create: {
            id: valyxaId,
            handle: "valyxa",
            isBot: true,
            name: "Valyxa",
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
    users[valyxa.id] = valyxa as unknown as User;
}

async function initTeams() {
    let vrooli = await prismaInstance.team.findFirst({
        where: {
            AND: [
                { translations: { some: { language: EN, name: "Vrooli" } } },
                { members: { some: { userId: adminId } } },
            ],
        },
    });
    if (!vrooli) {
        logger.info("🏗 Creating Vrooli team");
        const teamId = uuid();
        vrooli = await prismaInstance.team.create({
            data: {
                id: teamId,
                handle: vrooliHandle,
                createdBy: { connect: { id: adminId } },
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
                                    team: { connect: { id: teamId } },
                                    user: { connect: { id: adminId } },
                                },
                            ],
                        },
                    },
                },
                tags: {
                    create: [
                        { tagTag: tagVrooli },
                        { tagTag: tagAi },
                        { tagTag: tagAutomation },
                        { tagTag: tagCollaboration },
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
                                    link: "https://x.com/VrooliOfficial",
                                    translations: {
                                        create: [{ language: EN, name: "X", description: "Follow us on X" }],
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
    else {
        vrooli = await prismaInstance.team.update({
            where: { id: vrooli.id },
            data: {
                handle: vrooliHandle,
            },
        });
    }
    teams[vrooli.handle as string] = vrooli as unknown as Team;
}

async function initProjects() {
    let projectEntrepreneur = await prismaInstance.project_version.findFirst({
        where: {
            AND: [
                { root: { ownedByTeamId: teams[vrooliHandle].id } },
                { translations: { some: { language: EN, name: "Project Catalyst Entrepreneur Guide" } } },
            ],
        },
    });
    if (!projectEntrepreneur) {
        logger.info("📚 Creating Project Catalyst Guide project");
        projectEntrepreneur = await prismaInstance.project_version.create({
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
                        createdBy: { connect: { id: adminId } },
                        ownedByTeam: { connect: { id: teams[vrooliHandle].id } },
                    },
                },
            },
        });
    }
    projects[projectEntrepreneur.id] = projectEntrepreneur as unknown as Project;
}

async function initStandards() {
    let standardCip0025 = await prismaInstance.standard_version.findFirst({
        where: {
            id: standardCip0025Id,
        },
    });
    if (!standardCip0025) {
        logger.info("📚 Creating CIP-0025 standard");
        standardCip0025 = await prismaInstance.standard_version.create({
            data: {
                id: standardCip0025Id,
                root: {
                    create: {
                        id: uuid(),
                        permissions: JSON.stringify({}),
                        createdById: adminId,
                        tags: {
                            create: [
                                { tag: { connect: { id: tags[tagCardano].id } } },
                                { tag: { connect: { id: tags[tagCip].id } } },
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
                codeLanguage: CodeLanguage.Json,
                variant: StandardType.DataStructure,
                props: "{\"format\":{\"<721>\":{\"<policy_id>\":{\"<asset_name>\":{\"name\":\"<asset_name>\",\"image\":\"<ipfs_link>\",\"?mediaType\":\"<mime_type>\",\"?description\":\"<description>\",\"?files\":[{\"name\":\"<asset_name>\",\"mediaType\":\"<mime_type>\",\"src\":\"<ipfs_link>\"}],\"[x]\":\"[any]\"}},\"version\":\"1.0\"}},\"defaults\":[]}",
            },
        });
    }
    standards[standardCip0025Id] = standardCip0025 as unknown as Standard;
}

async function initCodes() {
    loadCodes();
    for (const [id, codeInfo] of Object.entries(codes)) {
        let codeVersion = await prismaInstance.code_version.findFirst({ where: { id } });
        if (!codeVersion) {
            logger.info(`📚 Creating ${codeInfo.name} transform function`);
            try {
                codeVersion = await prismaInstance.code_version.create({
                    data: {
                        id,
                        isComplete: true,
                        isLatest: true,
                        isPrivate: false,
                        codeLanguage: "javascript",
                        codeType: CodeType.DataConvert,
                        content: codeInfo.code,
                        root: {
                            create: {
                                id: uuid(),
                                isPrivate: false,
                                permissions: JSON.stringify({}),
                                createdBy: { connect: { id: adminId } },
                                ownedByTeam: { connect: { id: teams[vrooliHandle].id } },
                                tags: {
                                    create: [
                                        { tag: { connect: { id: tags[tagVrooli].id } } },
                                    ],
                                },
                            },
                        },
                        translations: {
                            create: [
                                {
                                    language: EN,
                                    name: codeInfo.name,
                                    description: codeInfo.description,
                                },
                            ],
                        },
                        versionIndex: 0,
                        versionLabel: "1.0.0",
                    },
                });
                logger.info(`${codeInfo.name} function created successfully`);
            } catch (error) {
                logger.error(`Failed to create ${codeInfo.name} function`, { trace: error });
            }
        } else {
            logger.info(`${codeInfo.name} function already exists, skipping creation`);
        }
    }
}

async function initRoutines() {
    let mintToken: any = await prismaInstance.routine.findFirst({
        where: { id: mintTokenId },
    });
    if (!mintToken) {
        logger.info("📚 Creating Native Token Minting routine");
        mintToken = await prismaInstance.routine_version.create({
            data: {
                root: {
                    create: {
                        id: mintTokenId, // Set ID so we can know ahead of time this routine's URL, and link to it as an example/introductory routine
                        permissions: JSON.stringify({}),
                        isInternal: false,
                        createdBy: { connect: { id: adminId } },
                        ownedByTeam: { connect: { id: teams[vrooliHandle].id } },
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
    routines[mintTokenId] = mintToken as unknown as Routine;

    let mintNft: any = await prismaInstance.routine.findFirst({
        where: { id: mintNftId },
    });
    if (!mintNft) {
        logger.info("📚 Creating NFT Minting routine");
        mintNft = await prismaInstance.routine_version.create({
            data: {
                root: {
                    create: {
                        id: mintNftId,
                        permissions: JSON.stringify({}),
                        isInternal: false,
                        createdBy: { connect: { id: adminId } },
                        ownedByTeam: { connect: { id: teams[vrooliHandle].id } },
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
                                    usedFor: ResourceUsedFor.ExternalService,
                                    link: "https://minterr.io/mint-cardano-tokens/",
                                    translations: {
                                        create: [{ language: EN, name: "minterr.io" }],
                                    },
                                },
                                {
                                    usedFor: ResourceUsedFor.ExternalService,
                                    link: "https://cardano-tools.io/mint",
                                    translations: {
                                        create: [{ language: EN, name: "cardano-tools.io" }],
                                    },
                                },
                                {
                                    usedFor: ResourceUsedFor.Developer,
                                    link: "https://developers.cardano.org/docs/native-tokens/minting-nfts",
                                    translations: {
                                        create: [{ language: EN, name: "cardano.org guide" }],
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
    routines[mintNftId] = mintNft as unknown as Routine;

    await prismaInstance.routine.deleteMany({ where: { id: projectKickoffChecklistId } }); //TODO temp
    let projectKickoffChecklist: any = await prismaInstance.routine.findFirst({
        where: { id: projectKickoffChecklistId },
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
        projectKickoffChecklist = await prismaInstance.routine_version.create({
            data: {
                root: {
                    create: {
                        id: projectKickoffChecklistId,
                        permissions: JSON.stringify({}),
                        isInternal: false,
                        createdBy: { connect: { id: adminId } },
                        ownedByTeam: { connect: { id: teams[vrooliHandle].id } },
                    },
                },
                translations: {
                    create: [
                        {
                            language: EN,
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
                                        create: [{ language: EN, name: "Project Charter Template" }],
                                    },
                                },
                                {
                                    usedFor: ResourceUsedFor.Learning,
                                    link: "https://www.pmi.org/about/learn-about-pmi/what-is-project-management",
                                    translations: {
                                        create: [{ language: EN, name: "Project Management Best Practices" }],
                                    },
                                },
                                {
                                    usedFor: ResourceUsedFor.ExternalService,
                                    link: "https://asana.com",
                                    translations: {
                                        create: [{ language: EN, name: "Asana - Project Management Tool" }],
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
    routines[projectKickoffChecklistId] = projectKickoffChecklist as unknown as Routine;

    await prismaInstance.routine.deleteMany({ where: { id: workoutPlanGeneratorId } }); //TODO temp
    let workoutPlanGenerator: any = await prismaInstance.routine.findFirst({
        where: { id: workoutPlanGeneratorId },
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
            callData: {
                __version: "1.0",
                __type: RoutineType.Generate,
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
        workoutPlanGenerator = await prismaInstance.routine_version.create({
            data: {
                root: {
                    create: {
                        id: workoutPlanGeneratorId,
                        permissions: JSON.stringify({}),
                        isInternal: false,
                        createdBy: { connect: { id: adminId } },
                        ownedByTeam: { connect: { id: teams[vrooliHandle].id } },
                    },
                },
                translations: {
                    create: [
                        {
                            language: EN,
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
    routines[workoutPlanGeneratorId] = workoutPlanGenerator as unknown as Routine;
}

export async function init() {
    logger.info("🌱 Starting database initial seed...");
    // Check for required .env variables
    if (["ADMIN_WALLET", "ADMIN_PASSWORD", "SITE_EMAIL_USERNAME", "VALYXA_PASSWORD"].some(name => !process.env[name])) {
        logger.error("🚨 Missing required .env variables. Not seeding database.", { trace: "0006" });
        return;
    }

    // Order matters here. Some objects depend on others.
    await initTags();
    await initUsers();
    await initTeams();
    await initProjects();
    await initStandards();
    await initCodes();
    await initRoutines();

    logger.info("✅ Database seeding complete.");
}

