/**
 * Multi-Modal Request Processing Orchestration Script
 * Coordinates the entire workflow: Audio → Text → AI Analysis → Image Generation → Automation
 */

interface ProcessingInput {
    text_input?: string;              // Direct text input
    audio_file?: string;              // Base64 encoded audio file
    temperature?: number;             // AI creativity setting
    image_size?: string;              // Image dimensions (e.g., "512x512")
    session_id?: string;              // Session identifier
    enable_automation?: boolean;       // Whether to include screen automation
}

interface ProcessingOutput {
    success: boolean;
    session_id: string;
    workflow_results: {
        transcription?: {
            success: boolean;
            text?: string;
            processing_time?: number;
            error?: string;
        };
        analysis?: {
            success: boolean;
            intent?: any;
            response?: string;
            processing_time?: number;
            error?: string;
        };
        generation?: {
            success: boolean;
            prompt_id?: string;
            status?: string;
            processing_time?: number;
            error?: string;
        };
        automation?: {
            success: boolean;
            screenshot?: any;
            processing_time?: number;
            error?: string;
        };
    };
    total_processing_time: number;
    final_status: 'completed' | 'partial' | 'failed';
    error?: string;
}

export async function main(input: ProcessingInput): Promise<ProcessingOutput> {
    const startTime = Date.now();
    const sessionId = input.session_id || `session_${Date.now()}`;
    
    const workflowResults: any = {};
    let currentText = input.text_input || '';
    
    try {
        console.log(`🤖 Starting multi-modal processing workflow for session: ${sessionId}`);
        
        // Step 1: Audio Transcription (if audio file provided)
        if (input.audio_file && input.audio_file.length > 0) {
            console.log('🎤 Processing audio transcription...');
            
            try {
                // Import and call the whisper transcription script
                const { main: transcribeAudio } = await import('./whisper-transcribe');
                
                const transcriptionResult = await transcribeAudio({
                    audio_file: input.audio_file,
                    response_format: 'json',
                    temperature: 0.0
                });
                
                workflowResults.transcription = transcriptionResult;
                
                if (transcriptionResult.success && transcriptionResult.text) {
                    currentText = transcriptionResult.text;
                    console.log(`✅ Transcription completed: "${currentText.substring(0, 100)}..."`);
                } else {
                    console.log(`❌ Transcription failed: ${transcriptionResult.error}`);
                }
                
            } catch (error) {
                workflowResults.transcription = {
                    success: false,
                    error: `Transcription script error: ${error instanceof Error ? error.message : String(error)}`
                };
                console.log(`❌ Transcription error: ${workflowResults.transcription.error}`);
            }
        }
        
        // Step 2: Intent Analysis and Response Generation (if we have text)
        if (currentText && currentText.trim().length > 0) {
            console.log('🧠 Analyzing intent and generating response...');
            
            try {
                // Import and call the ollama analysis script
                const { main: analyzeIntent } = await import('./ollama-analyze');
                
                const analysisResult = await analyzeIntent({
                    text: currentText,
                    task_type: 'intent',
                    temperature: input.temperature || 0.7
                });
                
                workflowResults.analysis = analysisResult;
                
                if (analysisResult.success) {
                    console.log(`✅ Analysis completed. Intent: ${analysisResult.intent?.primary_action || 'Unknown'}`);
                } else {
                    console.log(`❌ Analysis failed: ${analysisResult.error}`);
                }
                
            } catch (error) {
                workflowResults.analysis = {
                    success: false,
                    error: `Analysis script error: ${error instanceof Error ? error.message : String(error)}`
                };
                console.log(`❌ Analysis error: ${workflowResults.analysis.error}`);
            }
        }
        
        // Step 3: Image Generation (if visual capabilities are required)
        const requiresVisualGeneration = workflowResults.analysis?.intent?.required_capabilities?.includes('image') ||
                                       currentText.toLowerCase().includes('image') ||
                                       currentText.toLowerCase().includes('logo') ||
                                       currentText.toLowerCase().includes('visual') ||
                                       currentText.toLowerCase().includes('create') ||
                                       currentText.toLowerCase().includes('generate');
        
        if (requiresVisualGeneration && currentText) {
            console.log('🎨 Starting image generation...');
            
            try {
                // Import and call the comfyui generation script
                const { main: generateImage } = await import('./comfyui-generate');
                
                // Parse image size
                const [width, height] = (input.image_size || '512x512').split('x').map(n => parseInt(n));
                
                // Create image generation prompt from the analysis or original text
                let imagePrompt = currentText;
                if (workflowResults.analysis?.response) {
                    // Try to extract a better prompt from the AI response
                    imagePrompt = extractImagePrompt(workflowResults.analysis.response, currentText);
                }
                
                const generationResult = await generateImage({
                    prompt: imagePrompt,
                    negative_prompt: "blur, low quality, distorted, ugly, bad anatomy, unprofessional",
                    width: width,
                    height: height,
                    steps: 25,
                    cfg_scale: 7.5,
                    filename_prefix: `multimodal_${sessionId}`
                });
                
                workflowResults.generation = generationResult;
                
                if (generationResult.success) {
                    console.log(`✅ Image generation queued. Prompt ID: ${generationResult.prompt_id}`);
                } else {
                    console.log(`❌ Image generation failed: ${generationResult.error}`);
                }
                
            } catch (error) {
                workflowResults.generation = {
                    success: false,
                    error: `Generation script error: ${error instanceof Error ? error.message : String(error)}`
                };
                console.log(`❌ Generation error: ${workflowResults.generation.error}`);
            }
        }
        
        // Step 4: Screen Automation (if requested and automation capabilities are required)
        const requiresAutomation = workflowResults.analysis?.intent?.required_capabilities?.includes('automation') ||
                                  input.enable_automation === true;
        
        if (requiresAutomation) {
            console.log('🖥️ Performing screen automation...');
            
            try {
                // Import and call the agent-s2 automation script
                const { main: performAutomation } = await import('./agent-s2-automation');
                
                // Take a screenshot to demonstrate automation capability
                const automationResult = await performAutomation({
                    action: 'screenshot',
                    format: 'png',
                    response_format: 'base64'
                });
                
                workflowResults.automation = automationResult;
                
                if (automationResult.success) {
                    console.log('✅ Screen automation completed');
                } else {
                    console.log(`❌ Screen automation failed: ${automationResult.error}`);
                }
                
            } catch (error) {
                workflowResults.automation = {
                    success: false,
                    error: `Automation script error: ${error instanceof Error ? error.message : String(error)}`
                };
                console.log(`❌ Automation error: ${workflowResults.automation.error}`);
            }
        }
        
        // Determine final status
        const completedSteps = Object.values(workflowResults).filter((result: any) => result.success).length;
        const totalSteps = Object.keys(workflowResults).length;
        
        let finalStatus: 'completed' | 'partial' | 'failed';
        if (completedSteps === totalSteps && totalSteps > 0) {
            finalStatus = 'completed';
        } else if (completedSteps > 0) {
            finalStatus = 'partial';
        } else {
            finalStatus = 'failed';
        }
        
        const totalProcessingTime = Date.now() - startTime;
        
        console.log(`🎯 Workflow completed: ${finalStatus} (${completedSteps}/${totalSteps} steps succeeded)`);
        console.log(`⏱️ Total processing time: ${totalProcessingTime}ms`);
        
        return {
            success: finalStatus !== 'failed',
            session_id: sessionId,
            workflow_results: workflowResults,
            total_processing_time: totalProcessingTime,
            final_status: finalStatus
        };
        
    } catch (error) {
        const totalProcessingTime = Date.now() - startTime;
        
        return {
            success: false,
            session_id: sessionId,
            workflow_results: workflowResults,
            total_processing_time: totalProcessingTime,
            final_status: 'failed',
            error: `Workflow orchestration failed: ${error instanceof Error ? error.message : String(error)}`
        };
    }
}

/**
 * Extract or enhance image generation prompt from AI analysis
 */
function extractImagePrompt(aiResponse: string, originalText: string): string {
    // Try to find image-related instructions in the AI response
    const imageKeywords = ['image', 'visual', 'picture', 'logo', 'design', 'create', 'generate'];
    const sentences = aiResponse.split(/[.!?]+/);
    
    for (const sentence of sentences) {
        const lowerSentence = sentence.toLowerCase();
        if (imageKeywords.some(keyword => lowerSentence.includes(keyword))) {
            // Found a sentence that mentions images, use it as the prompt
            if (sentence.trim().length > 20) {
                return sentence.trim();
            }
        }
    }
    
    // Fallback to original text with some enhancements
    let enhancedPrompt = originalText;
    
    // Add professional quality modifiers if not present
    if (!enhancedPrompt.toLowerCase().includes('professional')) {
        enhancedPrompt = `professional ${enhancedPrompt}`;
    }
    
    if (!enhancedPrompt.toLowerCase().includes('high quality')) {
        enhancedPrompt = `${enhancedPrompt}, high quality`;
    }
    
    return enhancedPrompt;
}

/**
 * Update application state with processing results
 */
export function updateAppState(results: ProcessingOutput): Record<string, any> {
    const state: Record<string, any> = {
        processing: false,
        session_id: results.session_id,
        last_result: results
    };
    
    // Update transcription state
    if (results.workflow_results.transcription) {
        state.transcription_complete = results.workflow_results.transcription.success;
        state.transcription_processing = false;
        if (results.workflow_results.transcription.text) {
            state.transcription_result = results.workflow_results.transcription.text;
        }
    }
    
    // Update analysis state
    if (results.workflow_results.analysis) {
        state.analysis_complete = results.workflow_results.analysis.success;
        state.analysis_processing = false;
        if (results.workflow_results.analysis.success) {
            state.analysis_result = results.workflow_results.analysis;
        }
    }
    
    // Update generation state
    if (results.workflow_results.generation) {
        state.generation_complete = results.workflow_results.generation.success;
        state.generation_processing = false;
        if (results.workflow_results.generation.prompt_id) {
            // In a real implementation, you might poll for completion and update images
            state.generated_images = [{
                prompt_id: results.workflow_results.generation.prompt_id,
                status: results.workflow_results.generation.status || 'queued'
            }];
        }
    }
    
    // Update automation state
    if (results.workflow_results.automation) {
        state.automation_complete = results.workflow_results.automation.success;
        state.automation_processing = false;
        if (results.workflow_results.automation.screenshot?.data) {
            state.screenshot_data = results.workflow_results.automation.screenshot.data;
        }
    }
    
    return state;
}

// Example usage for testing
export const test_input: ProcessingInput = {
    text_input: "Create a professional logo for TechCorp with blue and silver colors",
    temperature: 0.7,
    image_size: "512x512",
    enable_automation: true
};