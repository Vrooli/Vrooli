/**
 * Reasoning Behavior
 * 
 * Implements chain-of-thought and reasoning behaviors for AI mocks.
 */

import type { LLMRequest } from "@vrooli/shared";
import type { AIMockConfig, StatefulMockConfig, DynamicMockConfig } from "../types.js";
import { aiReasoningFixtures } from "../fixtures/reasoningResponses.js";

/**
 * Create a mock with step-by-step reasoning
 */
export function createStepByStepReasoning(): DynamicMockConfig {
    return {
        matcher: (request: LLMRequest): AIMockConfig | null => {
            const lastMessage = request.messages
                .filter(m => m.role === "user")
                .pop()?.content || "";
            
            // Check if the request requires reasoning
            if (requiresReasoning(lastMessage)) {
                const steps = generateReasoningSteps(lastMessage);
                const conclusion = deriveConclusion(steps);
                
                return {
                    content: conclusion,
                    reasoning: formatReasoningSteps(steps),
                    confidence: calculateConfidence(steps),
                    model: "o1-mini",
                };
            }
            
            return null;
        },
    };
}

/**
 * Create a mock with self-improving reasoning
 */
export function createSelfImprovingReasoning(): StatefulMockConfig<{
    reasoningPatterns: Map<string, { success: number; attempts: number }>;
    feedback: string[];
}> {
    return {
        initialState: {
            reasoningPatterns: new Map(),
            feedback: [],
        },
        
        behavior: (request, state) => {
            const problemType = classifyProblem(request);
            const pattern = state.reasoningPatterns.get(problemType) || { success: 0, attempts: 0 };
            
            // Select reasoning strategy based on past performance
            const strategy = selectOptimalStrategy(problemType, pattern);
            const reasoning = applyReasoningStrategy(strategy, request);
            
            // Update pattern statistics
            pattern.attempts++;
            state.reasoningPatterns.set(problemType, pattern);
            
            return {
                content: reasoning.conclusion,
                reasoning: reasoning.process,
                confidence: reasoning.confidence,
                metadata: {
                    strategy: strategy.name,
                    problemType,
                    improvementMetrics: {
                        successRate: pattern.success / Math.max(pattern.attempts, 1),
                        totalAttempts: pattern.attempts,
                    },
                },
            };
        },
    };
}

/**
 * Create a mock with hypothesis testing reasoning
 */
export function createHypothesisTestingMock(): DynamicMockConfig {
    return {
        matcher: (request) => {
            const problem = extractProblemStatement(request);
            if (!problem) return null;
            
            const hypotheses = generateHypotheses(problem);
            const testResults = hypotheses.map(h => testHypothesis(h, problem));
            const bestHypothesis = selectBestHypothesis(testResults);
            
            return {
                content: `Based on hypothesis testing, ${bestHypothesis.conclusion}`,
                reasoning: formatHypothesisTesting(hypotheses, testResults, bestHypothesis),
                confidence: bestHypothesis.confidence,
                metadata: {
                    hypothesesTested: hypotheses.length,
                    selectedHypothesis: bestHypothesis.id,
                },
            };
        },
    };
}

/**
 * Create a mock with causal reasoning
 */
export function createCausalReasoningMock(): DynamicMockConfig {
    return {
        matcher: (request) => {
            const scenario = extractScenario(request);
            if (!scenario) return null;
            
            const causalChain = buildCausalChain(scenario);
            const rootCause = identifyRootCause(causalChain);
            const effects = predictEffects(rootCause, causalChain);
            
            return {
                content: formatCausalConclusion(rootCause, effects),
                reasoning: formatCausalChain(causalChain),
                confidence: calculateCausalConfidence(causalChain),
                metadata: {
                    causalDepth: causalChain.length,
                    rootCause: rootCause.description,
                },
            };
        },
    };
}

/**
 * Create a mock with analogical reasoning
 */
export function createAnalogicalReasoningMock(): StatefulMockConfig<{
    knownPatterns: Array<{ domain: string; pattern: string; solution: string }>;
}> {
    return {
        initialState: {
            knownPatterns: getDefaultPatterns(),
        },
        
        behavior: (request, state) => {
            const currentProblem = extractProblemStatement(request);
            if (!currentProblem) {
                return aiReasoningFixtures.simpleReasoning();
            }
            
            // Find similar patterns
            const similarities = state.knownPatterns.map(pattern => ({
                pattern,
                similarity: calculateSimilarity(currentProblem, pattern.pattern),
            })).sort((a, b) => b.similarity - a.similarity);
            
            const bestMatch = similarities[0];
            
            if (bestMatch.similarity > 0.7) {
                return {
                    content: `This problem is similar to ${bestMatch.pattern.domain}. ${adaptSolution(bestMatch.pattern.solution, currentProblem)}`,
                    reasoning: `Drawing analogy from ${bestMatch.pattern.domain}:\n` +
                              `- Similar pattern: ${bestMatch.pattern.pattern}\n` +
                              "- Adapted solution: Applying similar principles to current context",
                    confidence: bestMatch.similarity,
                    metadata: {
                        analogySource: bestMatch.pattern.domain,
                        similarityScore: bestMatch.similarity,
                    },
                };
            }
            
            return aiReasoningFixtures.stepByStepReasoning();
        },
    };
}

/**
 * Create a mock with metacognitive reasoning
 */
export function createMetacognitiveMock(): StatefulMockConfig<{
    thinkingDepth: number;
    uncertainties: string[];
}> {
    return {
        initialState: {
            thinkingDepth: 0,
            uncertainties: [],
        },
        
        behavior: (request, state) => {
            state.thinkingDepth++;
            
            // Analyze own thinking process
            const thinkingAnalysis = analyzeOwnThinking(request, state.thinkingDepth);
            
            if (thinkingAnalysis.needsMoreInfo) {
                state.uncertainties.push(thinkingAnalysis.uncertainty);
                return aiReasoningFixtures.metaReasoning();
            }
            
            return {
                content: thinkingAnalysis.response,
                reasoning: "Metacognitive analysis:\n" +
                          `- Thinking depth: ${state.thinkingDepth}\n` +
                          `- Confidence in approach: ${thinkingAnalysis.confidence}\n` +
                          `- Key insights: ${thinkingAnalysis.insights.join(", ")}`,
                confidence: thinkingAnalysis.confidence,
                metadata: {
                    metacognitionLevel: state.thinkingDepth,
                    identifiedGaps: state.uncertainties,
                },
            };
        },
    };
}

/**
 * Helper functions
 */
function requiresReasoning(message: string): boolean {
    const reasoningTriggers = [
        /why|how|explain|reasoning|think through|analyze/i,
        /pros and cons|compare|evaluate|decide/i,
        /what would happen if|predict|forecast/i,
        /solve|calculate|figure out|work out/i,
    ];
    
    return reasoningTriggers.some(pattern => pattern.test(message));
}

function generateReasoningSteps(message: string): string[] {
    const steps: string[] = [];
    
    // Identify problem type
    if (message.includes("calculate") || message.includes("math")) {
        steps.push("Identify the mathematical operation required");
        steps.push("Extract the relevant numbers and variables");
        steps.push("Apply the appropriate formula or method");
        steps.push("Verify the calculation");
    } else if (message.includes("compare")) {
        steps.push("Identify the items to compare");
        steps.push("Determine comparison criteria");
        steps.push("Evaluate each item against criteria");
        steps.push("Synthesize findings");
    } else {
        steps.push("Understand the core question");
        steps.push("Identify relevant factors");
        steps.push("Analyze relationships");
        steps.push("Draw conclusions");
    }
    
    return steps;
}

function formatReasoningSteps(steps: string[]): string {
    return steps.map((step, i) => `${i + 1}. ${step}`).join("\n");
}

function deriveConclusion(steps: string[]): string {
    return `After following ${steps.length} analytical steps, I've reached a comprehensive conclusion.`;
}

function calculateConfidence(steps: string[]): number {
    // More steps generally means more thorough analysis
    const baseConfidence = 0.7;
    const stepBonus = Math.min(steps.length * 0.05, 0.25);
    return baseConfidence + stepBonus;
}

function classifyProblem(request: LLMRequest): string {
    const lastMessage = request.messages
        .filter(m => m.role === "user")
        .pop()?.content || "";
    
    const classifications = [
        { pattern: /calculat|math|number|sum|average/i, type: "mathematical" },
        { pattern: /compar|versus|better|choose/i, type: "comparative" },
        { pattern: /why|cause|reason|because/i, type: "causal" },
        { pattern: /how|step|process|procedure/i, type: "procedural" },
        { pattern: /what if|would|could|should/i, type: "hypothetical" },
        { pattern: /predict|forecast|future|will/i, type: "predictive" },
    ];
    
    for (const { pattern, type } of classifications) {
        if (pattern.test(lastMessage)) return type;
    }
    
    return "general";
}

function selectOptimalStrategy(
    problemType: string,
    performance: { success: number; attempts: number },
): { name: string; approach: string } {
    const strategies = {
        mathematical: [
            { name: "algebraic", approach: "systematic equation solving" },
            { name: "numerical", approach: "iterative approximation" },
        ],
        comparative: [
            { name: "criteria-based", approach: "weighted scoring matrix" },
            { name: "pairwise", approach: "direct feature comparison" },
        ],
        causal: [
            { name: "root-cause", approach: "5-whys analysis" },
            { name: "fishbone", approach: "cause-and-effect diagram" },
        ],
        general: [
            { name: "first-principles", approach: "fundamental reasoning" },
            { name: "pattern-matching", approach: "similar case analysis" },
        ],
    };
    
    const typeStrategies = strategies[problemType as keyof typeof strategies] || strategies.general;
    
    // Select based on past performance
    const successRate = performance.success / Math.max(performance.attempts, 1);
    const strategyIndex = successRate > 0.8 ? 0 : 1;
    
    return typeStrategies[Math.min(strategyIndex, typeStrategies.length - 1)];
}

function applyReasoningStrategy(
    strategy: { name: string; approach: string },
    request: LLMRequest,
): { process: string; conclusion: string; confidence: number } {
    return {
        process: `Applying ${strategy.name} strategy using ${strategy.approach}...`,
        conclusion: `Using ${strategy.approach}, I've determined the optimal solution.`,
        confidence: 0.85,
    };
}

function extractProblemStatement(request: LLMRequest): string | null {
    const messages = request.messages.filter(m => m.role === "user");
    if (messages.length === 0) return null;
    
    const lastMessage = messages[messages.length - 1].content;
    
    // Look for problem indicators
    if (lastMessage.includes("?") || 
        lastMessage.match(/solve|fix|help|explain|how|why|what/i)) {
        return lastMessage;
    }
    
    return null;
}

function generateHypotheses(problem: string): Array<{
    id: string;
    hypothesis: string;
    testable: boolean;
}> {
    // Simplified hypothesis generation
    return [
        {
            id: "h1",
            hypothesis: "The issue is caused by incorrect configuration",
            testable: true,
        },
        {
            id: "h2",
            hypothesis: "The problem stems from resource constraints",
            testable: true,
        },
        {
            id: "h3",
            hypothesis: "This is due to external dependencies",
            testable: true,
        },
    ];
}

function testHypothesis(
    hypothesis: { id: string; hypothesis: string; testable: boolean },
    problem: string,
): {
    hypothesis: typeof hypothesis;
    result: "supported" | "rejected" | "inconclusive";
    evidence: string;
    confidence: number;
} {
    // Simplified testing logic
    const randomResult = Math.random();
    
    return {
        hypothesis,
        result: randomResult > 0.7 ? "supported" : randomResult > 0.3 ? "inconclusive" : "rejected",
        evidence: `Testing revealed ${randomResult > 0.7 ? "strong" : "weak"} correlation`,
        confidence: randomResult,
    };
}

function selectBestHypothesis(testResults: any[]): {
    id: string;
    conclusion: string;
    confidence: number;
} {
    const supported = testResults
        .filter(r => r.result === "supported")
        .sort((a, b) => b.confidence - a.confidence);
    
    if (supported.length > 0) {
        return {
            id: supported[0].hypothesis.id,
            conclusion: supported[0].hypothesis.hypothesis,
            confidence: supported[0].confidence,
        };
    }
    
    return {
        id: "none",
        conclusion: "No hypothesis was strongly supported",
        confidence: 0.5,
    };
}

function formatHypothesisTesting(
    hypotheses: any[],
    results: any[],
    selected: any,
): string {
    return `Hypothesis testing process:
${hypotheses.map((h, i) => 
    `- ${h.hypothesis}: ${results[i].result} (${(results[i].confidence * 100).toFixed(0)}% confidence)`,
).join("\n")}

Selected: ${selected.conclusion}`;
}

function extractScenario(request: LLMRequest): any {
    return {
        description: request.messages.map(m => m.content).join(" "),
        type: "general",
    };
}

function buildCausalChain(scenario: any): Array<{
    cause: string;
    effect: string;
    strength: number;
}> {
    // Simplified causal chain
    return [
        { cause: "Initial condition", effect: "Intermediate state", strength: 0.8 },
        { cause: "Intermediate state", effect: "Final outcome", strength: 0.9 },
    ];
}

function identifyRootCause(chain: any[]): { description: string } {
    return { description: chain[0]?.cause || "Unknown root cause" };
}

function predictEffects(rootCause: any, chain: any[]): string[] {
    return chain.map(link => link.effect);
}

function formatCausalConclusion(rootCause: any, effects: string[]): string {
    return `The root cause is ${rootCause.description}, leading to: ${effects.join(" → ")}`;
}

function formatCausalChain(chain: any[]): string {
    return "Causal analysis:\n" + 
           chain.map(link => `${link.cause} → ${link.effect} (strength: ${link.strength})`).join("\n");
}

function calculateCausalConfidence(chain: any[]): number {
    if (chain.length === 0) return 0.5;
    const avgStrength = chain.reduce((sum, link) => sum + link.strength, 0) / chain.length;
    return Math.min(avgStrength, 0.95);
}

function getDefaultPatterns(): Array<{
    domain: string;
    pattern: string;
    solution: string;
}> {
    return [
        {
            domain: "software architecture",
            pattern: "components need loose coupling",
            solution: "use event-driven architecture or dependency injection",
        },
        {
            domain: "performance optimization",
            pattern: "system slows under load",
            solution: "implement caching, optimize queries, or scale horizontally",
        },
        {
            domain: "team collaboration",
            pattern: "communication breakdown between teams",
            solution: "establish clear interfaces and documentation",
        },
    ];
}

function calculateSimilarity(problem: string, pattern: string): number {
    // Simplified similarity calculation
    const problemWords = problem.toLowerCase().split(/\s+/);
    const patternWords = pattern.toLowerCase().split(/\s+/);
    
    const commonWords = problemWords.filter(word => patternWords.includes(word));
    return commonWords.length / Math.max(problemWords.length, patternWords.length);
}

function adaptSolution(solution: string, problem: string): string {
    return `Adapting the solution: ${solution}`;
}

function analyzeOwnThinking(
    request: LLMRequest,
    depth: number,
): {
    needsMoreInfo: boolean;
    uncertainty: string;
    response: string;
    confidence: number;
    insights: string[];
} {
    const hasEnoughContext = request.messages.length >= 3;
    const complexity = estimateComplexity(request);
    
    return {
        needsMoreInfo: !hasEnoughContext && complexity > 0.7,
        uncertainty: "Insufficient context for high-confidence response",
        response: "Based on my analysis at depth " + depth,
        confidence: hasEnoughContext ? 0.85 : 0.6,
        insights: [
            "Pattern recognized",
            "Multiple factors considered",
            "Trade-offs evaluated",
        ],
    };
}

function estimateComplexity(request: LLMRequest): number {
    const factors = [
        request.messages.length > 5 ? 0.2 : 0,
        request.messages.some(m => m.content.length > 500) ? 0.3 : 0,
        request.messages.some(m => m.content.includes("complex")) ? 0.2 : 0,
        request.tools && request.tools.length > 0 ? 0.3 : 0,
    ];
    
    return factors.reduce((sum, factor) => sum + factor, 0);
}
