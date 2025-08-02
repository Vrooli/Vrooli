/**
 * Take Screenshot Utility Script
 * Simple wrapper for Agent-S2 screenshot functionality
 */

interface ScreenshotInput {
    format?: 'png' | 'jpeg';
    response_format?: 'binary' | 'base64' | 'url';
}

interface ScreenshotOutput {
    success: boolean;
    screenshot?: {
        format: string;
        data?: string;        // Base64 encoded image data
        url?: string;         // URL to access the image
        size?: { width: number; height: number; };
        timestamp: number;
    };
    processing_time?: number;
    error?: string;
}

export async function main(input: ScreenshotInput = {}): Promise<ScreenshotOutput> {
    const startTime = Date.now();
    
    try {
        console.log('📸 Taking screenshot...');
        
        // Import the agent-s2 automation script
        const { main: performAutomation } = await import('./agent-s2-automation');
        
        // Take screenshot with specified parameters
        const result = await performAutomation({
            action: 'screenshot',
            format: input.format || 'png',
            response_format: input.response_format || 'base64'
        });
        
        if (result.success && result.screenshot) {
            console.log('✅ Screenshot captured successfully');
            
            return {
                success: true,
                screenshot: {
                    ...result.screenshot,
                    timestamp: Date.now()
                },
                processing_time: Date.now() - startTime
            };
        } else {
            console.log(`❌ Screenshot failed: ${result.error}`);
            
            return {
                success: false,
                error: result.error || 'Screenshot capture failed',
                processing_time: Date.now() - startTime
            };
        }
        
    } catch (error) {
        console.log(`❌ Screenshot error: ${error instanceof Error ? error.message : String(error)}`);
        
        return {
            success: false,
            error: `Screenshot failed: ${error instanceof Error ? error.message : String(error)}`,
            processing_time: Date.now() - startTime
        };
    }
}

/**
 * Update application state with screenshot results
 */
export function updateAppStateWithScreenshot(result: ScreenshotOutput): Record<string, any> {
    const state: Record<string, any> = {};
    
    if (result.success && result.screenshot?.data) {
        state.screenshot_data = result.screenshot.data;
        console.log('🖼️ Screenshot data updated in app state');
    } else {
        state.screenshot_data = '';
        console.log('⚠️ Screenshot data cleared from app state');
    }
    
    return state;
}

// Example usage for testing
export const test_input: ScreenshotInput = {
    format: 'png',
    response_format: 'base64'
};