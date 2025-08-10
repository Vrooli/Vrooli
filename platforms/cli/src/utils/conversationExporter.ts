import { writeFile } from "fs/promises";
import { extname } from "path";
import { type ChatMessage, type User } from "@vrooli/shared";
import { TIMEOUTS } from "./constants.js";

export type ExportFormat = "markdown" | "json" | "txt" | "html" | "csv";

export interface ExportOptions {
    format: ExportFormat;
    includeTimestamps?: boolean;
    includeUserIds?: boolean;
    includeMessageIds?: boolean;
    maxMessages?: number;
    title?: string;
}

export interface ConversationExportData {
    chatId: string;
    title?: string;
    participants: User[];
    messages: ChatMessage[];
    exportedAt: Date;
    totalMessages: number;
}

export class ConversationExporter {
    /**
     * Export conversation to file
     */
    async exportToFile(
        data: ConversationExportData,
        filePath: string,
        options: Partial<ExportOptions> = {},
    ): Promise<void> {
        const format = this.determineFormat(filePath, options.format);
        const exportOptions: ExportOptions = {
            format,
            includeTimestamps: true,
            includeUserIds: false,
            includeMessageIds: false,
            maxMessages: undefined,
            title: data.title,
            ...options,
        };

        let content: string;

        switch (format) {
            case "markdown":
                content = this.exportToMarkdown(data, exportOptions);
                break;
            case "json":
                content = this.exportToJson(data, exportOptions);
                break;
            case "txt":
                content = this.exportToText(data, exportOptions);
                break;
            case "html":
                content = this.exportToHtml(data, exportOptions);
                break;
            case "csv":
                content = this.exportToCsv(data, exportOptions);
                break;
            default:
                throw new Error(`Unsupported export format: ${format}`);
        }

        await writeFile(filePath, content, "utf8");
    }

    /**
     * Export conversation to markdown format
     */
    private exportToMarkdown(data: ConversationExportData, options: ExportOptions): string {
        const lines: string[] = [];
        
        // Header
        lines.push(`# ${options.title || "Chat Conversation"}`);
        lines.push("");
        lines.push(`**Chat ID:** ${data.chatId}`);
        lines.push(`**Exported:** ${data.exportedAt.toISOString()}`);
        lines.push(`**Total Messages:** ${data.totalMessages}`);
        
        // Participants
        if (data.participants.length > 0) {
            lines.push("");
            lines.push("## Participants");
            data.participants.forEach(participant => {
                const name = participant.name || participant.handle || "Unknown";
                const type = participant.isBot ? "🤖 Bot" : "👤 User";
                lines.push(`- ${type}: **${name}**${options.includeUserIds ? ` (${participant.id})` : ""}`);
            });
        }
        
        // Messages
        lines.push("");
        lines.push("## Conversation");
        lines.push("");
        
        const messages = this.getMessagesToExport(data.messages, options.maxMessages);
        
        messages.forEach(message => {
            const userName = message.user.name || message.user.handle || "Unknown";
            const timestamp = options.includeTimestamps ? 
                ` - ${new Date(message.createdAt).toLocaleString()}` : "";
            const messageId = options.includeMessageIds ? 
                ` (${message.id})` : "";
            
            const userType = message.user.isBot ? "🤖" : "👤";
            lines.push(`### ${userType} ${userName}${timestamp}${messageId}`);
            lines.push("");
            
            // Format message text with proper markdown escaping
            const formattedText = this.escapeMarkdown(message.text);
            lines.push(formattedText);
            lines.push("");
        });
        
        return lines.join("\n");
    }

    /**
     * Export conversation to JSON format
     */
    private exportToJson(data: ConversationExportData, options: ExportOptions): string {
        const messages = this.getMessagesToExport(data.messages, options.maxMessages);
        
        const exportData = {
            meta: {
                chatId: data.chatId,
                title: options.title,
                exportedAt: data.exportedAt.toISOString(),
                totalMessages: data.totalMessages,
                exportedMessages: messages.length,
                format: "json",
                version: "1.0",
            },
            participants: data.participants.map(p => ({
                id: p.id,
                name: p.name,
                handle: p.handle,
                isBot: p.isBot,
            })),
            messages: messages.map(message => ({
                id: options.includeMessageIds ? message.id : undefined,
                user: {
                    id: options.includeUserIds ? message.user.id : undefined,
                    name: message.user.name,
                    handle: message.user.handle,
                    isBot: message.user.isBot,
                },
                text: message.text,
                createdAt: options.includeTimestamps ? message.createdAt : undefined,
                language: message.language,
                sequence: message.sequence,
            })),
        };
        
        return JSON.stringify(exportData, null, 2);
    }

    /**
     * Export conversation to plain text format
     */
    private exportToText(data: ConversationExportData, options: ExportOptions): string {
        const lines: string[] = [];
        
        // Header
        lines.push(`Chat Conversation: ${options.title || data.chatId}`);
        lines.push(`Exported: ${data.exportedAt.toLocaleString()}`);
        lines.push(`Total Messages: ${data.totalMessages}`);
        lines.push("=".repeat(TIMEOUTS.MAX_WAIT_SECONDS));
        lines.push("");
        
        const messages = this.getMessagesToExport(data.messages, options.maxMessages);
        
        messages.forEach(message => {
            const userName = message.user.name || message.user.handle || "Unknown";
            const timestamp = options.includeTimestamps ? 
                ` (${new Date(message.createdAt).toLocaleString()})` : "";
            const userType = message.user.isBot ? "[Bot]" : "[User]";
            
            lines.push(`${userType} ${userName}${timestamp}:`);
            lines.push(message.text);
            lines.push("");
        });
        
        return lines.join("\n");
    }

    /**
     * Export conversation to HTML format
     */
    private exportToHtml(data: ConversationExportData, options: ExportOptions): string {
        const messages = this.getMessagesToExport(data.messages, options.maxMessages);
        
        const html = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>${this.escapeHtml(options.title || "Chat Conversation")}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; line-height: 1.6; }
        .header { border-bottom: 2px solid #ccc; padding-bottom: 10px; margin-bottom: 20px; }
        .message { margin-bottom: 15px; }
        .user-info { font-weight: bold; color: #333; }
        .bot { color: #0066cc; }
        .user { color: #008800; }
        .timestamp { font-size: 0.9em; color: #666; }
        .text { margin-top: 5px; white-space: pre-wrap; }
        pre { background: #f5f5f5; padding: 10px; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>${this.escapeHtml(options.title || "Chat Conversation")}</h1>
        <p><strong>Chat ID:</strong> ${this.escapeHtml(data.chatId)}</p>
        <p><strong>Exported:</strong> ${data.exportedAt.toLocaleString()}</p>
        <p><strong>Total Messages:</strong> ${data.totalMessages}</p>
    </div>
    
    <div class="conversation">
        ${messages.map(message => {
            const userName = this.escapeHtml(message.user.name || message.user.handle || "Unknown");
            const timestamp = options.includeTimestamps ? 
                `<span class="timestamp">${new Date(message.createdAt).toLocaleString()}</span>` : "";
            const userClass = message.user.isBot ? "bot" : "user";
            const userIcon = message.user.isBot ? "🤖" : "👤";
            
            return `
        <div class="message">
            <div class="user-info ${userClass}">
                ${userIcon} ${userName} ${timestamp}
            </div>
            <div class="text">${this.formatHtmlText(message.text)}</div>
        </div>`;
        }).join("")}
    </div>
</body>
</html>`;
        
        return html;
    }

    /**
     * Export conversation to CSV format
     */
    private exportToCsv(data: ConversationExportData, options: ExportOptions): string {
        const lines: string[] = [];
        
        // Header
        const headers = ["Timestamp", "User", "IsBot", "Message"];
        if (options.includeUserIds) headers.splice(2, 0, "UserId");
        if (options.includeMessageIds) headers.splice(-1, 0, "MessageId");
        
        lines.push(headers.map(h => this.escapeCsv(h)).join(","));
        
        // Messages
        const messages = this.getMessagesToExport(data.messages, options.maxMessages);
        
        messages.forEach(message => {
            const row = [
                options.includeTimestamps ? new Date(message.createdAt).toISOString() : "",
                message.user.name || message.user.handle || "Unknown",
                message.user.isBot ? "true" : "false",
            ];
            
            if (options.includeUserIds) {
                row.splice(2, 0, message.user.id);
            }
            
            if (options.includeMessageIds) {
                row.splice(-1, 0, message.id);
            }
            
            row.push(message.text);
            
            lines.push(row.map(cell => this.escapeCsv(String(cell))).join(","));
        });
        
        return lines.join("\n");
    }

    /**
     * Determine export format from file extension or explicit format
     */
    private determineFormat(filePath: string, explicitFormat?: ExportFormat): ExportFormat {
        if (explicitFormat) {
            return explicitFormat;
        }
        
        const ext = extname(filePath).toLowerCase();
        
        switch (ext) {
            case ".md":
            case ".markdown":
                return "markdown";
            case ".json":
                return "json";
            case ".txt":
                return "txt";
            case ".html":
            case ".htm":
                return "html";
            case ".csv":
                return "csv";
            default:
                return "txt"; // Default fallback
        }
    }

    /**
     * Get messages to export with optional limit
     */
    private getMessagesToExport(messages: ChatMessage[], maxMessages?: number): ChatMessage[] {
        if (!maxMessages || maxMessages >= messages.length) {
            return messages;
        }
        
        return messages.slice(-maxMessages); // Get last N messages
    }

    /**
     * Escape markdown special characters
     */
    private escapeMarkdown(text: string): string {
        return text
            .replace(/\\/g, "\\\\")
            .replace(/\*/g, "\\*")
            .replace(/_/g, "\\_")
            .replace(/`/g, "\\`")
            .replace(/\[/g, "\\[")
            .replace(/\]/g, "\\]")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;");
    }

    /**
     * Escape HTML special characters
     */
    private escapeHtml(text: string): string {
        return text
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/"/g, "&quot;")
            .replace(/'/g, "&#39;");
    }

    /**
     * Format text for HTML with basic formatting
     */
    private formatHtmlText(text: string): string {
        let formatted = this.escapeHtml(text);
        
        // Convert code blocks
        formatted = formatted.replace(/```(\w+)?\n([\s\S]*?)```/g, 
            "<pre><code>$2</code></pre>");
        
        // Convert inline code
        formatted = formatted.replace(/`([^`]+)`/g, "<code>$1</code>");
        
        return formatted;
    }

    /**
     * Escape CSV special characters
     */
    private escapeCsv(text: string): string {
        if (text.includes("\"") || text.includes(",") || text.includes("\n")) {
            return `"${text.replace(/"/g, "\"\"")}"`;
        }
        return text;
    }

    /**
     * Get available export formats
     */
    static getAvailableFormats(): ExportFormat[] {
        return ["markdown", "json", "txt", "html", "csv"];
    }

    /**
     * Validate export format
     */
    static isValidFormat(format: string): format is ExportFormat {
        return ConversationExporter.getAvailableFormats().includes(format as ExportFormat);
    }
}
