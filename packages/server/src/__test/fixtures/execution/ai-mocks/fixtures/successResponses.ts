/**
 * Success Response Fixtures
 * 
 * Pre-defined successful AI response patterns for common scenarios.
 */

import type { AIMockConfig } from "../types.js";

/**
 * Simple greeting response
 */
export const greeting = (): AIMockConfig => ({
    content: "Hello! How can I assist you today?",
    confidence: 0.95,
    model: "gpt-4o-mini",
});

/**
 * Helpful assistant response
 */
export const helpfulResponse = (): AIMockConfig => ({
    content: "I'd be happy to help you with that. Let me break down the solution for you step by step.",
    confidence: 0.9,
    model: "gpt-4o",
});

/**
 * Acknowledgment response
 */
export const acknowledgment = (): AIMockConfig => ({
    content: "I understand. Let me process that information for you.",
    confidence: 0.88,
});

/**
 * Completion response
 */
export const taskComplete = (): AIMockConfig => ({
    content: "I've successfully completed the task. Here's a summary of what was done.",
    confidence: 0.92,
    metadata: {
        taskStatus: "completed",
    },
});

/**
 * Clarification request
 */
export const clarificationRequest = (): AIMockConfig => ({
    content: "I need a bit more information to help you effectively. Could you please clarify what you mean by that?",
    confidence: 0.75,
});

/**
 * Multi-option response
 */
export const multipleOptions = (): AIMockConfig => ({
    content: "There are several ways to approach this:\n1. Option A: Quick but limited\n2. Option B: Comprehensive but time-consuming\n3. Option C: Balanced approach\n\nWhich would you prefer?",
    confidence: 0.85,
});

/**
 * Technical explanation
 */
export const technicalExplanation = (): AIMockConfig => ({
    content: "The system uses a three-tier architecture where Tier 1 handles coordination, Tier 2 manages processes, and Tier 3 executes tasks. This separation ensures scalability and maintainability.",
    confidence: 0.93,
    model: "gpt-4o",
});

/**
 * Context-aware response
 */
export const contextAwareResponse = (overrides?: Partial<AIMockConfig>): AIMockConfig => ({
    content: "Based on our previous discussion, I understand you're working on improving the system performance. Here's my recommendation considering that context.",
    confidence: 0.91,
    reasoning: "The user mentioned performance in their last message, so I'm tailoring my response to address that specific concern.",
    ...overrides,
});

/**
 * Urgent response
 */
export const urgentResponse = (): AIMockConfig => ({
    content: "I understand this is urgent. I'm prioritizing your request and will provide a solution immediately.",
    confidence: 0.94,
    metadata: {
        priority: "high",
        responseTime: "immediate",
    },
});

/**
 * Summary response
 */
export const summaryResponse = (): AIMockConfig => ({
    content: "To summarize our discussion: We covered three main points - system architecture, performance optimization, and testing strategies. The key takeaway is to focus on incremental improvements.",
    confidence: 0.89,
});

/**
 * Creative response
 */
export const creativeResponse = (): AIMockConfig => ({
    content: "Here's a creative approach to your problem: Instead of traditional methods, we could implement an event-driven architecture that allows components to evolve independently.",
    confidence: 0.82,
    model: "gpt-4o",
});

/**
 * Educational response
 */
export const educationalResponse = (): AIMockConfig => ({
    content: "Let me explain this concept in simple terms. Think of it like a restaurant where the waiter (Tier 1) takes orders, the kitchen manager (Tier 2) coordinates cooking, and the chefs (Tier 3) prepare the food.",
    confidence: 0.9,
    reasoning: "Using analogies helps explain complex technical concepts.",
});

/**
 * Validation response
 */
export const validationResponse = (): AIMockConfig => ({
    content: "I've validated your input and everything looks correct. The data meets all the required criteria and is ready for processing.",
    confidence: 0.96,
    metadata: {
        validationStatus: "passed",
        checks: ["format", "range", "consistency"],
    },
});

/**
 * Error recovery response
 */
export const errorRecoveryResponse = (): AIMockConfig => ({
    content: "I noticed an issue with the previous attempt. I've adjusted the approach and successfully completed the task using an alternative method.",
    confidence: 0.87,
    metadata: {
        recoveryMethod: "alternative-approach",
        originalError: "timeout",
    },
});

/**
 * Collaborative response
 */
export const collaborativeResponse = (): AIMockConfig => ({
    content: "Great idea! Let's work together on this. I'll handle the technical implementation while you provide the business requirements. How does that sound?",
    confidence: 0.88,
});

/**
 * Data analysis response
 */
export const dataAnalysisResponse = (): AIMockConfig => ({
    content: "After analyzing the data, I found three key insights: 1) Performance improved by 23%, 2) User engagement increased, and 3) Error rates decreased significantly.",
    confidence: 0.91,
    model: "gpt-4o",
    metadata: {
        analysisType: "statistical",
        dataPoints: 1500,
    },
});

/**
 * Decision support response
 */
export const decisionSupportResponse = (): AIMockConfig => ({
    content: "Based on the criteria you've provided, I recommend Option B. It offers the best balance of performance (85%), cost-effectiveness (90%), and maintainability (88%).",
    confidence: 0.89,
    reasoning: "Weighted analysis of multiple factors with Option B scoring highest overall.",
});

/**
 * Progress update response
 */
export const progressUpdateResponse = (progress: number): AIMockConfig => ({
    content: `Current progress: ${progress}% complete. Estimated time remaining: ${Math.ceil((100 - progress) / 10)} minutes.`,
    confidence: 0.92,
    metadata: {
        progress,
        status: progress === 100 ? "completed" : "in-progress",
    },
});

/**
 * Confirmation response
 */
export const confirmationResponse = (): AIMockConfig => ({
    content: "I've confirmed that your request has been processed successfully. All changes have been saved and are now active.",
    confidence: 0.94,
    finishReason: "stop",
});

/**
 * Standard response
 */
export const standardResponse = (): AIMockConfig => ({
    content: "I've processed your request. The operation completed successfully.",
    confidence: 0.85,
});
