import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import { execSync } from "child_process";
import { generateKeyPairSync } from "crypto";
import { GenericContainer, StartedTestContainer } from "testcontainers";
import { DbProvider } from "../index.js";
import { initializeRedis } from "../redisConn.js";
import { setupTaskQueues } from "../tasks/setup.js";
import { initSingletons } from "../utils/singletons.js";

const SETUP_TIMEOUT_MS = 120_000;

// Enable chai-as-promised for async tests
chai.use(chaiAsPromised);

let redisContainer: StartedTestContainer;
let postgresContainer: StartedTestContainer;

before(async function beforeAllTests() {
    this.timeout(SETUP_TIMEOUT_MS); // Allow extra time for container startup

    // Set up environment variables
    const { publicKey, privateKey } = generateKeyPairSync("rsa", {
        modulusLength: 2048,
        publicKeyEncoding: {
            type: "spki",
            format: "pem",
        },
        privateKeyEncoding: {
            type: "pkcs8",
            format: "pem",
        },
    });
    process.env.JWT_PRIV = privateKey;
    process.env.JWT_PUB = publicKey;
    process.env.ANTHROPIC_API_KEY = "dummy";
    process.env.MISTRAL_API_KEY = "dummy";
    process.env.OPENAI_API_KEY = "dummy";
    process.env.VITE_SERVER_LOCATION = "local";

    // Start the Redis container
    redisContainer = await new GenericContainer("redis")
        .withExposedPorts(6379)
        .start();
    // Set the REDIS_URL environment variable
    const redisHost = redisContainer.getHost();
    const redisPort = redisContainer.getMappedPort(6379);
    process.env.REDIS_URL = `redis://${redisHost}:${redisPort}`;

    // Start the PostgreSQL container
    const POSTGRES_USER = "testuser";
    const POSTGRES_PASSWORD = "testpassword";
    const POSTGRES_DB = "testdb";
    postgresContainer = await new GenericContainer("ankane/pgvector:v0.4.4")
        .withExposedPorts(5432)
        .withEnvironment({
            "POSTGRES_USER": POSTGRES_USER,
            "POSTGRES_PASSWORD": POSTGRES_PASSWORD,
            "POSTGRES_DB": POSTGRES_DB
        })
        .start();
    // Set the POSTGRES_URL environment variable
    const postgresHost = postgresContainer.getHost();
    const postgresPort = postgresContainer.getMappedPort(5432);
    process.env.DB_URL = `postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${postgresHost}:${postgresPort}/${POSTGRES_DB}`;

    // Apply Prisma migrations and generate client
    try {
        console.info('Applying Prisma migrations...');
        execSync('yarn prisma migrate deploy', { stdio: 'inherit' });
        console.info('Generating Prisma client...');
        execSync('yarn prisma generate', { stdio: 'inherit' });
        console.info('Database setup complete.');
    } catch (error) {
        console.error('Failed to set up database:', error);
        throw error; // Fail the test setup if migrations fail
    }

    // Initialize singletons
    await initSingletons();
    // Setup queues
    await setupTaskQueues();
    // Setup databases
    await initializeRedis();
    await DbProvider.init();

    // TODO add sinon mocks for LLM services
});

after(async function afterAllTests() {
    this.timeout(SETUP_TIMEOUT_MS); // Allow extra time for container shutdown

    // Stop the Redis container
    if (redisContainer) {
        await redisContainer.stop();
    }

    // Stop the Postgres container
    if (postgresContainer) {
        await postgresContainer.stop();
    }
});

// Old (jest) AI mocks (to replace with sinon mocks)

// import OpenAI from "openai";

// type ChatCompletion = OpenAI.Chat.Completions.ChatCompletion;
// type ChatCompletionCreateParams = OpenAI.Chat.Completions.ChatCompletionCreateParams;

// class OpenAIMock {
//     chat = {
//         completions: {
//             create: (params: ChatCompletionCreateParams): Promise<ChatCompletion> => {
//                 // Dynamically generate the content based on the input params
//                 let mockContent: string;
//                 if (params.messages.length === 0) {
//                     mockContent = "Mocked response for an empty prompt";
//                 } else {
//                     const lastMessageContent = params.messages[params.messages.length - 1].content;
//                     mockContent = `Mocked response for: ${lastMessageContent}`;
//                 }

//                 // Construct a response that matches the ChatCompletion structure
//                 const mockResponse: ChatCompletion = {
//                     id: "mock_id", // Example mock id
//                     model: params.model, // Use the model specified in params
//                     object: "chat.completion", // Typically the type of the object
//                     created: Math.floor(Date.now() / 1000), // Current timestamp in seconds
//                     choices: [{
//                         message: {
//                             content: mockContent,
//                             role: "assistant", // Include the 'role' property as required by the ChatCompletionMessage type
//                             // Include other properties for the message if needed
//                         },
//                         index: 0, // Assuming a single choice for simplicity
//                         finish_reason: "length", // Example finish reason
//                         logprobs: {} as any,
//                     }],
//                     usage: {
//                         total_tokens: mockContent.split(" ").length, // Example token count
//                         prompt_tokens: params.messages.reduce((acc, message) => acc + (message.content as string).split(" ").length, 0), // Tokens in the prompt
//                         completion_tokens: mockContent.split(" ").length, // Tokens in the completion
//                     },
//                 };
//                 return Promise.resolve(mockResponse);
//             },
//         },
//     };

//     // Mock any other necessary OpenAI methods or properties here
// }

// export default OpenAIMock;


// class AnthropicMock {
//     messages = {
//         create: (params: Anthropic.MessageCreateParams): Promise<Anthropic.Message> => {
//             // Dynamically generate the content based on the input params
//             let mockContent: string;
//             if (params.messages.length === 0) {
//                 mockContent = "Mocked response for an empty prompt";
//             } else {
//                 const lastMessageContent = params.messages[params.messages.length - 1].content;
//                 mockContent = `Mocked response for: ${lastMessageContent}`;
//             }

//             // Construct a response that matches the Message structure
//             const mockResponse: Anthropic.Message = {
//                 id: "mock_id" as const, // Example mock id
//                 model: params.model, // Use the model specified in params
//                 content: [
//                     {
//                         type: "text" as const,
//                         text: mockContent,
//                     },
//                 ],
//                 role: "assistant" as const,
//                 stop_reason: "end_turn" as const,
//                 stop_sequence: null,
//                 type: "message" as const,
//                 usage: {
//                     input_tokens: params.messages.reduce((acc, message) => acc + message.content.length, 0), // Example input token count
//                     output_tokens: mockContent.length, // Example output token count
//                 },
//             };

//             return Promise.resolve(mockResponse);
//         },
//     };
// }

// export default AnthropicMock;


// interface TokenUsage {
//     prompt_tokens: number;
//     completion_tokens: number;
//     total_tokens: number;
// }

// interface ChatCompletionResponseChoice {
//     index: number;
//     message: {
//         role: string;
//         content: string;
//     };
//     finish_reason: string;
// }

// export interface ChatCompletionResponse {
//     id: string;
//     object: "chat.completion";
//     created: number;
//     model: string;
//     choices: ChatCompletionResponseChoice[];
//     usage: TokenUsage;
// }

// class MistralMock {

//     constructor(key: string | undefined) {
//         // Do nothing
//     }

//     chat = (params): Promise<ChatCompletionResponse> => {
//         // Dynamically generate the content based on the input params
//         let mockContent: string;
//         if (params.messages.length === 0) {
//             mockContent = "Mocked response for an empty prompt";
//         } else {
//             const lastMessageContent = params.messages[params.messages.length - 1].content;
//             mockContent = `Mocked response for: ${lastMessageContent}`;
//         }

//         // Construct a response that matches the Message structure
//         const prompt_tokens = params.messages.reduce((acc, message) => acc + message.content.length, 0);
//         const completion_tokens = mockContent.length;
//         const mockResponse: ChatCompletionResponse = {
//             id: "mock_id" as const, // Example mock id
//             object: "chat.completion" as const,
//             created: new Date().getTime() / 1000, // Current timestamp in seconds
//             model: params.model, // Use the model specified in params
//             choices: [{
//                 index: 0,
//                 message: {
//                     role: "assistant" as const,
//                     content: mockContent,
//                 },
//                 finish_reason: "end" as const,
//             }],
//             usage: {
//                 prompt_tokens,
//                 completion_tokens,
//                 total_tokens: prompt_tokens + completion_tokens,
//             },
//         };

//         return Promise.resolve(mockResponse);
//     };
// }

// export default MistralMock;
