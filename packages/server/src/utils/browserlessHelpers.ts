/**
 * Browserless Helper Utilities
 * 
 * Simplified utilities for common Browserless operations.
 * These helpers provide easy-to-use functions for screenshots, PDFs, and web scraping.
 */

import { ResourceRegistry } from "../services/resources/ResourceRegistry.js";
import type { BrowserlessResource } from "../services/resources/providers/BrowserlessResource.js";
import { logger } from "../events/logger.js";
import { promises as fs } from "fs";
import path from "path";

export interface ScreenshotOptions {
    fullPage?: boolean;
    width?: number;
    height?: number;
    type?: "png" | "jpeg";
    quality?: number; // 0-100 for jpeg
    waitFor?: number; // Additional wait time in ms
}

export interface PdfOptions {
    format?: "A4" | "Letter" | "Legal" | "Tabloid";
    printBackground?: boolean;
    margin?: {
        top?: string;
        right?: string;
        bottom?: string;
        left?: string;
    };
}

export interface ScrapeOptions {
    waitUntil?: "load" | "domcontentloaded" | "networkidle0" | "networkidle2";
    waitFor?: number;
}

/**
 * Get Browserless resource instance
 */
async function getBrowserlessResource(): Promise<BrowserlessResource | null> {
    const registry = ResourceRegistry.getInstance();
    const resource = await registry.getResource("browserless");
    
    if (!resource) {
        logger.warn("Browserless resource is not available");
        return null;
    }
    
    // Check if resource is healthy
    const healthResult = await resource.healthCheck();
    if (!healthResult.healthy) {
        logger.warn("Browserless resource is not healthy", { result: healthResult });
        return null;
    }
    
    return resource as BrowserlessResource;
}

/**
 * Determine if a URL should use Browserless based on patterns
 */
export function shouldUseBrowserless(url: string, context?: {
    needsVisualReasoning?: boolean;
    hasAntiBot?: boolean;
    isDesktopApp?: boolean;
}): boolean {
    // Desktop apps always use Agent-S2
    if (context?.isDesktopApp) {
        return false;
    }

    // Visual reasoning or anti-bot requires Agent-S2
    if (context?.needsVisualReasoning || context?.hasAntiBot) {
        return false;
    }

    // Check if URL is local/internal
    const localPatterns = [
        /^https?:\/\/localhost/i,
        /^https?:\/\/127\.0\.0\.1/i,
        /^https?:\/\/0\.0\.0\.0/i,
        /^https?:\/\/192\.168\./i,
        /^https?:\/\/10\./i,
        /^https?:\/\/172\.(1[6-9]|2[0-9]|3[0-1])\./i,
        /:\d{4,5}/, // Development ports
    ];

    return localPatterns.some(pattern => pattern.test(url));
}

/**
 * Take a screenshot of a webpage
 */
export async function takeScreenshot(
    url: string,
    options?: ScreenshotOptions,
): Promise<Buffer | null> {
    try {
        const resource = await getBrowserlessResource();
        if (!resource) {
            throw new Error("Browserless is not available");
        }

        const body = {
            url,
            options: {
                fullPage: options?.fullPage ?? false,
                type: options?.type ?? "png",
                quality: options?.quality ?? 80,
                ...(options?.width && options?.height && {
                    viewport: {
                        width: options.width,
                        height: options.height,
                    },
                }),
            },
            gotoOptions: {
                waitUntil: "networkidle2",
            },
            ...(options?.waitFor && { waitFor: options.waitFor }),
        };

        const baseUrl = resource.getBaseUrl();
        const token = resource.getToken();
        
        if (!baseUrl) {
            throw new Error("Browserless baseUrl not configured");
        }

        const response = await fetch(`${baseUrl}/chrome/screenshot`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                ...(token && {
                    "Authorization": `Bearer ${token}`,
                }),
            },
            body: JSON.stringify(body),
        });

        if (!response.ok) {
            const error = await response.text();
            throw new Error(`Screenshot failed: ${error}`);
        }

        const buffer = Buffer.from(await response.arrayBuffer());
        logger.info(`Screenshot captured for ${url} (${buffer.length} bytes)`);
        
        return buffer;
    } catch (error) {
        logger.error(`Failed to take screenshot of ${url}:`, error);
        return null;
    }
}

/**
 * Take a screenshot and save to file
 */
export async function screenshotToFile(
    url: string,
    filepath: string,
    options?: ScreenshotOptions,
): Promise<string | null> {
    try {
        const screenshot = await takeScreenshot(url, options);
        if (!screenshot) {
            return null;
        }

        const absolutePath = path.resolve(filepath);
        await fs.mkdir(path.dirname(absolutePath), { recursive: true });
        await fs.writeFile(absolutePath, new Uint8Array(screenshot));
        
        logger.info(`Screenshot saved to ${absolutePath}`);
        return absolutePath;
    } catch (error) {
        logger.error(`Failed to save screenshot to ${filepath}:`, error);
        return null;
    }
}

/**
 * Generate a PDF from a webpage
 */
export async function generatePdf(
    url: string,
    options?: PdfOptions,
): Promise<Buffer | null> {
    try {
        const resource = await getBrowserlessResource();
        if (!resource) {
            throw new Error("Browserless is not available");
        }

        const body = {
            url,
            options: {
                format: options?.format ?? "A4",
                printBackground: options?.printBackground ?? true,
                margin: options?.margin ?? {
                    top: "20px",
                    right: "20px",
                    bottom: "20px",
                    left: "20px",
                },
            },
            gotoOptions: {
                waitUntil: "networkidle2",
            },
        };

        const baseUrl = resource.getBaseUrl();
        const token = resource.getToken();
        
        if (!baseUrl) {
            throw new Error("Browserless baseUrl not configured");
        }

        const response = await fetch(`${baseUrl}/chrome/pdf`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                ...(token && {
                    "Authorization": `Bearer ${token}`,
                }),
            },
            body: JSON.stringify(body),
        });

        if (!response.ok) {
            const error = await response.text();
            throw new Error(`PDF generation failed: ${error}`);
        }

        const buffer = Buffer.from(await response.arrayBuffer());
        logger.info(`PDF generated for ${url} (${buffer.length} bytes)`);
        
        return buffer;
    } catch (error) {
        logger.error(`Failed to generate PDF for ${url}:`, error);
        return null;
    }
}

/**
 * Generate a PDF and save to file
 */
export async function pdfToFile(
    url: string,
    filepath: string,
    options?: PdfOptions,
): Promise<string | null> {
    try {
        const pdf = await generatePdf(url, options);
        if (!pdf) {
            return null;
        }

        const absolutePath = path.resolve(filepath);
        await fs.mkdir(path.dirname(absolutePath), { recursive: true });
        await fs.writeFile(absolutePath, new Uint8Array(pdf));
        
        logger.info(`PDF saved to ${absolutePath}`);
        return absolutePath;
    } catch (error) {
        logger.error(`Failed to save PDF to ${filepath}:`, error);
        return null;
    }
}

/**
 * Extract HTML content from a webpage
 */
export async function getPageContent(
    url: string,
    options?: ScrapeOptions,
): Promise<string | null> {
    try {
        const resource = await getBrowserlessResource();
        if (!resource) {
            throw new Error("Browserless is not available");
        }

        const body = {
            url,
            gotoOptions: {
                waitUntil: options?.waitUntil ?? "networkidle2",
            },
            ...(options?.waitFor && { waitFor: options.waitFor }),
        };

        const baseUrl = resource.getBaseUrl();
        const token = resource.getToken();
        
        if (!baseUrl) {
            throw new Error("Browserless baseUrl not configured");
        }

        const response = await fetch(`${baseUrl}/chrome/content`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                ...(token && {
                    "Authorization": `Bearer ${token}`,
                }),
            },
            body: JSON.stringify(body),
        });

        if (!response.ok) {
            const error = await response.text();
            throw new Error(`Content extraction failed: ${error}`);
        }

        const content = await response.text();
        logger.info(`Content extracted from ${url} (${content.length} chars)`);
        
        return content;
    } catch (error) {
        logger.error(`Failed to extract content from ${url}:`, error);
        return null;
    }
}

/**
 * Scrape specific elements from a webpage
 */
export async function scrapeElements(
    url: string,
    selectors: Array<{ selector: string; property?: string }>,
    options?: ScrapeOptions,
): Promise<any[] | null> {
    try {
        const resource = await getBrowserlessResource();
        if (!resource) {
            throw new Error("Browserless is not available");
        }

        const body = {
            url,
            elements: selectors.map(s => ({
                selector: s.selector,
                property: s.property ?? "innerText",
            })),
            gotoOptions: {
                waitUntil: options?.waitUntil ?? "networkidle2",
            },
            ...(options?.waitFor && { waitFor: options.waitFor }),
        };

        const baseUrl = resource.getBaseUrl();
        const token = resource.getToken();
        
        if (!baseUrl) {
            throw new Error("Browserless baseUrl not configured");
        }

        const response = await fetch(`${baseUrl}/chrome/scrape`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                ...(token && {
                    "Authorization": `Bearer ${token}`,
                }),
            },
            body: JSON.stringify(body),
        });

        if (!response.ok) {
            const error = await response.text();
            throw new Error(`Scraping failed: ${error}`);
        }

        const results = await response.json();
        logger.info(`Scraped ${results.length} elements from ${url}`);
        
        return results;
    } catch (error) {
        logger.error(`Failed to scrape elements from ${url}:`, error);
        return null;
    }
}

/**
 * Capture a dashboard screenshot (convenience method)
 */
export async function captureDashboard(
    url: string,
    outputPath: string,
    options?: {
        width?: number;
        height?: number;
        waitFor?: number;
    },
): Promise<string | null> {
    return screenshotToFile(url, outputPath, {
        fullPage: true,
        width: options?.width ?? 1920,
        height: options?.height ?? 1080,
        waitFor: options?.waitFor ?? 2000, // Wait 2s for dashboards to load
    });
}

/**
 * Monitor service health by checking status page
 */
export async function checkServiceHealth(
    url: string,
    statusSelector = ".status",
    options?: {
        screenshot?: boolean;
        outputPath?: string;
    },
): Promise<{
    status?: string;
    screenshot?: string;
    healthy: boolean;
} | null> {
    try {
        // Try to scrape status
        const elements = await scrapeElements(url, [
            { selector: statusSelector, property: "innerText" },
        ]);

        const status = elements?.[0]?.innerText;
        let screenshotPath: string | undefined;

        // Optionally take screenshot
        if (options?.screenshot && options?.outputPath) {
            screenshotPath = await screenshotToFile(url, options.outputPath) ?? undefined;
        }

        return {
            status,
            screenshot: screenshotPath,
            healthy: status?.toLowerCase().includes("healthy") ?? false,
        };
    } catch (error) {
        logger.error(`Failed to check service health for ${url}:`, error);
        return null;
    }
}
