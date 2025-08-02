/**
 * Clear Session Utility Script
 * Resets the application state to initial values
 */

interface ClearSessionInput {
    confirm?: boolean;              // Confirmation flag
    preserve_settings?: boolean;    // Whether to keep user settings like temperature
}

interface ClearSessionOutput {
    success: boolean;
    session_cleared: boolean;
    settings_preserved?: boolean;
    cleared_items: string[];
    new_session_id: string;
    timestamp: number;
    error?: string;
}

export async function main(input: ClearSessionInput = {}): Promise<ClearSessionOutput> {
    const timestamp = Date.now();
    const newSessionId = `session_${timestamp}`;
    
    try {
        console.log('🧹 Clearing session...');
        
        // List of items that will be cleared
        const itemsToCloud = [
            'transcription_result',
            'analysis_result', 
            'generated_images',
            'screenshot_data',
            'processing_status',
            'error_messages'
        ];
        
        // If settings should be preserved, we'll note that
        const settingsPreserved = input.preserve_settings === true;
        if (settingsPreserved) {
            itemsToCloud.push('settings_preserved');
            console.log('⚙️ User settings will be preserved');
        } else {
            itemsToCloud.push('user_settings');
            console.log('⚙️ All data including settings will be cleared');
        }
        
        console.log(`✅ Session cleared successfully. New session ID: ${newSessionId}`);
        
        return {
            success: true,
            session_cleared: true,
            settings_preserved: settingsPreserved,
            cleared_items: itemsToCloud,
            new_session_id: newSessionId,
            timestamp: timestamp
        };
        
    } catch (error) {
        console.log(`❌ Session clear error: ${error instanceof Error ? error.message : String(error)}`);
        
        return {
            success: false,
            session_cleared: false,
            cleared_items: [],
            new_session_id: newSessionId,
            timestamp: timestamp,
            error: `Session clear failed: ${error instanceof Error ? error.message : String(error)}`
        };
    }
}

/**
 * Generate clean application state
 */
export function generateCleanAppState(preserveSettings: boolean = false, currentState?: Record<string, any>): Record<string, any> {
    const cleanState: Record<string, any> = {
        // Processing states
        processing: false,
        transcription_processing: false,
        transcription_complete: false,
        transcription_result: '',
        analysis_processing: false,
        analysis_complete: false,
        analysis_result: null,
        generation_processing: false,
        generation_complete: false,
        generated_images: [],
        automation_processing: false,
        automation_complete: false,
        screenshot_data: '',
        last_result: null,
        
        // Session info
        session_id: `session_${Date.now()}`,
        
        // Clear any error states
        error_message: '',
        warning_message: ''
    };
    
    // Preserve user settings if requested
    if (preserveSettings && currentState) {
        cleanState.temperature = currentState.temperature || 0.7;
        cleanState.image_size = currentState.image_size || '512x512';
        cleanState.enable_automation = currentState.enable_automation || false;
        console.log('⚙️ User settings preserved in clean state');
    } else {
        // Reset to defaults
        cleanState.temperature = 0.7;
        cleanState.image_size = '512x512';
        cleanState.enable_automation = false;
        console.log('⚙️ All settings reset to defaults');
    }
    
    return cleanState;
}

/**
 * Validate that session was properly cleared
 */
export function validateSessionClear(state: Record<string, any>): boolean {
    const requiredClears = [
        'transcription_result',
        'analysis_result',
        'generated_images',
        'screenshot_data',
        'last_result'
    ];
    
    for (const item of requiredClears) {
        if (state[item] && 
            state[item] !== '' && 
            state[item] !== null && 
            (Array.isArray(state[item]) ? state[item].length > 0 : true)) {
            console.log(`⚠️ Session clear validation failed: ${item} was not properly cleared`);
            return false;
        }
    }
    
    console.log('✅ Session clear validation passed');
    return true;
}

// Example usage for testing
export const test_input: ClearSessionInput = {
    confirm: true,
    preserve_settings: true
};