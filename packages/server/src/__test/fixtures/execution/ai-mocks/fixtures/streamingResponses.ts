/**
 * Streaming Response Fixtures
 * 
 * Pre-defined streaming response patterns for testing real-time AI interactions.
 */

import type { StreamingConfig, StreamChunk } from "../types.js";

/**
 * Simple word-by-word streaming
 */
export const wordByWord = (): StreamingConfig => ({
    chunks: [
        "Hello", " there!", " How", " can", " I", " help", " you", " today?",
    ],
    chunkDelay: 50,
});

/**
 * Character-by-character typing simulation
 */
export const characterTyping = (): StreamingConfig => ({
    chunks: "Hello! I'm here to assist you.".split("").map(char => char),
    chunkDelay: 30,
    simulateTyping: true,
});

/**
 * Variable speed streaming (simulates thinking)
 */
export const variableSpeed = (): StreamingConfig => ({
    chunks: [
        "Let me think about that...",
        " Hmm...",
        " I believe",
        " the answer is",
        " quite straightforward.",
    ],
    chunkDelay: [100, 500, 200, 100, 50],
});

/**
 * Streaming with reasoning
 */
export const withReasoning = (): StreamingConfig => ({
    chunks: ["Based on the analysis,", " I recommend", " using the factory pattern."],
    includeReasoning: true,
    reasoningChunks: [
        "Considering the requirements...",
        "Factory pattern provides flexibility...",
        "This approach scales well...",
    ],
    chunkDelay: 75,
});

/**
 * Long-form streaming response
 */
export const longFormResponse = (): StreamingConfig => ({
    chunks: [
        "This is a comprehensive response that will be streamed in multiple parts.",
        "\n\nFirst, let's understand the problem:",
        "\n- The system needs to handle multiple request types",
        "\n- Each type has different processing requirements",
        "\n- We need to maintain flexibility for future extensions",
        "\n\nSecond, here's the proposed solution:",
        "\n1. Implement a base processor interface",
        "\n2. Create specific processors for each type",
        "\n3. Use a factory to instantiate the correct processor",
        "\n\nFinally, this approach provides several benefits:",
        "\n- Clear separation of concerns",
        "\n- Easy to add new processor types",
        "\n- Testable and maintainable code",
    ],
    chunkDelay: 150,
});

/**
 * Interrupted streaming
 */
export const interruptedStream = (): StreamingConfig => ({
    chunks: [
        "Starting to process your request...",
        " Analyzing the data...",
        " Found an interesting pattern...",
        " Wait, something unexpected—",
    ],
    chunkDelay: 100,
    interruptAt: 3,
    error: {
        type: "NETWORK_ERROR" as any,
        message: "Connection lost",
    },
});

/**
 * Mixed content streaming (text + reasoning)
 */
export const mixedContentStream = (): StreamingConfig => ({
    chunks: [
        { type: "text", content: "I'll analyze this step by step." },
        { type: "reasoning", content: "First, examining the input data..." },
        { type: "text", content: " The data shows three key patterns:" },
        { type: "text", content: "\n1. Increasing trend in user engagement" },
        { type: "reasoning", content: "This suggests the new features are working..." },
        { type: "text", content: "\n2. Lower error rates" },
        { type: "text", content: "\n3. Improved performance metrics" },
    ] as StreamChunk[],
    chunkDelay: 80,
});

/**
 * Code generation streaming
 */
export const codeGeneration = (): StreamingConfig => ({
    chunks: [
        "```typescript",
        "\ninterface UserService {",
        "\n  getUser(id: string): Promise<User>;",
        "\n  updateUser(id: string, data: Partial<User>): Promise<User>;",
        "\n  deleteUser(id: string): Promise<void>;",
        "\n}",
        "\n```",
    ],
    chunkDelay: 60,
});

/**
 * Streaming with tool calls
 */
export const withToolCalls = (): StreamingConfig => ({
    chunks: [
        { type: "text", content: "Let me search for that information..." },
        { type: "tool_call", content: JSON.stringify({ name: "search", arguments: { query: "TypeScript best practices" } }) },
        { type: "text", content: " Based on the search results:" },
        { type: "text", content: "\n- Use strict type checking" },
        { type: "text", content: "\n- Leverage type inference" },
        { type: "text", content: "\n- Avoid 'any' type" },
    ] as StreamChunk[],
    chunkDelay: 100,
});

/**
 * Markdown formatted streaming
 */
export const markdownFormatted = (): StreamingConfig => ({
    chunks: [
        "# Analysis Results\n\n",
        "## Summary\n",
        "The analysis revealed several important findings:\n\n",
        "### Performance Metrics\n",
        "- **Response Time**: 120ms (↓ 15%)\n",
        "- **Throughput**: 1000 req/s (↑ 25%)\n",
        "- **Error Rate**: 0.1% (↓ 50%)\n\n",
        "### Recommendations\n",
        "1. Continue monitoring these metrics\n",
        "2. Consider scaling horizontally\n",
        "3. Implement caching for frequent queries",
    ],
    chunkDelay: 120,
});

/**
 * Conversational streaming
 */
export const conversationalStream = (): StreamingConfig => ({
    chunks: [
        "You know,",
        " that's a really",
        " interesting question!",
        " Let me share",
        " my thoughts on this.",
        " From what I understand,",
        " you're looking for",
        " a solution that balances",
        " performance and maintainability.",
        " Am I right?",
    ],
    chunkDelay: [80, 60, 100, 70, 90, 80, 70, 80, 90, 120],
});

/**
 * Error recovery streaming
 */
export const errorRecoveryStream = (): StreamingConfig => ({
    chunks: [
        "Processing your request...",
        " Encountered an issue...",
        " Attempting alternative approach...",
        " Success! Here's the result:",
        " The operation completed with fallback method.",
    ],
    chunkDelay: [50, 200, 300, 100, 50],
});

/**
 * Multi-language streaming
 */
export const multiLanguageStream = (): StreamingConfig => ({
    chunks: [
        "Hello! ",
        "Bonjour! ",
        "¡Hola! ",
        "你好! ",
        "こんにちは! ",
        "I can help you in multiple languages.",
    ],
    chunkDelay: 100,
});

/**
 * Streaming with progress indicators
 */
export const progressStream = (): StreamingConfig => ({
    chunks: [
        "[10%] Initializing analysis...",
        "\n[25%] Loading data...",
        "\n[50%] Processing information...",
        "\n[75%] Generating insights...",
        "\n[90%] Finalizing results...",
        "\n[100%] Complete! Here are the findings:",
    ],
    chunkDelay: 200,
});

/**
 * Adaptive streaming (starts slow, speeds up)
 */
export const adaptiveStream = (): StreamingConfig => ({
    chunks: [
        "Hmm...",
        " let me...",
        " think about this...",
        " Ah yes!",
        " Now I understand.",
        " Here's what you need to do:",
        " First, implement the base class.",
        " Then add the specific implementations.",
        " Finally, wire everything together.",
    ],
    chunkDelay: [300, 250, 200, 150, 100, 80, 60, 50, 40],
});
