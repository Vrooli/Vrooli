/**
 * Conversational Behavior
 * 
 * Implements realistic conversational AI behaviors for testing.
 */

import type { LLMRequest, LLMMessage } from "@vrooli/shared";
import type { 
    AIMockConfig, 
    StatefulMockConfig, 
    DynamicMockConfig,
    ConversationMockState, 
} from "../types.js";
import { aiSuccessFixtures } from "../fixtures/successResponses.js";

/**
 * Create a stateful conversational mock
 */
export function createConversationalMock(
    config?: {
        personality?: "helpful" | "technical" | "casual" | "formal";
        memoryLimit?: number;
        contextAware?: boolean;
    },
): StatefulMockConfig<ConversationMockState> {
    const personality = config?.personality || "helpful";
    const memoryLimit = config?.memoryLimit || 10;
    const contextAware = config?.contextAware ?? true;
    
    return {
        initialState: {
            messages: [],
            context: {},
            turnCount: 0,
            totalTokens: 0,
        },
        
        behavior: (request: LLMRequest, state: ConversationMockState): AIMockConfig => {
            state.turnCount++;
            
            // Extract user's latest message
            const userMessage = request.messages
                .filter(m => m.role === "user")
                .pop();
            
            if (!userMessage) {
                return aiSuccessFixtures.clarificationRequest();
            }
            
            // Generate response based on personality and context
            const response = generatePersonalityResponse(
                userMessage.content,
                personality,
                state,
                contextAware,
            );
            
            // Update state with conversation history
            state.messages = [
                ...state.messages.slice(-memoryLimit + 2),
                userMessage,
                { role: "assistant", content: response.content || "" },
            ];
            
            state.totalTokens += response.tokensUsed || 100;
            
            return response;
        },
        
        stateUpdater: (request, response, state) => {
            // Extract and store important context
            const entities = extractEntities(request.messages);
            state.context = { ...state.context, ...entities };
            
            return state;
        },
    };
}

/**
 * Create a dynamic conversational mock with pattern matching
 */
export function createDynamicConversationalMock(): DynamicMockConfig {
    return {
        matcher: (request: LLMRequest): AIMockConfig | null => {
            const lastMessage = request.messages
                .filter(m => m.role === "user")
                .pop()?.content || "";
            
            // Greeting patterns
            if (/^(hi|hello|hey|good\s+(morning|afternoon|evening))/i.test(lastMessage)) {
                return {
                    content: getContextualGreeting(),
                    confidence: 0.95,
                };
            }
            
            // Question patterns
            if (/^(what|when|where|who|why|how|can you|could you|would you)/i.test(lastMessage)) {
                return {
                    content: "I'd be happy to help answer your question. Let me provide you with the information you're looking for.",
                    confidence: 0.88,
                    reasoning: "User asked a direct question, providing helpful response",
                };
            }
            
            // Clarification needed
            if (lastMessage.length < 10 || /\?{2,}/.test(lastMessage)) {
                return aiSuccessFixtures.clarificationRequest();
            }
            
            // Technical questions
            if (/\b(api|function|class|method|algorithm|architecture)\b/i.test(lastMessage)) {
                return aiSuccessFixtures.technicalExplanation();
            }
            
            // Urgent requests
            if (/\b(urgent|asap|immediately|emergency|critical)\b/i.test(lastMessage)) {
                return aiSuccessFixtures.urgentResponse();
            }
            
            return null;
        },
        
        fallback: aiSuccessFixtures.standardResponse(),
    };
}

/**
 * Create a mock that maintains conversation flow
 */
export function createFlowAwareMock(): StatefulMockConfig<{
    topic?: string;
    depth: number;
    previousIntents: string[];
}> {
    return {
        initialState: {
            depth: 0,
            previousIntents: [],
        },
        
        behavior: (request, state) => {
            const intent = detectIntent(request);
            const isTopicChange = state.topic && intent !== state.topic;
            
            if (isTopicChange) {
                // Acknowledge topic change
                return {
                    content: `I see we're shifting from ${state.topic} to ${intent}. Let me help you with that.`,
                    confidence: 0.85,
                    metadata: { topicTransition: true },
                };
            }
            
            state.topic = intent;
            state.depth++;
            state.previousIntents = [...state.previousIntents.slice(-4), intent];
            
            // Deeper conversations get more detailed responses
            const detailLevel = Math.min(state.depth / 3, 1);
            
            return {
                content: generateDepthAwareResponse(intent, detailLevel),
                confidence: 0.8 + (detailLevel * 0.1),
                metadata: {
                    conversationDepth: state.depth,
                    currentTopic: intent,
                },
            };
        },
    };
}

/**
 * Create a mock with emotional intelligence
 */
export function createEmotionallyAwareMock(): DynamicMockConfig {
    return {
        matcher: (request) => {
            const sentiment = analyzeSentiment(request);
            
            if (sentiment.score < -0.5) {
                // User seems frustrated
                return {
                    content: "I understand this might be frustrating. Let me help you resolve this issue step by step.",
                    confidence: 0.9,
                    metadata: { detectedEmotion: "frustration" },
                };
            }
            
            if (sentiment.score > 0.5) {
                // User seems positive
                return {
                    content: "Great! I'm glad to help with this. Your enthusiasm is wonderful!",
                    confidence: 0.92,
                    metadata: { detectedEmotion: "positive" },
                };
            }
            
            if (sentiment.isConfused) {
                // User seems confused
                return {
                    content: "Let me clarify this for you. I'll break it down into simpler terms.",
                    confidence: 0.87,
                    reasoning: "Detected confusion in user message, providing clarification",
                };
            }
            
            return null;
        },
    };
}

/**
 * Helper functions
 */
function generatePersonalityResponse(
    message: string,
    personality: string,
    state: ConversationMockState,
    contextAware: boolean,
): AIMockConfig {
    const responses: Record<string, () => AIMockConfig> = {
        helpful: () => ({
            content: `I'd be happy to help with that! ${contextAware && state.turnCount > 1 ? "Building on our previous discussion, " : ""}Let me assist you.`,
            confidence: 0.9,
        }),
        technical: () => ({
            content: `From a technical perspective, ${message.toLowerCase().includes("implementation") ? "the implementation approach" : "this solution"} involves several key considerations.`,
            confidence: 0.93,
            model: "gpt-4o",
        }),
        casual: () => ({
            content: `Sure thing! ${state.turnCount > 2 ? "We've been chatting for a bit now, and " : ""}I've got you covered.`,
            confidence: 0.85,
        }),
        formal: () => ({
            content: `Thank you for your inquiry. ${contextAware && state.context.userName ? `${state.context.userName}, ` : ""}I shall provide you with the requested information.`,
            confidence: 0.91,
        }),
    };
    
    return responses[personality]?.() || aiSuccessFixtures.standardResponse();
}

function getContextualGreeting(): string {
    const hour = new Date().getHours();
    if (hour < 12) return "Good morning! How can I assist you today?";
    if (hour < 17) return "Good afternoon! What can I help you with?";
    return "Good evening! How may I help you?";
}

function extractEntities(messages: LLMMessage[]): Record<string, any> {
    const entities: Record<string, any> = {};
    
    for (const message of messages) {
        // Extract names
        const nameMatch = message.content.match(/(?:I'm|I am|my name is)\s+(\w+)/i);
        if (nameMatch) {
            entities.userName = nameMatch[1];
        }
        
        // Extract topics
        const topicMatches = message.content.match(/\b(project|task|feature|bug|issue)\s+(\w+)/gi);
        if (topicMatches) {
            entities.topics = topicMatches;
        }
    }
    
    return entities;
}

function detectIntent(request: LLMRequest): string {
    const lastMessage = request.messages
        .filter(m => m.role === "user")
        .pop()?.content || "";
    
    const intents = [
        { pattern: /help|assist|support/i, intent: "help" },
        { pattern: /explain|what is|how does/i, intent: "explanation" },
        { pattern: /create|build|make|generate/i, intent: "creation" },
        { pattern: /fix|debug|error|issue/i, intent: "troubleshooting" },
        { pattern: /optimize|improve|enhance/i, intent: "optimization" },
        { pattern: /test|verify|check/i, intent: "testing" },
    ];
    
    for (const { pattern, intent } of intents) {
        if (pattern.test(lastMessage)) {
            return intent;
        }
    }
    
    return "general";
}

function generateDepthAwareResponse(intent: string, detailLevel: number): string {
    const templates: Record<string, (level: number) => string> = {
        help: (level) => level > 0.7 
            ? "I'll provide comprehensive assistance with detailed steps and examples."
            : "I'm here to help! What specific aspect would you like assistance with?",
        explanation: (level) => level > 0.7
            ? "Let me give you a thorough explanation with technical details and context."
            : "I'll explain this concept for you.",
        creation: (level) => level > 0.7
            ? "I'll help you create this with best practices and optimization in mind."
            : "Let's create what you need.",
        troubleshooting: (level) => level > 0.7
            ? "I'll help diagnose this issue systematically, checking all possible causes."
            : "Let's identify and fix the problem.",
        optimization: (level) => level > 0.7
            ? "I'll analyze performance bottlenecks and suggest targeted optimizations."
            : "Let's improve the performance.",
        testing: (level) => level > 0.7
            ? "I'll help design comprehensive test cases covering edge cases and scenarios."
            : "Let's set up some tests.",
        general: (level) => level > 0.7
            ? "I'll provide a detailed response addressing all aspects of your query."
            : "I'll help you with that.",
    };
    
    return templates[intent]?.(detailLevel) || templates.general(detailLevel);
}

function analyzeSentiment(request: LLMRequest): {
    score: number;
    isConfused: boolean;
} {
    const messages = request.messages.filter(m => m.role === "user");
    if (messages.length === 0) return { score: 0, isConfused: false };
    
    const text = messages.map(m => m.content).join(" ").toLowerCase();
    
    // Simple sentiment analysis
    const positiveWords = ["thanks", "great", "awesome", "perfect", "excellent", "love", "happy"];
    const negativeWords = ["frustrat", "annoying", "stupid", "hate", "terrible", "awful", "useless"];
    const confusionWords = ["confused", "don't understand", "unclear", "lost", "what do you mean", "huh"];
    
    let score = 0;
    for (const word of positiveWords) {
        if (text.includes(word)) score += 0.2;
    }
    for (const word of negativeWords) {
        if (text.includes(word)) score -= 0.3;
    }
    
    const isConfused = confusionWords.some(word => text.includes(word)) || text.includes("??");
    
    return {
        score: Math.max(-1, Math.min(1, score)),
        isConfused,
    };
}
