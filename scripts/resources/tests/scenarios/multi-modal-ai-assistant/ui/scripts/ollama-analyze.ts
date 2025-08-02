/**
 * Ollama Intent Analysis and Response Generation Script
 * Analyzes user input and generates intelligent responses with execution plans
 */

interface AnalysisInput {
    text: string;                  // Input text to analyze
    model?: string;                // Ollama model to use
    context?: string;              // Previous conversation context
    task_type?: 'intent' | 'response' | 'plan' | 'creative';
    system_prompt?: string;        // Custom system prompt
    temperature?: number;          // Sampling temperature (0-1)
    max_tokens?: number;           // Maximum response length
}

interface AnalysisOutput {
    success: boolean;
    response?: string;
    intent?: {
        primary_action: string;
        parameters: Record<string, any>;
        confidence: number;
        required_capabilities: string[];
    };
    execution_plan?: {
        steps: Array<{
            service: string;
            action: string;
            parameters: Record<string, any>;
        }>;
        estimated_time: number;
    };
    error?: string;
    processing_time?: number;
    model_used?: string;
}

export async function main(input: AnalysisInput): Promise<AnalysisOutput> {
    const startTime = Date.now();
    
    try {
        // Validate input
        if (!input.text || input.text.trim().length === 0) {
            return {
                success: false,
                error: "Input text cannot be empty"
            };
        }

        // Get Ollama service URL from environment or default
        const ollamaUrl = process.env.OLLAMA_BASE_URL || 'http://localhost:11434';
        
        // First, get available models
        const modelsResponse = await fetch(`${ollamaUrl}/api/tags`);
        if (!modelsResponse.ok) {
            return {
                success: false,
                error: `Cannot connect to Ollama: ${modelsResponse.statusText}`,
                processing_time: Date.now() - startTime
            };
        }

        const modelsData = await modelsResponse.json();
        const availableModels = modelsData.models || [];
        
        if (availableModels.length === 0) {
            return {
                success: false,
                error: "No Ollama models available",
                processing_time: Date.now() - startTime
            };
        }

        // Select model (use provided model or first available)
        const selectedModel = input.model || availableModels[0].name;

        // Create system prompt based on task type
        let systemPrompt = input.system_prompt || getSystemPrompt(input.task_type || 'intent');
        
        // Create the full prompt
        const fullPrompt = createPrompt(input.text, input.context, systemPrompt, input.task_type);

        // Make request to Ollama
        const generateRequest = {
            model: selectedModel,
            prompt: fullPrompt,
            stream: false,
            options: {
                temperature: input.temperature || 0.7,
                num_predict: input.max_tokens || 500
            }
        };

        const response = await fetch(`${ollamaUrl}/api/generate`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(generateRequest)
        });

        if (!response.ok) {
            return {
                success: false,
                error: `Ollama API error: ${response.status} ${response.statusText}`,
                processing_time: Date.now() - startTime
            };
        }

        const result = await response.json();
        const rawResponse = result.response || '';

        // Parse the response based on task type
        const parsedResult = parseResponse(rawResponse, input.task_type || 'intent');

        return {
            success: true,
            response: rawResponse,
            intent: parsedResult.intent,
            execution_plan: parsedResult.execution_plan,
            processing_time: Date.now() - startTime,
            model_used: selectedModel
        };

    } catch (error) {
        return {
            success: false,
            error: `Analysis failed: ${error instanceof Error ? error.message : String(error)}`,
            processing_time: Date.now() - startTime
        };
    }
}

function getSystemPrompt(taskType: string): string {
    const prompts = {
        intent: `You are an AI assistant that analyzes user commands to extract intent and parameters. 
                Respond with JSON format including: primary_action, parameters, confidence (0-1), 
                and required_capabilities (array of: "text", "image", "audio", "web", "automation").`,
        
        response: `You are a helpful AI assistant. Provide clear, concise, and helpful responses 
                  to user queries. Be professional and informative.`,
        
        plan: `You are an AI workflow planner. Create detailed execution plans with specific steps, 
              services needed, and time estimates. Format as JSON with steps array.`,
        
        creative: `You are a creative AI assistant specializing in visual and multimedia content. 
                  Help users create prompts and specifications for images, audio, and interactive content.`
    };
    
    return prompts[taskType as keyof typeof prompts] || prompts.response;
}

function createPrompt(text: string, context?: string, systemPrompt?: string, taskType?: string): string {
    let prompt = '';
    
    if (systemPrompt) {
        prompt += `${systemPrompt}\n\n`;
    }
    
    if (context) {
        prompt += `Previous conversation context:\n${context}\n\n`;
    }
    
    prompt += `User input: ${text}\n\n`;
    
    if (taskType === 'intent') {
        prompt += `Analyze this input and provide a JSON response with the user's intent, extracted parameters, confidence level, and required AI capabilities.`;
    } else if (taskType === 'plan') {
        prompt += `Create a detailed execution plan for this request. Include specific steps, services needed, and time estimates.`;
    }
    
    return prompt;
}

function parseResponse(response: string, taskType: string): any {
    const result: any = {};
    
    if (taskType === 'intent') {
        // Try to extract JSON from response
        try {
            const jsonMatch = response.match(/\{[\s\S]*\}/);
            if (jsonMatch) {
                const parsed = JSON.parse(jsonMatch[0]);
                result.intent = {
                    primary_action: parsed.primary_action || 'unknown',
                    parameters: parsed.parameters || {},
                    confidence: parsed.confidence || 0.8,
                    required_capabilities: parsed.required_capabilities || ['text']
                };
            } else {
                // Fallback parsing
                result.intent = {
                    primary_action: extractAction(response),
                    parameters: {},
                    confidence: 0.7,
                    required_capabilities: detectCapabilities(response)
                };
            }
        } catch (e) {
            result.intent = {
                primary_action: extractAction(response),
                parameters: {},
                confidence: 0.6,
                required_capabilities: detectCapabilities(response)
            };
        }
    }
    
    if (taskType === 'plan') {
        // Try to extract execution plan
        try {
            const jsonMatch = response.match(/\{[\s\S]*\}/);
            if (jsonMatch) {
                const parsed = JSON.parse(jsonMatch[0]);
                result.execution_plan = parsed;
            }
        } catch (e) {
            // Fallback: create basic plan structure
            result.execution_plan = {
                steps: [{ service: 'ollama', action: 'analyze', parameters: {} }],
                estimated_time: 30
            };
        }
    }
    
    return result;
}

function extractAction(text: string): string {
    const actions = ['create', 'generate', 'analyze', 'process', 'search', 'transcribe', 'translate'];
    const lowerText = text.toLowerCase();
    
    for (const action of actions) {
        if (lowerText.includes(action)) {
            return action;
        }
    }
    
    return 'analyze';
}

function detectCapabilities(text: string): string[] {
    const capabilities = [];
    const lowerText = text.toLowerCase();
    
    if (lowerText.includes('image') || lowerText.includes('visual') || lowerText.includes('picture')) {
        capabilities.push('image');
    }
    if (lowerText.includes('audio') || lowerText.includes('voice') || lowerText.includes('sound')) {
        capabilities.push('audio');
    }
    if (lowerText.includes('search') || lowerText.includes('web') || lowerText.includes('browse')) {
        capabilities.push('web');
    }
    if (lowerText.includes('automation') || lowerText.includes('control') || lowerText.includes('click')) {
        capabilities.push('automation');
    }
    
    capabilities.push('text'); // Always include text capability
    
    return capabilities;
}

// Example usage for testing
export const test_input: AnalysisInput = {
    text: "Create a professional logo for TechCorp with blue and silver colors",
    task_type: 'intent',
    temperature: 0.7
};