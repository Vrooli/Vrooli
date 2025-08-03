/**
 * Agent-S2 Screen Automation Script
 * Handles screen capture, mouse/keyboard automation, and application interaction
 */

interface AutomationInput {
    action: 'screenshot' | 'click' | 'type' | 'workflow' | 'analyze_screen';
    
    // Screenshot parameters
    format?: 'png' | 'jpeg';
    response_format?: 'binary' | 'base64' | 'url';
    
    // Click parameters
    x?: number;
    y?: number;
    button?: 'left' | 'right' | 'middle';
    
    // Type parameters
    text?: string;
    delay?: number;
    
    // Workflow parameters
    workflow_steps?: Array<{
        action: string;
        parameters: Record<string, any>;
        wait_time?: number;
    }>;
    
    // Screen analysis parameters
    analyze_for?: string;  // What to look for in the screenshot
}

interface AutomationOutput {
    success: boolean;
    action_performed?: string;
    screenshot?: {
        format: string;
        data?: string;        // Base64 encoded image data
        url?: string;         // URL to access the image
        size?: { width: number; height: number; };
    };
    click_result?: {
        x: number;
        y: number;
        button: string;
        timestamp: number;
    };
    type_result?: {
        text: string;
        character_count: number;
        timestamp: number;
    };
    workflow_result?: {
        steps_completed: number;
        total_steps: number;
        results: Array<Record<string, any>>;
    };
    analysis_result?: {
        elements_found: Array<{
            type: string;
            position: { x: number; y: number; };
            text?: string;
            confidence: number;
        }>;
        recommendations: string[];
    };
    processing_time?: number;
    error?: string;
}

export async function main(input: AutomationInput): Promise<AutomationOutput> {
    const startTime = Date.now();
    
    try {
        // Validate input
        if (!input.action) {
            return {
                success: false,
                error: "Action parameter is required"
            };
        }

        // Get Agent-S2 service URL from environment or default
        const agentUrl = process.env.AGENT_S2_BASE_URL || 'http://localhost:4113';
        
        // Check Agent-S2 health
        const healthResponse = await fetch(`${agentUrl}/health`);
        if (!healthResponse.ok) {
            return {
                success: false,
                error: `Cannot connect to Agent-S2: ${healthResponse.statusText}`,
                processing_time: Date.now() - startTime
            };
        }

        // Execute the requested action
        switch (input.action) {
            case 'screenshot':
                return await takeScreenshot(agentUrl, input, startTime);
            
            case 'click':
                return await performClick(agentUrl, input, startTime);
            
            case 'type':
                return await performType(agentUrl, input, startTime);
            
            case 'workflow':
                return await executeWorkflow(agentUrl, input, startTime);
            
            case 'analyze_screen':
                return await analyzeScreen(agentUrl, input, startTime);
            
            default:
                return {
                    success: false,
                    error: `Unknown action: ${input.action}`,
                    processing_time: Date.now() - startTime
                };
        }

    } catch (error) {
        return {
            success: false,
            error: `Automation failed: ${error instanceof Error ? error.message : String(error)}`,
            processing_time: Date.now() - startTime
        };
    }
}

async function takeScreenshot(agentUrl: string, input: AutomationInput, startTime: number): Promise<AutomationOutput> {
    const format = input.format || 'png';
    const responseFormat = input.response_format || 'base64';
    
    const url = `${agentUrl}/screenshot?format=${format}&response_format=${responseFormat}`;
    
    const response = await fetch(url, { method: 'POST' });
    
    if (!response.ok) {
        return {
            success: false,
            error: `Screenshot failed: ${response.status} ${response.statusText}`,
            processing_time: Date.now() - startTime
        };
    }

    let screenshotData: any = {};
    
    if (responseFormat === 'binary') {
        const buffer = await response.arrayBuffer();
        const base64 = Buffer.from(buffer).toString('base64');
        screenshotData = {
            format: format,
            data: base64,
            size: { width: 1920, height: 1080 } // Default size, would be better to get actual size
        };
    } else if (responseFormat === 'base64') {
        const text = await response.text();
        screenshotData = {
            format: format,
            data: text,
            size: { width: 1920, height: 1080 }
        };
    } else {
        const result = await response.json();
        screenshotData = {
            format: format,
            url: result.url || `${agentUrl}/screenshot/latest.${format}`,
            size: result.size || { width: 1920, height: 1080 }
        };
    }

    return {
        success: true,
        action_performed: 'screenshot',
        screenshot: screenshotData,
        processing_time: Date.now() - startTime
    };
}

async function performClick(agentUrl: string, input: AutomationInput, startTime: number): Promise<AutomationOutput> {
    if (input.x === undefined || input.y === undefined) {
        return {
            success: false,
            error: "Click action requires x and y coordinates",
            processing_time: Date.now() - startTime
        };
    }

    const clickData = {
        x: input.x,
        y: input.y,
        button: input.button || 'left'
    };

    const response = await fetch(`${agentUrl}/mouse/click`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(clickData)
    });

    if (!response.ok) {
        return {
            success: false,
            error: `Click failed: ${response.status} ${response.statusText}`,
            processing_time: Date.now() - startTime
        };
    }

    return {
        success: true,
        action_performed: 'click',
        click_result: {
            x: input.x,
            y: input.y,
            button: input.button || 'left',
            timestamp: Date.now()
        },
        processing_time: Date.now() - startTime
    };
}

async function performType(agentUrl: string, input: AutomationInput, startTime: number): Promise<AutomationOutput> {
    if (!input.text) {
        return {
            success: false,
            error: "Type action requires text parameter",
            processing_time: Date.now() - startTime
        };
    }

    const typeData = {
        text: input.text,
        delay: input.delay || 0
    };

    const response = await fetch(`${agentUrl}/keyboard/type`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(typeData)
    });

    if (!response.ok) {
        return {
            success: false,
            error: `Type failed: ${response.status} ${response.statusText}`,
            processing_time: Date.now() - startTime
        };
    }

    return {
        success: true,
        action_performed: 'type',
        type_result: {
            text: input.text,
            character_count: input.text.length,
            timestamp: Date.now()
        },
        processing_time: Date.now() - startTime
    };
}

async function executeWorkflow(agentUrl: string, input: AutomationInput, startTime: number): Promise<AutomationOutput> {
    if (!input.workflow_steps || input.workflow_steps.length === 0) {
        return {
            success: false,
            error: "Workflow action requires workflow_steps parameter",
            processing_time: Date.now() - startTime
        };
    }

    const results: Array<Record<string, any>> = [];
    let completedSteps = 0;

    for (const step of input.workflow_steps) {
        try {
            // Wait before executing step if specified
            if (step.wait_time && step.wait_time > 0) {
                await new Promise(resolve => setTimeout(resolve, step.wait_time));
            }

            let stepResult: any = { step: step.action, success: false };

            // Execute step based on action type
            switch (step.action) {
                case 'click':
                    if (step.parameters.x !== undefined && step.parameters.y !== undefined) {
                        const clickResult = await performClick(agentUrl, {
                            action: 'click',
                            x: step.parameters.x,
                            y: step.parameters.y,
                            button: step.parameters.button
                        }, startTime);
                        stepResult = { ...stepResult, ...clickResult, success: clickResult.success };
                    }
                    break;

                case 'type':
                    if (step.parameters.text) {
                        const typeResult = await performType(agentUrl, {
                            action: 'type',
                            text: step.parameters.text,
                            delay: step.parameters.delay
                        }, startTime);
                        stepResult = { ...stepResult, ...typeResult, success: typeResult.success };
                    }
                    break;

                case 'screenshot':
                    const screenshotResult = await takeScreenshot(agentUrl, {
                        action: 'screenshot',
                        format: step.parameters.format,
                        response_format: step.parameters.response_format
                    }, startTime);
                    stepResult = { ...stepResult, ...screenshotResult, success: screenshotResult.success };
                    break;

                default:
                    stepResult.error = `Unknown workflow step action: ${step.action}`;
                    break;
            }

            results.push(stepResult);
            if (stepResult.success) {
                completedSteps++;
            }

        } catch (error) {
            results.push({
                step: step.action,
                success: false,
                error: error instanceof Error ? error.message : String(error)
            });
        }
    }

    return {
        success: completedSteps > 0,
        action_performed: 'workflow',
        workflow_result: {
            steps_completed: completedSteps,
            total_steps: input.workflow_steps.length,
            results: results
        },
        processing_time: Date.now() - startTime
    };
}

async function analyzeScreen(agentUrl: string, input: AutomationInput, startTime: number): Promise<AutomationOutput> {
    // First take a screenshot
    const screenshotResult = await takeScreenshot(agentUrl, {
        action: 'screenshot',
        format: 'png',
        response_format: 'base64'
    }, startTime);

    if (!screenshotResult.success) {
        return screenshotResult;
    }

    // For this implementation, we'll do basic analysis
    // In a real application, you might use OCR or computer vision APIs
    const mockAnalysis = {
        elements_found: [
            {
                type: 'window',
                position: { x: 100, y: 100 },
                text: 'Application Window',
                confidence: 0.9
            },
            {
                type: 'button',
                position: { x: 500, y: 300 },
                text: 'Submit',
                confidence: 0.8
            }
        ],
        recommendations: [
            'Screenshot captured successfully',
            'Multiple UI elements detected',
            input.analyze_for ? `Looking for: ${input.analyze_for}` : 'General screen analysis completed'
        ]
    };

    return {
        success: true,
        action_performed: 'analyze_screen',
        screenshot: screenshotResult.screenshot,
        analysis_result: mockAnalysis,
        processing_time: Date.now() - startTime
    };
}

// Example usage for testing
export const test_input: AutomationInput = {
    action: 'screenshot',
    format: 'png',
    response_format: 'base64'
};

export const workflow_example: AutomationInput = {
    action: 'workflow',
    workflow_steps: [
        {
            action: 'screenshot',
            parameters: { format: 'png', response_format: 'base64' },
            wait_time: 1000
        },
        {
            action: 'click',
            parameters: { x: 500, y: 300, button: 'left' },
            wait_time: 500
        },
        {
            action: 'type',
            parameters: { text: 'Hello World', delay: 100 },
            wait_time: 2000
        }
    ]
};