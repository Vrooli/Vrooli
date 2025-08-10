import { readFile, stat } from "fs/promises";
import { extname, basename } from "path";
import chalk from "chalk";
import { type TaskContextInfoInput } from "@vrooli/shared";
import { LIMITS } from "./constants.js";

export interface ContextFile {
    path: string;
    name: string;
    content: string;
    type: "text" | "code" | "json" | "markdown" | "binary";
    size: number;
    encoding?: string;
}

export interface ContextUrl {
    url: string;
    name: string;
    title?: string;
    content: string;
    type: "webpage" | "document" | "api";
    size: number;
}

export interface ContextRoutine {
    id: string;
    name: string;
    description?: string;
    type: "routine";
}

export type ContextItem = ContextFile | ContextUrl | ContextRoutine;

export class ContextManager {
    private contexts: Map<string, ContextItem> = new Map();
    private maxFileSize = LIMITS.MAX_CONTEXT_SIZE_BYTES; // 10MB limit
    private maxContexts = LIMITS.MAX_CONTEXT_MESSAGES; // Maximum number of context items

    /**
     * Add a file as context
     */
    async addFile(filePath: string, alias?: string): Promise<string> {
        try {
            // Check if file exists and get stats
            const stats = await stat(filePath);
            
            if (!stats.isFile()) {
                throw new Error(`Path is not a file: ${filePath}`);
            }

            if (stats.size > this.maxFileSize) {
                throw new Error(`File too large: ${this.formatFileSize(stats.size)} (max: ${this.formatFileSize(this.maxFileSize)})`);
            }

            // Determine file type
            const fileType = this.determineFileType(filePath);
            
            // Read file content
            const content = await this.readFileContent(filePath, fileType);
            
            // Create context item
            const contextId = alias || basename(filePath);
            const contextFile: ContextFile = {
                path: filePath,
                name: contextId,
                content,
                type: fileType,
                size: stats.size,
                encoding: fileType === "binary" ? "base64" : "utf8",
            };

            // Check context limit
            if (this.contexts.size >= this.maxContexts) {
                throw new Error(`Too many context items (max: ${this.maxContexts}). Remove some first.`);
            }

            this.contexts.set(contextId, contextFile);
            return contextId;

        } catch (error) {
            throw new Error(`Failed to add file context: ${error instanceof Error ? error.message : String(error)}`);
        }
    }

    /**
     * Add a URL as context
     */
    async addUrl(url: string, alias?: string): Promise<string> {
        try {
            // Basic URL validation
            const urlObj = new URL(url);
            if (!["http:", "https:"].includes(urlObj.protocol)) {
                throw new Error("Only HTTP and HTTPS URLs are supported");
            }

            // Fetch content (simplified - in production would use proper web scraping)
            const { content, title } = await this.fetchUrlContentWithTitle(url);
            
            const contextId = alias || urlObj.hostname;
            const contextUrl: ContextUrl = {
                url,
                name: contextId,
                title,
                content,
                type: "webpage",
                size: content.length,
            };

            // Check context limit
            if (this.contexts.size >= this.maxContexts) {
                throw new Error(`Too many context items (max: ${this.maxContexts}). Remove some first.`);
            }

            this.contexts.set(contextId, contextUrl);
            return contextId;

        } catch (error) {
            throw new Error(`Failed to add URL context: ${error instanceof Error ? error.message : String(error)}`);
        }
    }

    /**
     * Add a routine as context
     */
    addRoutine(routineId: string, name: string, description?: string): string {
        const contextId = name || routineId;
        const contextRoutine: ContextRoutine = {
            id: routineId,
            name: contextId,
            description,
            type: "routine",
        };

        // Check context limit
        if (this.contexts.size >= this.maxContexts) {
            throw new Error(`Too many context items (max: ${this.maxContexts}). Remove some first.`);
        }

        this.contexts.set(contextId, contextRoutine);
        return contextId;
    }

    /**
     * Remove a context item
     */
    removeContext(contextId: string): boolean {
        return this.contexts.delete(contextId);
    }

    /**
     * Clear all context items
     */
    clearAll(): void {
        this.contexts.clear();
    }

    /**
     * Get all context items
     */
    getAllContexts(): Map<string, ContextItem> {
        return new Map(this.contexts);
    }

    /**
     * Get context as TaskContextInfoInput array
     */
    getTaskContexts(): TaskContextInfoInput[] {
        const taskContexts: TaskContextInfoInput[] = [];

        for (const [id, context] of this.contexts) {
            let contextData: Record<string, unknown>;

            if ("path" in context) {
                // File context
                contextData = {
                    type: "file",
                    path: context.path,
                    content: context.content,
                    encoding: context.encoding,
                    size: context.size,
                };
            } else if ("url" in context) {
                // URL context
                contextData = {
                    type: "url",
                    url: context.url,
                    title: context.title,
                    content: context.content,
                    size: context.size,
                };
            } else {
                // Routine context
                contextData = {
                    type: "routine",
                    routineId: context.id,
                    description: context.description,
                };
            }

            taskContexts.push({
                id,
                label: context.name,
                name: context.name,
                description: "description" in context ? context.description : undefined,
                data: contextData,
            });
        }

        return taskContexts;
    }

    /**
     * Display context summary
     */
    displayContextSummary(): void {
        if (this.contexts.size === 0) {
            console.log(chalk.gray("No context items"));
            return;
        }

        console.log(chalk.bold(`Context Items (${this.contexts.size}/${this.maxContexts}):`));
        console.log();

        for (const [id, context] of this.contexts) {
            if ("path" in context) {
                // File context
                console.log(`📄 ${chalk.cyan(id)} - ${context.path}`);
                console.log(`   Type: ${context.type} | Size: ${this.formatFileSize(context.size)}`);
            } else if ("url" in context) {
                // URL context
                console.log(`🌐 ${chalk.cyan(id)} - ${context.url}`);
                console.log(`   Title: ${context.title || "Unknown"} | Size: ${this.formatFileSize(context.size)}`);
            } else {
                // Routine context
                console.log(`⚙️ ${chalk.cyan(id)} - Routine ${context.id}`);
                if (context.description) {
                    console.log(`   ${context.description}`);
                }
            }
            console.log();
        }
    }

    /**
     * Get context statistics
     */
    getStats(): {
        totalContexts: number;
        maxContexts: number;
        totalSize: number;
        byType: Record<string, number>;
    } {
        const stats = {
            totalContexts: this.contexts.size,
            maxContexts: this.maxContexts,
            totalSize: 0,
            byType: {} as Record<string, number>,
        };

        for (const context of this.contexts.values()) {
            // Count by type
            const type = "path" in context ? "file" : "url" in context ? "url" : "routine";
            stats.byType[type] = (stats.byType[type] || 0) + 1;

            // Add to total size
            if ("size" in context) {
                stats.totalSize += context.size;
            }
        }

        return stats;
    }

    /**
     * Determine file type from extension
     */
    private determineFileType(filePath: string): ContextFile["type"] {
        const ext = extname(filePath).toLowerCase();
        
        const codeExts = [".ts", ".js", ".tsx", ".jsx", ".py", ".java", ".cpp", ".c", ".h", ".cs", ".php", ".rb", ".go", ".rs", ".swift", ".kt"];
        const textExts = [".txt", ".log", ".csv", ".xml", ".html", ".css", ".sql", ".yaml", ".yml", ".toml", ".ini"];
        const jsonExts = [".json", ".jsonl"];
        const markdownExts = [".md", ".markdown", ".mdown", ".mkd"];

        if (codeExts.includes(ext)) return "code";
        if (jsonExts.includes(ext)) return "json";
        if (markdownExts.includes(ext)) return "markdown";
        if (textExts.includes(ext)) return "text";
        
        return "binary";
    }

    /**
     * Read file content with appropriate encoding
     */
    private async readFileContent(filePath: string, fileType: ContextFile["type"]): Promise<string> {
        if (fileType === "binary") {
            const buffer = await readFile(filePath);
            return buffer.toString("base64");
        } else {
            return await readFile(filePath, "utf8");
        }
    }

    /**
     * Fetch URL content (simplified implementation)
     */
    private async fetchUrlContent(url: string): Promise<string> {
        const { content } = await this.fetchUrlContentWithTitle(url);
        return content;
    }

    /**
     * Fetch URL content and extract title
     */
    private async fetchUrlContentWithTitle(url: string): Promise<{ content: string; title?: string }> {
        // This is a basic implementation - in production, you'd want proper web scraping
        // with handling for different content types, JavaScript rendering, etc.
        try {
            const response = await fetch(url, {
                headers: {
                    "User-Agent": "Vrooli CLI Bot",
                },
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const rawContent = await response.text();
            let title: string | undefined;
            let content: string;
            
            // Basic HTML stripping for web pages
            if (response.headers.get("content-type")?.includes("text/html")) {
                title = this.extractTitleFromContent(rawContent);
                content = this.stripHtmlTags(rawContent);
            } else {
                content = rawContent;
            }

            return { content, title };

        } catch (error) {
            throw new Error(`Failed to fetch URL: ${error instanceof Error ? error.message : String(error)}`);
        }
    }

    /**
     * Extract title from HTML content
     */
    private extractTitleFromContent(content: string): string | undefined {
        const titleMatch = content.match(/<title[^>]*>([^<]+)<\/title>/i);
        return titleMatch ? titleMatch[1].trim() : undefined;
    }

    /**
     * Basic HTML tag stripping
     */
    private stripHtmlTags(html: string): string {
        return html
            .replace(/<script[^>]*>.*?<\/script>/gis, "")
            .replace(/<style[^>]*>.*?<\/style>/gis, "")
            .replace(/<[^>]*>/g, "")
            .replace(/\s+/g, " ")
            .trim();
    }

    /**
     * Format file size for display
     */
    private formatFileSize(bytes: number): string {
        if (bytes === 0) return "0 B";
        
        const k = 1024;
        const sizes = ["B", "KB", "MB", "GB"];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        
        return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
    }
}
