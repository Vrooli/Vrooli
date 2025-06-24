# AI Mock Fixtures

This directory contains comprehensive fixtures for mocking AI/LLM interactions in tests. These fixtures enable reliable, deterministic testing of AI-powered features across all three tiers of the execution architecture.

## Overview

The AI mock system provides:
- **Fixture-based Response Definitions**: Pre-defined AI responses for common scenarios
- **Behavior Pattern Support**: Success, error, streaming, tool calls, and more
- **Type-Safe Factories**: Strongly-typed factories for creating mock responses
- **Validation Utilities**: Ensure mocks match real AI interface contracts
- **Integration Testing**: Seamless integration with emergent capability tests

## Directory Structure

```
ai-mocks/
├── README.md                    # This file
├── index.ts                     # Main exports
├── types.ts                     # AI mock type definitions
├── factories/                   # Mock response factories
│   ├── index.ts
│   ├── responseFactory.ts       # Main response factory
│   ├── streamingFactory.ts      # Streaming response factory
│   ├── toolCallFactory.ts       # Tool call response factory
│   └── errorFactory.ts          # Error response factory
├── fixtures/                    # Pre-defined mock scenarios
│   ├── index.ts
│   ├── successResponses.ts      # Successful AI responses
│   ├── errorResponses.ts        # Error scenarios
│   ├── streamingResponses.ts    # Streaming responses
│   ├── toolCallResponses.ts     # Tool calling scenarios
│   └── reasoningResponses.ts    # Chain-of-thought responses
├── validators/                  # Response validation
│   ├── index.ts
│   ├── responseValidator.ts     # Validate mock responses
│   └── contractValidator.ts     # Ensure interface compliance
├── behaviors/                   # Complex behavior patterns
│   ├── index.ts
│   ├── conversationalBehavior.ts
│   ├── reasoningBehavior.ts
│   └── toolUseBehavior.ts
└── integration/                 # Integration with test framework
    ├── index.ts
    ├── mockRegistry.ts          # Global mock registry
    └── testHelpers.ts           # Test utility functions
```

## Quick Start

### Basic Usage

```typescript
import { createAIMockResponse, createStreamingMock } from "./factories/index.js";

// Create a simple successful response
const response = createAIMockResponse({
    content: "Hello! I can help you with that.",
    model: "gpt-4o",
    confidence: 0.95
});

// Create a streaming response
const streamingResponse = createStreamingMock({
    chunks: ["Hello", " there", "!"],
    model: "gpt-4o-mini"
});
```

### Using Pre-defined Fixtures

```typescript
import { aiSuccessFixtures, aiErrorFixtures } from "./fixtures/index.js";

// Use a pre-defined success response
const greetingResponse = aiSuccessFixtures.greeting();

// Use an error response
const rateLimitError = aiErrorFixtures.rateLimit();
```

### Tool Call Mocking

```typescript
import { createToolCallResponse } from "./factories/toolCallFactory.js";

const toolResponse = createToolCallResponse({
    content: "I'll search for that information.",
    toolCalls: [{
        name: "search",
        arguments: { query: "TypeScript best practices" },
        result: { found: 5, items: [...] }
    }]
});
```

### Integration with Tests

```typescript
import { withAIMocks, registerMockBehavior } from "./integration/index.js";

describe("AI-powered feature", () => {
    // Register mock behavior for this test suite
    beforeAll(() => {
        registerMockBehavior("helpful-assistant", {
            pattern: /help|assist/i,
            response: aiSuccessFixtures.helpfulResponse
        });
    });

    it("should handle AI responses", async () => {
        await withAIMocks(async (mockService) => {
            // Your test code here
            const result = await myAIFeature.execute();
            expect(result).toBeDefined();
        });
    });
});
```

## Behavior Patterns

### 1. Success Patterns
- Simple text responses
- Responses with high confidence
- Multi-turn conversations
- Context-aware responses

### 2. Error Patterns
- Rate limiting (429)
- Authentication errors (401)
- Model not found (404)
- Content filter violations
- Timeout errors
- Network failures

### 3. Streaming Patterns
- Character-by-character streaming
- Word-by-word streaming
- Chunk streaming with variable delays
- Stream interruption scenarios

### 4. Tool Call Patterns
- Single tool calls
- Multiple sequential tools
- Parallel tool execution
- Tool call failures
- Tool result processing

### 5. Reasoning Patterns
- Chain-of-thought responses
- Step-by-step reasoning
- Self-correction behaviors
- Confidence adjustments

## Validation

All mock responses are validated to ensure they match the real AI interfaces:

```typescript
import { validateAIResponse } from "./validators/index.js";

const mockResponse = createAIMockResponse({ ... });
const validation = validateAIResponse(mockResponse);

if (!validation.valid) {
    console.error("Invalid mock:", validation.errors);
}
```

## Advanced Features

### Dynamic Behavior

```typescript
import { createDynamicMock } from "./behaviors/index.js";

const dynamicMock = createDynamicMock({
    // Response changes based on input
    matcher: (request) => {
        if (request.messages.some(m => m.content.includes("urgent"))) {
            return aiSuccessFixtures.urgentResponse();
        }
        return aiSuccessFixtures.standardResponse();
    }
});
```

### Stateful Mocks

```typescript
import { createStatefulMock } from "./behaviors/index.js";

const statefulMock = createStatefulMock({
    initialState: { messageCount: 0 },
    behavior: (request, state) => {
        state.messageCount++;
        if (state.messageCount > 3) {
            return aiSuccessFixtures.contextAwareResponse({
                content: "We've been chatting for a while now..."
            });
        }
        return aiSuccessFixtures.greeting();
    }
});
```

### Cost Simulation

```typescript
import { withCostTracking } from "./integration/index.js";

await withCostTracking(async (tracker) => {
    const response = await mockLLM.execute(request);
    const cost = tracker.getTotalCost();
    expect(cost).toBeLessThan(0.10); // $0.10 limit
});
```

## Best Practices

1. **Use Type-Safe Factories**: Always use the provided factories to ensure type safety
2. **Validate Responses**: Run validation on custom mock responses
3. **Match Real Behavior**: Ensure mocks accurately represent real AI behavior
4. **Test Edge Cases**: Include error scenarios and edge cases in your tests
5. **Document Custom Mocks**: Add comments explaining non-obvious mock behaviors

## Integration with Emergent Capabilities

The AI mock system integrates seamlessly with emergent capability testing:

```typescript
import { createEmergentMockBehavior } from "./integration/index.js";

const emergentBehavior = createEmergentMockBehavior({
    capability: "self-improvement",
    evolution: [
        { iteration: 1, response: aiSuccessFixtures.basic() },
        { iteration: 5, response: aiSuccessFixtures.improved() },
        { iteration: 10, response: aiSuccessFixtures.advanced() }
    ]
});
```

## Troubleshooting

### Common Issues

1. **Type Mismatch**: Ensure you're using the correct factory for your response type
2. **Validation Failures**: Check that all required fields are provided
3. **Streaming Issues**: Verify chunk ordering and timing in streaming mocks
4. **Tool Call Errors**: Ensure tool names match registered tools

### Debug Mode

Enable debug logging for detailed mock behavior:

```typescript
import { enableAIMockDebug } from "./integration/index.js";

enableAIMockDebug(true);
```

## Contributing

When adding new mock patterns:
1. Add type definitions to `types.ts`
2. Create appropriate factory methods
3. Add validation rules
4. Include usage examples in tests
5. Update this README

## Related Documentation

- [Execution Fixtures](../README.md)
- [Emergent Capabilities](../emergent-capabilities/README.md)
- [LLM Integration Service](../../../../services/execution/integration/llmIntegrationService.ts)