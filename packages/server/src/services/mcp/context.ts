import { AsyncLocalStorage } from "async_hooks";
import type { Request, Response } from "express";

/**
 * Maintains per-request MCP HTTP context (req and res) using AsyncLocalStorage.
 */
type McpContext = { req: Request; res: Response };
const asyncLocalStorage = new AsyncLocalStorage<McpContext>();

/**
 * Runs the provided function within the MCP HTTP request context.
 * This ensures getCurrentMcpContext() returns the correct req/res for concurrent calls.
 */
export function runWithMcpContext<T>(req: Request, res: Response, fn: () => Promise<T>): Promise<T> {
    return asyncLocalStorage.run({ req, res }, fn);
}

/**
 * Retrieves the current MCP HTTP context (req and res).
 * Throws if called outside of runWithMcpContext.
 */
export function getCurrentMcpContext(): McpContext {
    const context = asyncLocalStorage.getStore();
    if (!context) {
        throw new Error(
            "No MCP HTTP context is set. Ensure runWithMcpContext is used when handling MCP requests.",
        );
    }
    return context;
} 
