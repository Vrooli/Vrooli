import { logger } from "../events/logger.js";

/**
 * Wrapper for Sharp image processing library with graceful fallback
 * This allows the application to continue running even if Sharp fails to load
 */

// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-02
// Type definitions for Sharp-like interface
interface ResizeOptions {
    withoutEnlargement?: boolean;
    fit?: "cover" | "contain" | "fill" | "inside" | "outside";
    position?: string | number;
    background?: string | { r: number; g: number; b: number; alpha?: number };
    kernel?: "nearest" | "cubic" | "mitchell" | "lanczos2" | "lanczos3";
}

interface FlattenOptions {
    background?: string | { r: number; g: number; b: number; alpha?: number };
}

interface SharpConstructorOptions {
    animated?: boolean;
    failOn?: "none" | "truncated" | "error" | "warning";
    limitInputPixels?: number | boolean;
    sequentialRead?: boolean;
    density?: number;
    pages?: number;
    page?: number;
    subifd?: number;
}

interface SharpInstance {
    resize(width: number, height: number, options?: ResizeOptions): SharpInstance;
    flatten(options?: FlattenOptions): SharpInstance;
    toFormat(format: string): SharpInstance;
    toBuffer(): Promise<Buffer>;
}

interface SharpModule {
    (input: Buffer | Uint8Array | string, options?: SharpConstructorOptions): SharpInstance;
}

// State tracking
let sharpModule: SharpModule | null = null;
let sharpLoadAttempted = false;
let sharpLoadError: Error | null = null;

/**
 * Attempts to load Sharp module dynamically
 */
async function loadSharp(): Promise<SharpModule | null> {
    if (sharpLoadAttempted) {
        return sharpModule;
    }

    sharpLoadAttempted = true;

    try {
        const sharp = await import("sharp");
        sharpModule = sharp as unknown as SharpModule;
        logger.info("[SharpWrapper] Sharp module loaded successfully");
        return sharpModule;
    } catch (error) {
        sharpLoadError = error instanceof Error ? error : new Error(String(error));
        logger.error("[SharpWrapper] Failed to load Sharp module", {
            error: sharpLoadError.message,
            stack: sharpLoadError.stack,
        });
        logger.warn("[SharpWrapper] Image processing features will be disabled");
        return null;
    }
}

/**
 * Checks if Sharp is available
 */
export async function isSharpAvailable(): Promise<boolean> {
    const sharp = await loadSharp();
    return sharp !== null;
}

/**
 * Gets the Sharp loading error if any
 */
export function getSharpError(): Error | null {
    return sharpLoadError;
}

/**
 * Creates a no-op Sharp instance for fallback
 */
function createNoOpSharpInstance(buffer: Buffer): SharpInstance {
    return {
        resize() { return this; },
        flatten() { return this; },
        toFormat() { return this; },
        async toBuffer() { 
            // Return original buffer unchanged
            return buffer;
        },
    };
}

/**
 * Wrapper function that mimics Sharp's API but handles failures gracefully
 */
export async function getSharp(): Promise<SharpModule | null> {
    return await loadSharp();
}

/**
 * Safe resize function that falls back to returning original image
 */
export async function safeResizeImage(
    buffer: Buffer, 
    width: number, 
    height: number, 
    format: "jpeg" | "png" | "webp" = "jpeg",
): Promise<{ buffer: Buffer; resized: boolean; error?: string }> {
    try {
        const sharp = await loadSharp();
        
        if (!sharp) {
            return {
                buffer, // Return original
                resized: false,
                error: "Sharp module not available",
            };
        }

        const resizedBuffer = await sharp(buffer)
            .resize(width, height, { withoutEnlargement: true })
            .flatten({ background: { r: 255, g: 255, b: 255, alpha: 1 } })
            .toFormat(format)
            .toBuffer();

        return {
            buffer: resizedBuffer,
            resized: true,
        };
    } catch (error) {
        logger.error("[SharpWrapper] Image resize failed", {
            error: error instanceof Error ? error.message : String(error),
            width,
            height,
            format,
        });

        return {
            buffer, // Return original
            resized: false,
            error: error instanceof Error ? error.message : "Unknown error",
        };
    }
}

/**
 * Check if we can process images
 */
export function canProcessImages(): boolean {
    if (!sharpLoadAttempted) {
        // Don't block on checking - assume we can't until proven otherwise
        return false;
    }
    return sharpModule !== null;
}

/**
 * Get feature availability status
 */
export function getImageProcessingStatus(): {
    available: boolean;
    error?: string;
    features: {
        resize: boolean;
        heicConversion: boolean;
        formatConversion: boolean;
    };
} {
    const available = canProcessImages();
    
    return {
        available,
        error: sharpLoadError?.message,
        features: {
            resize: available,
            heicConversion: available, // Requires sharp
            formatConversion: available,
        },
    };
}
