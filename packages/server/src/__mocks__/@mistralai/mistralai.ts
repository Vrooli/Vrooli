interface TokenUsage {
    prompt_tokens: number;
    completion_tokens: number;
    total_tokens: number;
}

interface ChatCompletionResponseChoice {
    index: number;
    message: {
        role: string;
        content: string;
    };
    finish_reason: string;
}

interface ChatCompletionResponse {
    id: string;
    object: "chat.completion";
    created: number;
    model: string;
    choices: ChatCompletionResponseChoice[];
    usage: TokenUsage;
}

class MistralMock {
    chat = (params): Promise<ChatCompletionResponse> => {
        // Dynamically generate the content based on the input params
        let mockContent: string;
        if (params.messages.length === 0) {
            mockContent = "Mocked response for an empty prompt";
        } else {
            const lastMessageContent = params.messages[params.messages.length - 1].content;
            mockContent = `Mocked response for: ${lastMessageContent}`;
        }

        // Construct a response that matches the Message structure
        const prompt_tokens = params.messages.reduce((acc, message) => acc + message.content.length, 0);
        const completion_tokens = mockContent.length;
        const mockResponse: ChatCompletionResponse = {
            id: "mock_id" as const, // Example mock id
            object: "chat.completion" as const,
            created: new Date().getTime() / 1000, // Current timestamp in seconds
            model: params.model, // Use the model specified in params
            choices: [{
                index: 0,
                message: {
                    role: "assistant" as const,
                    content: mockContent,
                },
                finish_reason: "end" as const,
            }],
            usage: {
                prompt_tokens,
                completion_tokens,
                total_tokens: prompt_tokens + completion_tokens,
            },
        };

        return Promise.resolve(mockResponse);
    };
}

export default MistralMock;
