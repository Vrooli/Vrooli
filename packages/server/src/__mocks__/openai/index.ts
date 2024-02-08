
import "../../../../../node_modules/openai/src/shims/node"; // NOTE: Make sure to save without formatting (use command palette for this), so that this import is above the openai import
import { ChatCompletion, ChatCompletionCreateParams } from "../../../../../node_modules/openai/src/resources";

export { ChatCompletion, ChatCompletionCreateParams };

class OpenAIMock {
    chat = {
        completions: {
            create: (params: ChatCompletionCreateParams): Promise<ChatCompletion> => {
                // Dynamically generate the content based on the input params
                const lastMessageContent = params.messages[params.messages.length - 1].content;
                const mockContent = `Mocked response for: ${lastMessageContent}`;

                // Construct a response that matches the ChatCompletion structure
                const mockResponse: ChatCompletion = {
                    id: "mock_id", // Example mock id
                    model: params.model, // Use the model specified in params
                    object: "chat.completion", // Typically the type of the object
                    created: Math.floor(Date.now() / 1000), // Current timestamp in seconds
                    choices: [{
                        message: {
                            content: mockContent,
                            role: "assistant", // Include the 'role' property as required by the ChatCompletionMessage type
                            // Include other properties for the message if needed
                        },
                        index: 0, // Assuming a single choice for simplicity
                        finish_reason: "length", // Example finish reason
                        logprobs: {} as ChatCompletion.Choice.Logprobs,
                    }],
                    usage: {
                        total_tokens: mockContent.split(" ").length, // Example token count
                        prompt_tokens: params.messages.reduce((acc, message) => acc + (message.content as string).split(" ").length, 0), // Tokens in the prompt
                        completion_tokens: mockContent.split(" ").length, // Tokens in the completion
                    },
                };
                return Promise.resolve(mockResponse);
            },
        },
    };

    // Mock any other necessary OpenAI methods or properties here
}

export default OpenAIMock;
