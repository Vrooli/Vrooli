/**
 * ComfyUI Image Generation Script
 * Creates images using ComfyUI workflows and Stable Diffusion models
 */

interface ImageGenerationInput {
    prompt: string;                // Text prompt for image generation
    negative_prompt?: string;      // Negative prompt to avoid unwanted elements
    width?: number;               // Image width (default: 512)
    height?: number;              // Image height (default: 512)
    steps?: number;               // Sampling steps (default: 20)
    cfg_scale?: number;           // CFG scale (default: 7.0)
    seed?: number;                // Random seed (default: random)
    model?: string;               // Model name (default: sd_xl_base_1.0.safetensors)
    sampler?: string;             // Sampler name (default: euler)
    scheduler?: string;           // Scheduler (default: normal)
    batch_size?: number;          // Number of images to generate (default: 1)
    filename_prefix?: string;     // Prefix for saved files
}

interface ImageGenerationOutput {
    success: boolean;
    prompt_id?: string;           // ComfyUI prompt ID for tracking
    images?: Array<{
        filename: string;
        url?: string;
        base64?: string;
    }>;
    generation_parameters?: Record<string, any>;
    processing_time?: number;
    queue_position?: number;
    error?: string;
    status?: 'queued' | 'running' | 'completed' | 'failed';
}

export async function main(input: ImageGenerationInput): Promise<ImageGenerationOutput> {
    const startTime = Date.now();
    
    try {
        // Validate input
        if (!input.prompt || input.prompt.trim().length === 0) {
            return {
                success: false,
                error: "Prompt cannot be empty"
            };
        }

        // Get ComfyUI service URL from environment or default
        const comfyuiUrl = process.env.COMFYUI_BASE_URL || 'http://localhost:8188';
        
        // Check ComfyUI system status
        const statusResponse = await fetch(`${comfyuiUrl}/system_stats`);
        if (!statusResponse.ok) {
            return {
                success: false,
                error: `Cannot connect to ComfyUI: ${statusResponse.statusText}`,
                processing_time: Date.now() - startTime
            };
        }

        // Set default parameters
        const parameters = {
            prompt: input.prompt,
            negative_prompt: input.negative_prompt || "blur, low quality, distorted, ugly, bad anatomy",
            width: input.width || 512,
            height: input.height || 512,
            steps: input.steps || 20,
            cfg_scale: input.cfg_scale || 7.0,
            seed: input.seed || Math.floor(Math.random() * 1000000),
            model: input.model || "sd_xl_base_1.0.safetensors",
            sampler: input.sampler || "euler",
            scheduler: input.scheduler || "normal",
            batch_size: input.batch_size || 1,
            filename_prefix: input.filename_prefix || "windmill_generated"
        };

        // Create ComfyUI workflow definition
        const workflow = createComfyUIWorkflow(parameters);

        // Submit workflow to ComfyUI
        const submitResponse = await fetch(`${comfyuiUrl}/prompt`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(workflow)
        });

        if (!submitResponse.ok) {
            return {
                success: false,
                error: `Failed to submit workflow: ${submitResponse.status} ${submitResponse.statusText}`,
                processing_time: Date.now() - startTime
            };
        }

        const submitResult = await submitResponse.json();
        const promptId = submitResult.prompt_id;

        if (!promptId) {
            return {
                success: false,
                error: "No prompt ID returned from ComfyUI",
                processing_time: Date.now() - startTime
            };
        }

        // For this implementation, we'll return the prompt ID and status
        // In a real application, you might want to poll for completion
        const queueResponse = await fetch(`${comfyuiUrl}/queue`);
        let queuePosition = 0;
        
        if (queueResponse.ok) {
            const queueData = await queueResponse.json();
            // Find position in queue
            if (queueData.queue_running && queueData.queue_running.length > 0) {
                const runningIndex = queueData.queue_running.findIndex((item: any) => item[1] === promptId);
                if (runningIndex >= 0) {
                    queuePosition = runningIndex + 1;
                }
            }
            if (queueData.queue_pending && queueData.queue_pending.length > 0) {
                const pendingIndex = queueData.queue_pending.findIndex((item: any) => item[1] === promptId);
                if (pendingIndex >= 0) {
                    queuePosition = pendingIndex + 1 + (queueData.queue_running?.length || 0);
                }
            }
        }

        return {
            success: true,
            prompt_id: promptId,
            status: 'queued',
            queue_position: queuePosition,
            generation_parameters: parameters,
            processing_time: Date.now() - startTime
        };

    } catch (error) {
        return {
            success: false,
            error: `Image generation failed: ${error instanceof Error ? error.message : String(error)}`,
            processing_time: Date.now() - startTime
        };
    }
}

function createComfyUIWorkflow(params: any): any {
    return {
        prompt: {
            "1": {
                "class_type": "CheckpointLoaderSimple",
                "inputs": {
                    "ckpt_name": params.model
                }
            },
            "2": {
                "class_type": "CLIPTextEncode",
                "inputs": {
                    "text": params.prompt,
                    "clip": ["1", 1]
                }
            },
            "3": {
                "class_type": "CLIPTextEncode",
                "inputs": {
                    "text": params.negative_prompt,
                    "clip": ["1", 1]
                }
            },
            "4": {
                "class_type": "EmptyLatentImage",
                "inputs": {
                    "width": params.width,
                    "height": params.height,
                    "batch_size": params.batch_size
                }
            },
            "5": {
                "class_type": "KSampler",
                "inputs": {
                    "seed": params.seed,
                    "steps": params.steps,
                    "cfg": params.cfg_scale,
                    "sampler_name": params.sampler,
                    "scheduler": params.scheduler,
                    "denoise": 1.0,
                    "model": ["1", 0],
                    "positive": ["2", 0],
                    "negative": ["3", 0],
                    "latent_image": ["4", 0]
                }
            },
            "6": {
                "class_type": "VAEDecode",
                "inputs": {
                    "samples": ["5", 0],
                    "vae": ["1", 2]
                }
            },
            "7": {
                "class_type": "SaveImage",
                "inputs": {
                    "filename_prefix": params.filename_prefix,
                    "images": ["6", 0]
                }
            }
        }
    };
}

/**
 * Check generation status by prompt ID
 */
export async function checkStatus(promptId: string): Promise<ImageGenerationOutput> {
    try {
        const comfyuiUrl = process.env.COMFYUI_BASE_URL || 'http://localhost:8188';
        
        // Check history for completed generations
        const historyResponse = await fetch(`${comfyuiUrl}/history/${promptId}`);
        if (historyResponse.ok) {
            const historyData = await historyResponse.json();
            
            if (historyData[promptId]) {
                const historyItem = historyData[promptId];
                
                if (historyItem.status?.completed) {
                    // Generation completed, extract image information
                    const outputs = historyItem.outputs;
                    const images: any[] = [];
                    
                    // Look for SaveImage node outputs
                    for (const nodeId in outputs) {
                        const nodeOutput = outputs[nodeId];
                        if (nodeOutput.images) {
                            for (const img of nodeOutput.images) {
                                images.push({
                                    filename: img.filename,
                                    url: `${comfyuiUrl}/view?filename=${img.filename}&subfolder=${img.subfolder || ''}&type=${img.type || 'output'}`
                                });
                            }
                        }
                    }
                    
                    return {
                        success: true,
                        prompt_id: promptId,
                        status: 'completed',
                        images: images
                    };
                } else {
                    return {
                        success: true,
                        prompt_id: promptId,
                        status: 'running'
                    };
                }
            }
        }
        
        // Check if still in queue
        const queueResponse = await fetch(`${comfyuiUrl}/queue`);
        if (queueResponse.ok) {
            const queueData = await queueResponse.json();
            
            // Check running queue
            if (queueData.queue_running?.some((item: any) => item[1] === promptId)) {
                return {
                    success: true,
                    prompt_id: promptId,
                    status: 'running'
                };
            }
            
            // Check pending queue
            const pendingIndex = queueData.queue_pending?.findIndex((item: any) => item[1] === promptId);
            if (pendingIndex >= 0) {
                return {
                    success: true,
                    prompt_id: promptId,
                    status: 'queued',
                    queue_position: pendingIndex + 1
                };
            }
        }
        
        return {
            success: false,
            error: "Generation not found in queue or history"
        };
        
    } catch (error) {
        return {
            success: false,
            error: `Status check failed: ${error instanceof Error ? error.message : String(error)}`
        };
    }
}

// Example usage for testing
export const test_input: ImageGenerationInput = {
    prompt: "professional minimalist logo design, tech startup, clean lines, modern typography, blue and silver colors",
    negative_prompt: "blur, low quality, complex, cluttered, unprofessional",
    width: 512,
    height: 512,
    steps: 25,
    cfg_scale: 7.5,
    filename_prefix: "multimodal_test"
};