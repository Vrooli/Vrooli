import Anthropic from "../../../../../node_modules/@anthropic-ai/sdk";

class AnthropicMock {
    messages = {
        create: (params: Anthropic.MessageCreateParams): Promise<Anthropic.Message> => {
            // Dynamically generate the content based on the input params
            let mockContent: string;
            if (params.messages.length === 0) {
                mockContent = "Mocked response for an empty prompt";
            } else {
                const lastMessageContent = params.messages[params.messages.length - 1].content;
                mockContent = `Mocked response for: ${lastMessageContent}`;
            }

            // Construct a response that matches the Message structure
            const mockResponse: Anthropic.Message = {
                id: "mock_id" as const, // Example mock id
                model: params.model, // Use the model specified in params
                content: [
                    {
                        type: "text" as const,
                        text: mockContent,
                    },
                ],
                role: "assistant" as const,
                stop_reason: "end_turn" as const,
                stop_sequence: null,
                type: "message" as const,
                usage: {
                    input_tokens: params.messages.reduce((acc, message) => acc + message.content.length, 0), // Example input token count
                    output_tokens: mockContent.length, // Example output token count
                },
            };

            return Promise.resolve(mockResponse);
        },
    };
}

export default AnthropicMock;
