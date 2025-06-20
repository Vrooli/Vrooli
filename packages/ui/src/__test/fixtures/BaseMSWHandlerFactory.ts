/* c8 ignore start */
/**
 * Base implementation for MSW handler factories
 * 
 * This class provides utilities for creating Mock Service Worker handlers
 * that simulate API responses for component testing.
 */

import { rest, type RestHandler } from "msw";
import type {
    MSWHandlerFactory,
    HandlerConfig,
    DelayConfig,
    ErrorScenario,
    AppError
} from "./types.js";

/**
 * Configuration for creating an MSW handler factory
 */
export interface MSWHandlerFactoryConfig<TCreateInput, TUpdateInput, TFindResult> {
    // Base URL for API endpoints
    baseUrl: string;
    
    // Endpoint paths
    endpoints: {
        create: string;
        update: string;
        delete: string;
        find: string;
        list?: string;
    };
    
    // Success response data
    successResponses: {
        create: (input: TCreateInput) => TFindResult;
        update: (input: TUpdateInput) => TFindResult;
        find: (id: string) => TFindResult;
        list?: () => TFindResult[];
    };
    
    // Validation function (optional)
    validate?: {
        create?: (input: TCreateInput) => { isValid: boolean; errors?: string[] };
        update?: (input: TUpdateInput) => { isValid: boolean; errors?: string[] };
    };
    
    // Default delay configuration
    defaultDelay?: number;
}

/**
 * Base MSW handler factory implementation
 * 
 * Provides comprehensive API mocking capabilities for UI testing with
 * support for success scenarios, error handling, and network simulation.
 * 
 * @template TCreateInput - The API create input type
 * @template TUpdateInput - The API update input type
 * @template TFindResult - The API response type
 */
export class BaseMSWHandlerFactory<
    TCreateInput = unknown,
    TUpdateInput = unknown,
    TFindResult = unknown
> implements MSWHandlerFactory {
    
    protected config: MSWHandlerFactoryConfig<TCreateInput, TUpdateInput, TFindResult>;
    
    constructor(config: MSWHandlerFactoryConfig<TCreateInput, TUpdateInput, TFindResult>) {
        this.config = config;
    }
    
    /**
     * Create handlers for successful API operations
     */
    createSuccessHandlers(): RestHandler[] {
        const handlers: RestHandler[] = [];
        const { baseUrl, endpoints, successResponses, defaultDelay = 0 } = this.config;
        
        // Create handler
        handlers.push(
            rest.post(`${baseUrl}${endpoints.create}`, async (req, res, ctx) => {
                const body = await req.json() as TCreateInput;
                
                // Validate if configured
                if (this.config.validate?.create) {
                    const validation = this.config.validate.create(body);
                    if (!validation.isValid) {
                        return res(
                            ctx.status(400),
                            ctx.json({
                                success: false,
                                errors: validation.errors
                            })
                        );
                    }
                }
                
                // Return success response
                const response = successResponses.create(body);
                return res(
                    ctx.delay(defaultDelay),
                    ctx.status(201),
                    ctx.json({
                        success: true,
                        data: response
                    })
                );
            })
        );
        
        // Update handler
        handlers.push(
            rest.put(`${baseUrl}${endpoints.update}`, async (req, res, ctx) => {
                const body = await req.json() as TUpdateInput;
                
                // Validate if configured
                if (this.config.validate?.update) {
                    const validation = this.config.validate.update(body);
                    if (!validation.isValid) {
                        return res(
                            ctx.status(400),
                            ctx.json({
                                success: false,
                                errors: validation.errors
                            })
                        );
                    }
                }
                
                // Return success response
                const response = successResponses.update(body);
                return res(
                    ctx.delay(defaultDelay),
                    ctx.json({
                        success: true,
                        data: response
                    })
                );
            })
        );
        
        // Find handler
        handlers.push(
            rest.get(`${baseUrl}${endpoints.find}/:id`, (req, res, ctx) => {
                const { id } = req.params as { id: string };
                
                try {
                    const response = successResponses.find(id);
                    return res(
                        ctx.delay(defaultDelay),
                        ctx.json({
                            success: true,
                            data: response
                        })
                    );
                } catch (error) {
                    return res(
                        ctx.status(404),
                        ctx.json({
                            success: false,
                            error: "Not found"
                        })
                    );
                }
            })
        );
        
        // Delete handler
        handlers.push(
            rest.delete(`${baseUrl}${endpoints.delete}/:id`, (req, res, ctx) => {
                return res(
                    ctx.delay(defaultDelay),
                    ctx.json({
                        success: true,
                        message: "Deleted successfully"
                    })
                );
            })
        );
        
        // List handler (if configured)
        if (endpoints.list && successResponses.list) {
            handlers.push(
                rest.get(`${baseUrl}${endpoints.list}`, (req, res, ctx) => {
                    const response = successResponses.list!();
                    return res(
                        ctx.delay(defaultDelay),
                        ctx.json({
                            success: true,
                            data: response,
                            pageInfo: {
                                hasMore: false,
                                totalCount: response.length
                            }
                        })
                    );
                })
            );
        }
        
        return handlers;
    }
    
    /**
     * Create handlers for error scenarios
     */
    createErrorHandlers(errorScenarios: ErrorScenario[]): RestHandler[] {
        const handlers: RestHandler[] = [];
        const { baseUrl, endpoints } = this.config;
        
        for (const scenario of errorScenarios) {
            switch (scenario.type) {
                case "validation":
                    handlers.push(
                        rest.post(`${baseUrl}${endpoints.create}`, (req, res, ctx) => {
                            return res(
                                ctx.status(400),
                                ctx.json({
                                    success: false,
                                    code: "VALIDATION_ERROR",
                                    message: "Validation failed",
                                    errors: scenario.config.errors || ["Invalid input"]
                                })
                            );
                        })
                    );
                    break;
                    
                case "permission":
                    handlers.push(
                        rest.all(`${baseUrl}/*`, (req, res, ctx) => {
                            return res(
                                ctx.status(403),
                                ctx.json({
                                    success: false,
                                    code: "PERMISSION_DENIED",
                                    message: scenario.config.message || "You don't have permission to perform this action"
                                })
                            );
                        })
                    );
                    break;
                    
                case "server":
                    handlers.push(
                        rest.all(`${baseUrl}/*`, (req, res, ctx) => {
                            return res(
                                ctx.status(500),
                                ctx.json({
                                    success: false,
                                    code: "INTERNAL_SERVER_ERROR",
                                    message: scenario.config.message || "An unexpected error occurred"
                                })
                            );
                        })
                    );
                    break;
                    
                case "timeout":
                    handlers.push(
                        rest.all(`${baseUrl}/*`, (req, res, ctx) => {
                            // Delay longer than typical timeout
                            return res(ctx.delay(scenario.config.delay || 35000));
                        })
                    );
                    break;
                    
                case "network":
                    handlers.push(
                        rest.all(`${baseUrl}/*`, (req, res, ctx) => {
                            return res.networkError("Failed to connect");
                        })
                    );
                    break;
            }
        }
        
        return handlers;
    }
    
    /**
     * Create handlers with configurable delays
     */
    createDelayHandlers(delays: DelayConfig): RestHandler[] {
        const successHandlers = this.createSuccessHandlers();
        
        // Wrap each handler with custom delay
        return successHandlers.map(handler => {
            return rest.all(handler.info.path, async (req, res, ctx) => {
                // Calculate delay with optional jitter
                let delay = delays.min;
                if (delays.jitter && delays.max > delays.min) {
                    delay = delays.min + Math.random() * (delays.max - delays.min);
                }
                
                // Get original response
                const originalResponse = await handler.run(req, res, ctx);
                
                // Add delay to response
                return res(
                    ctx.delay(delay),
                    ...originalResponse.headers,
                    ...originalResponse.cookies,
                    ctx.body(originalResponse.body)
                );
            });
        });
    }
    
    /**
     * Create handlers that simulate network errors
     */
    createNetworkErrorHandlers(): RestHandler[] {
        const { baseUrl } = this.config;
        
        return [
            // Connection refused
            rest.all(`${baseUrl}/*`, (req, res, ctx) => {
                return res.networkError("Connection refused");
            })
        ];
    }
    
    /**
     * Create a custom handler with specific configuration
     */
    createCustomHandler(config: HandlerConfig): RestHandler {
        const { method, path, status = 200, delay = 0, response, error } = config;
        const fullPath = `${this.config.baseUrl}${path}`;
        
        const handler = rest[method.toLowerCase() as keyof typeof rest];
        
        return handler(fullPath, (req, res, ctx) => {
            // Simulate delay
            const delayCtx = delay > 0 ? ctx.delay(delay) : null;
            
            // Return error if configured
            if (error) {
                return res(
                    ...(delayCtx ? [delayCtx] : []),
                    ctx.status(status),
                    ctx.json({
                        success: false,
                        code: error.code,
                        message: error.message,
                        details: error.details
                    })
                );
            }
            
            // Return success response
            return res(
                ...(delayCtx ? [delayCtx] : []),
                ctx.status(status),
                ctx.json({
                    success: true,
                    data: response
                })
            );
        });
    }
    
    /**
     * Create paginated list handler
     */
    createPaginatedHandler(
        path: string,
        items: TFindResult[],
        pageSize: number = 10
    ): RestHandler {
        const fullPath = `${this.config.baseUrl}${path}`;
        
        return rest.get(fullPath, (req, res, ctx) => {
            // Get pagination params
            const params = req.url.searchParams;
            const page = parseInt(params.get("page") || "1");
            const take = parseInt(params.get("take") || String(pageSize));
            
            // Calculate pagination
            const start = (page - 1) * take;
            const end = start + take;
            const pageItems = items.slice(start, end);
            const totalPages = Math.ceil(items.length / take);
            
            return res(
                ctx.json({
                    success: true,
                    data: pageItems,
                    pageInfo: {
                        page,
                        pageSize: take,
                        totalPages,
                        totalCount: items.length,
                        hasNext: page < totalPages,
                        hasPrevious: page > 1
                    }
                })
            );
        });
    }
    
    /**
     * Create handlers for testing loading states
     */
    createLoadingStateHandlers(loadingDuration: number = 2000): RestHandler[] {
        const handlers = this.createSuccessHandlers();
        
        // Add delay to all handlers
        return handlers.map(handler => {
            return rest.all(handler.info.path, async (req, res, ctx) => {
                const originalResponse = await handler.run(req, res, ctx);
                
                return res(
                    ctx.delay(loadingDuration),
                    ...originalResponse.headers,
                    ...originalResponse.cookies,
                    ctx.body(originalResponse.body)
                );
            });
        });
    }
}
/* c8 ignore stop */