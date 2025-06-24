/**
 * Tool Call Response Fixtures
 * 
 * Pre-defined tool calling patterns for testing AI tool integration.
 */

import type { AIMockConfig, AIMockToolCall } from "../types.js";

/**
 * Simple search tool call
 */
export const searchToolCall = (): AIMockConfig => ({
    content: "I'll search for that information for you.",
    toolCalls: [{
        name: "search",
        arguments: {
            query: "TypeScript best practices 2024",
            limit: 5
        },
        result: {
            results: [
                { title: "Official TypeScript Handbook", url: "https://www.typescriptlang.org/docs/" },
                { title: "TypeScript Best Practices Guide", url: "https://example.com/ts-guide" }
            ],
            totalFound: 42
        }
    }],
    confidence: 0.9
});

/**
 * Calculator tool call
 */
export const calculatorToolCall = (): AIMockConfig => ({
    content: "Let me calculate that for you.",
    toolCalls: [{
        name: "calculator",
        arguments: {
            operation: "multiply",
            operands: [15, 27]
        },
        result: 405
    }],
    confidence: 0.99
});

/**
 * Multiple sequential tool calls
 */
export const sequentialToolCalls = (): AIMockConfig => ({
    content: "I'll need to gather some information and then process it.",
    toolCalls: [
        {
            name: "fetch_data",
            arguments: { source: "database", table: "users" },
            result: { count: 1500, sample: [{ id: 1, name: "John" }] }
        },
        {
            name: "analyze_data",
            arguments: { data: "users_dataset", metric: "engagement" },
            result: { average: 0.75, trend: "increasing" }
        },
        {
            name: "generate_report",
            arguments: { analysis: "engagement_analysis", format: "summary" },
            result: { report: "User engagement is at 75% and trending upward." }
        }
    ],
    confidence: 0.88
});

/**
 * Parallel tool calls
 */
export const parallelToolCalls = (): AIMockConfig => ({
    content: "I'll gather information from multiple sources simultaneously.",
    toolCalls: [
        {
            name: "weather_api",
            arguments: { location: "New York", units: "metric" },
            result: { temp: 22, condition: "sunny" }
        },
        {
            name: "news_api",
            arguments: { category: "technology", limit: 3 },
            result: { articles: ["AI breakthrough", "New smartphone launch", "Quantum computing advance"] }
        },
        {
            name: "stock_api",
            arguments: { symbols: ["AAPL", "GOOGL"], metrics: ["price", "change"] },
            result: { AAPL: { price: 180.50, change: "+1.2%" }, GOOGL: { price: 142.30, change: "-0.5%" } }
        }
    ],
    metadata: {
        executionMode: "parallel",
        executionTime: 250
    },
    confidence: 0.92
});

/**
 * Tool call with error handling
 */
export const toolCallWithError = (): AIMockConfig => ({
    content: "I encountered an issue with one of the tools, but I've found an alternative solution.",
    toolCalls: [
        {
            name: "primary_service",
            arguments: { action: "fetch" },
            error: "Service temporarily unavailable"
        },
        {
            name: "fallback_service",
            arguments: { action: "fetch", source: "cache" },
            result: { data: "cached_result", age: "5 minutes" }
        }
    ],
    confidence: 0.8
});

/**
 * Complex data processing tool chain
 */
export const dataProcessingChain = (): AIMockConfig => ({
    content: "I'll process this data through multiple transformation steps.",
    toolCalls: [
        {
            name: "data_validator",
            arguments: { 
                schema: "user_input",
                data: { name: "test", email: "test@example.com" }
            },
            result: { valid: true, errors: [] }
        },
        {
            name: "data_transformer",
            arguments: {
                transformations: ["normalize", "enrich"],
                input: { name: "test", email: "test@example.com" }
            },
            result: {
                name: "Test",
                email: "test@example.com",
                domain: "example.com",
                created_at: "2024-01-01T00:00:00Z"
            }
        },
        {
            name: "data_store",
            arguments: {
                collection: "users",
                operation: "upsert",
                data: { /* transformed data */ }
            },
            result: { success: true, id: "user_123" }
        }
    ],
    confidence: 0.91
});

/**
 * Conditional tool execution
 */
export const conditionalToolExecution = (): AIMockConfig => ({
    content: "Based on the initial check, I'll proceed with the appropriate action.",
    toolCalls: [
        {
            name: "check_condition",
            arguments: { condition: "user_premium_status" },
            result: { isPremium: true }
        },
        {
            name: "premium_feature",
            arguments: { feature: "advanced_analysis" },
            result: { 
                analysis: "Deep insights available",
                recommendations: ["Option A", "Option B", "Option C"]
            }
        }
    ],
    reasoning: "User has premium status, so I'm using the advanced analysis feature.",
    confidence: 0.93
});

/**
 * File operation tools
 */
export const fileOperationTools = (): AIMockConfig => ({
    content: "I'll help you manage these files.",
    toolCalls: [
        {
            name: "file_read",
            arguments: { path: "/data/config.json" },
            result: { content: '{"version": "1.0", "settings": {}}' }
        },
        {
            name: "file_transform",
            arguments: { 
                operation: "update_json",
                updates: { "settings.theme": "dark" }
            },
            result: { success: true }
        },
        {
            name: "file_write",
            arguments: { 
                path: "/data/config.json",
                content: '{"version": "1.0", "settings": {"theme": "dark"}}'
            },
            result: { bytesWritten: 48 }
        }
    ],
    confidence: 0.95
});

/**
 * API integration tools
 */
export const apiIntegrationTools = (): AIMockConfig => ({
    content: "I'll integrate with the external API to fetch the required data.",
    toolCalls: [
        {
            name: "api_auth",
            arguments: { service: "external_api", method: "oauth2" },
            result: { token: "bearer_xyz123", expires_in: 3600 }
        },
        {
            name: "api_request",
            arguments: {
                endpoint: "/v1/data",
                method: "GET",
                headers: { Authorization: "Bearer xyz123" }
            },
            result: {
                status: 200,
                data: { items: [], total: 0 }
            }
        }
    ],
    confidence: 0.89
});

/**
 * Monitoring and alerting tools
 */
export const monitoringTools = (): AIMockConfig => ({
    content: "I'm checking the system status and will alert if necessary.",
    toolCalls: [
        {
            name: "metrics_query",
            arguments: {
                metric: "cpu_usage",
                timeRange: "last_5_minutes",
                aggregation: "average"
            },
            result: { value: 85, unit: "percent" }
        },
        {
            name: "threshold_check",
            arguments: {
                metric: "cpu_usage",
                value: 85,
                threshold: 80,
                condition: "greater_than"
            },
            result: { triggered: true, severity: "warning" }
        },
        {
            name: "send_alert",
            arguments: {
                channel: "ops_team",
                severity: "warning",
                message: "CPU usage at 85% (threshold: 80%)"
            },
            result: { sent: true, messageId: "alert_456" }
        }
    ],
    confidence: 0.96
});

/**
 * Tool discovery and selection
 */
export const toolDiscovery = (): AIMockConfig => ({
    content: "Let me find the right tools for your task.",
    toolCalls: [
        {
            name: "list_available_tools",
            arguments: { category: "data_processing" },
            result: {
                tools: ["csv_parser", "json_transformer", "data_validator"],
                descriptions: {
                    csv_parser: "Parse CSV files",
                    json_transformer: "Transform JSON data",
                    data_validator: "Validate data against schemas"
                }
            }
        },
        {
            name: "select_optimal_tool",
            arguments: {
                task: "process CSV file",
                available: ["csv_parser", "json_transformer", "data_validator"]
            },
            result: {
                selected: "csv_parser",
                reason: "Best match for CSV processing"
            }
        }
    ],
    confidence: 0.87
});

/**
 * No tools needed response
 */
export const noToolsNeeded = (): AIMockConfig => ({
    content: "I can answer this directly without using any external tools. The answer to your question is 42.",
    toolCalls: [],
    confidence: 0.98,
    reasoning: "The question can be answered from my knowledge base without external tool assistance."
});