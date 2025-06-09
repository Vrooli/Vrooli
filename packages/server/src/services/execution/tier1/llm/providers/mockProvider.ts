/**
 * Mock LLM Provider - For testing and development
 * 
 * Provides simulated LLM responses for development and testing.
 * Replace with actual provider (OpenAI, Anthropic, etc.) in production.
 */

import { LLMProvider, CompletionRequest, CompletionResponse } from "../llmService.js";

/**
 * Mock LLM Provider
 * 
 * Generates synthetic responses for testing Tier 1 reasoning capabilities.
 * Responses are contextually relevant but not from a real LLM.
 */
export class MockLLMProvider implements LLMProvider {
    name = "mock";

    async complete(request: CompletionRequest): Promise<CompletionResponse> {
        // Simulate processing delay
        await new Promise(resolve => setTimeout(resolve, 100 + Math.random() * 200));

        // Generate contextual response based on prompt content
        const text = this.generateResponse(request.prompt, request.systemPrompt);
        
        // Simulate token usage
        const promptTokens = this.estimateTokens(request.prompt);
        const completionTokens = this.estimateTokens(text);

        return {
            text,
            usage: {
                promptTokens,
                completionTokens,
                totalTokens: promptTokens + completionTokens,
            },
            model: "mock-llm-v1",
            finishReason: "stop",
        };
    }

    async isAvailable(): Promise<boolean> {
        return true; // Mock provider is always available
    }

    private generateResponse(prompt: string, systemPrompt?: string): string {
        const lowerPrompt = prompt.toLowerCase();

        // Strategic analysis responses
        if (lowerPrompt.includes("strategic") || lowerPrompt.includes("analyze")) {
            return this.generateStrategicAnalysis(prompt);
        }

        // Decision generation responses
        if (lowerPrompt.includes("decision") || lowerPrompt.includes("choose")) {
            return this.generateDecision(prompt);
        }

        // Situation assessment responses
        if (lowerPrompt.includes("situation") || lowerPrompt.includes("assessment")) {
            return this.generateSituationAssessment(prompt);
        }

        // Resource allocation responses
        if (lowerPrompt.includes("resource") || lowerPrompt.includes("allocation")) {
            return this.generateResourceRecommendation(prompt);
        }

        // Default response
        return this.generateGenericResponse(prompt);
    }

    private generateStrategicAnalysis(prompt: string): string {
        const analyses = [
            "Based on the current context, I recommend a multi-phase approach focusing on resource optimization and risk mitigation. The primary strategic considerations are operational efficiency and scalability.",
            "The situation requires balancing immediate execution needs with long-term strategic positioning. Key factors include resource constraints, time pressures, and quality requirements.",
            "Strategic analysis indicates opportunities for parallel execution and resource reallocation. The critical path should focus on high-impact, low-risk activities.",
            "Current strategic assessment suggests prioritizing adaptive approaches over rigid planning. The dynamic environment requires flexible response capabilities.",
        ];
        
        return analyses[Math.floor(Math.random() * analyses.length)];
    }

    private generateDecision(prompt: string): string {
        const decisions = [
            "allocate_resources(300)\nRationale: Current budget allows for increased resource allocation to accelerate critical path completion.",
            "execute_routine(parallel_approach)\nRationale: Multiple independent tasks can be executed simultaneously to optimize overall completion time.",
            "adapt_strategy(conservative)\nRationale: Given current risk factors, a conservative approach will ensure stable progress toward objectives.",
            "form_team(specialized)\nRationale: Task complexity requires specialized expertise for optimal execution quality.",
        ];
        
        return decisions[Math.floor(Math.random() * decisions.length)];
    }

    private generateSituationAssessment(prompt: string): string {
        const assessments = [
            "Current situation shows steady progress with moderate resource utilization. Key opportunities include process optimization and parallel task execution.",
            "Assessment indicates strong momentum with emerging bottlenecks in coordination. Recommend focusing on communication efficiency and resource rebalancing.",
            "Situation analysis reveals robust execution patterns with potential for acceleration. Strategic adjustments could improve overall throughput.",
            "Current state demonstrates effective resource management with room for strategic optimization. Consider adaptive planning approaches.",
        ];
        
        return assessments[Math.floor(Math.random() * assessments.length)];
    }

    private generateResourceRecommendation(prompt: string): string {
        const recommendations = [
            "Recommend reallocating 20% of available resources to critical path activities while maintaining buffer capacity for unexpected requirements.",
            "Optimal resource distribution suggests focusing primary allocation on high-priority tasks with distributed support for secondary objectives.",
            "Resource analysis indicates efficiency gains through consolidation of similar activities and parallel processing of independent tasks.",
            "Strategic resource management should prioritize flexible allocation patterns that can adapt to changing execution requirements.",
        ];
        
        return recommendations[Math.floor(Math.random() * recommendations.length)];
    }

    private generateGenericResponse(prompt: string): string {
        const responses = [
            "Based on the provided context, I recommend proceeding with a balanced approach that considers both immediate needs and strategic objectives.",
            "The current situation suggests implementing adaptive strategies that can respond effectively to changing conditions and requirements.",
            "Strategic coordination should focus on optimizing resource utilization while maintaining flexibility for dynamic adjustments.",
            "Recommend establishing clear priorities and execution pathways that align with overall strategic objectives and operational constraints.",
        ];
        
        return responses[Math.floor(Math.random() * responses.length)];
    }

    private estimateTokens(text: string): number {
        // Rough token estimation (4 characters per token on average)
        return Math.ceil(text.length / 4);
    }
}