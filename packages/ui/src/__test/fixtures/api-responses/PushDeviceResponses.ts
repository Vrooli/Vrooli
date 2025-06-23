/**
 * PushDevice API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for push device endpoints.
 * It includes success responses, error responses, and MSW handlers for testing
 * push notification device registration and management.
 */

import { rest, type RestHandler } from "msw";
import type { 
    PushDevice, 
    PushDeviceCreateInput, 
    PushDeviceUpdateInput,
    PushDeviceTestInput,
    User,
} from "@vrooli/shared";
import { 
    pushDeviceValidation, 
} from "@vrooli/shared";

/**
 * Standard API response wrapper
 */
export interface APIResponse<T> {
    data: T;
    meta: {
        timestamp: string;
        requestId: string;
        version: string;
        links?: {
            self?: string;
            related?: Record<string, string>;
        };
    };
}

/**
 * API error response structure
 */
export interface APIErrorResponse {
    error: {
        code: string;
        message: string;
        details?: Record<string, string | number | boolean | object>;
        timestamp: string;
        requestId: string;
        path: string;
    };
}

/**
 * Paginated response structure
 */
export interface PaginatedAPIResponse<T> extends APIResponse<T[]> {
    pagination: {
        page: number;
        pageSize: number;
        totalCount: number;
        totalPages: number;
        hasNextPage: boolean;
        hasPreviousPage: boolean;
    };
}

/**
 * PushDevice API response factory
 */
export class PushDeviceResponseFactory {
    private readonly baseUrl: string;
    
    constructor(baseUrl: string = process.env.VITE_SERVER_URL || "http://localhost:5329") {
        this.baseUrl = baseUrl;
    }
    
    /**
     * Generate unique request ID
     */
    private generateRequestId(): string {
        return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Generate unique resource ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create successful push device response
     */
    createSuccessResponse(pushDevice: PushDevice): APIResponse<PushDevice> {
        return {
            data: pushDevice,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/push-device/${pushDevice.id}`,
                    related: {
                        test: `${this.baseUrl}/api/push-device/${pushDevice.id}/test`,
                        user: `${this.baseUrl}/api/user/me`,
                    },
                },
            },
        };
    }
    
    /**
     * Create push device list response
     */
    createPushDeviceListResponse(pushDevices: PushDevice[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<PushDevice> {
        const paginationData = pagination || {
            page: 1,
            pageSize: pushDevices.length,
            totalCount: pushDevices.length,
        };
        
        return {
            data: pushDevices,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/push-device?page=${paginationData.page}&limit=${paginationData.pageSize}`,
                },
            },
            pagination: {
                ...paginationData,
                totalPages: Math.ceil(paginationData.totalCount / paginationData.pageSize),
                hasNextPage: paginationData.page * paginationData.pageSize < paginationData.totalCount,
                hasPreviousPage: paginationData.page > 1,
            },
        };
    }
    
    /**
     * Create validation error response
     */
    createValidationErrorResponse(fieldErrors: Record<string, string>): APIErrorResponse {
        return {
            error: {
                code: "VALIDATION_ERROR",
                message: "The request contains invalid data",
                details: {
                    fieldErrors,
                    invalidFields: Object.keys(fieldErrors),
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/push-device",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(deviceId: string): APIErrorResponse {
        return {
            error: {
                code: "PUSH_DEVICE_NOT_FOUND",
                message: `Push device with ID '${deviceId}' was not found`,
                details: {
                    deviceId,
                    searchCriteria: { id: deviceId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/push-device/${deviceId}`,
            },
        };
    }
    
    /**
     * Create permission error response
     */
    createPermissionErrorResponse(operation: string): APIErrorResponse {
        return {
            error: {
                code: "PERMISSION_DENIED",
                message: `You do not have permission to ${operation} this push device`,
                details: {
                    operation,
                    requiredPermissions: ["push_device:write"],
                    userPermissions: ["push_device:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/push-device",
            },
        };
    }
    
    /**
     * Create duplicate device error response
     */
    createDuplicateDeviceErrorResponse(endpoint: string): APIErrorResponse {
        return {
            error: {
                code: "DUPLICATE_DEVICE",
                message: "A push device with this endpoint already exists",
                details: {
                    endpoint,
                    suggestion: "Use the existing device or delete it first",
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/push-device",
            },
        };
    }
    
    /**
     * Create invalid subscription error response
     */
    createInvalidSubscriptionErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: "INVALID_SUBSCRIPTION",
                message: "The push subscription is invalid or has expired",
                details: {
                    reason: "Invalid endpoint or keys",
                    suggestion: "Request new push notification permissions",
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/push-device",
            },
        };
    }
    
    /**
     * Create push test success response
     */
    createPushTestSuccessResponse(): APIResponse<{ success: boolean; message: string }> {
        return {
            data: {
                success: true,
                message: "Test notification sent successfully",
            },
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
            },
        };
    }
    
    /**
     * Create push test failure response
     */
    createPushTestFailureResponse(reason: string): APIErrorResponse {
        return {
            error: {
                code: "PUSH_TEST_FAILED",
                message: "Failed to send test notification",
                details: {
                    reason,
                    troubleshooting: [
                        "Check if the device is online",
                        "Verify notification permissions are granted",
                        "Ensure the subscription is still valid",
                    ],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/push-device/test",
            },
        };
    }
    
    /**
     * Create network error response
     */
    createNetworkErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: "NETWORK_ERROR",
                message: "Network request failed",
                details: {
                    reason: "Connection timeout",
                    retryable: true,
                    retryAfter: 5000,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/push-device",
            },
        };
    }
    
    /**
     * Create server error response
     */
    createServerErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: "INTERNAL_SERVER_ERROR",
                message: "An unexpected server error occurred",
                details: {
                    errorId: `ERR_${Date.now()}`,
                    reportable: true,
                    retryable: true,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/push-device",
            },
        };
    }
    
    /**
     * Create mock push device data
     */
    createMockPushDevice(overrides?: Partial<PushDevice>): PushDevice {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultPushDevice: PushDevice = {
            __typename: "PushDevice",
            id,
            createdAt: now,
            updatedAt: now,
            deviceId: `device_${id}`,
            expires: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(), // 30 days from now
            name: "My Device",
        };
        
        return {
            ...defaultPushDevice,
            ...overrides,
        };
    }
    
    /**
     * Create push device from API input
     */
    createPushDeviceFromInput(input: PushDeviceCreateInput): PushDevice {
        const pushDevice = this.createMockPushDevice();
        
        // Update push device based on input
        if (input.name) {
            pushDevice.name = input.name;
        }
        
        if (input.expires !== undefined && input.expires !== null) {
            // Convert seconds to date
            pushDevice.expires = new Date(Date.now() + input.expires * 1000).toISOString();
        }
        
        // Generate device ID from endpoint
        const endpointHash = input.endpoint.split("/").pop() || this.generateId();
        pushDevice.deviceId = `device_${endpointHash}`;
        
        return pushDevice;
    }
    
    /**
     * Create multiple push devices for different platforms
     */
    createPushDevicesForAllPlatforms(): PushDevice[] {
        return [
            // iOS device
            this.createMockPushDevice({
                name: "iPhone 14 Pro",
                deviceId: "ios_device_123",
                expires: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString(), // 1 year
            }),
            // Android device
            this.createMockPushDevice({
                name: "Samsung Galaxy S23",
                deviceId: "android_device_456",
                expires: new Date(Date.now() + 180 * 24 * 60 * 60 * 1000).toISOString(), // 6 months
            }),
            // Web browser (Chrome)
            this.createMockPushDevice({
                name: "Chrome on Windows",
                deviceId: "web_chrome_789",
                expires: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(), // 30 days
            }),
            // Web browser (Firefox)
            this.createMockPushDevice({
                name: "Firefox on macOS",
                deviceId: "web_firefox_abc",
                expires: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(), // 30 days
            }),
            // Expired device
            this.createMockPushDevice({
                name: "Old Device",
                deviceId: "expired_device_def",
                expires: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(), // Expired yesterday
            }),
            // No name device
            this.createMockPushDevice({
                name: null,
                deviceId: "unnamed_device_ghi",
            }),
        ];
    }
    
    /**
     * Validate push device create input
     */
    async validateCreateInput(input: PushDeviceCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await pushDeviceValidation.create.validate(input);
            return { valid: true };
        } catch (error: unknown) {
            const fieldErrors: Record<string, string> = {};
            
            if (error && typeof error === "object" && "inner" in error) {
                const validationError = error as { inner?: Array<{ path?: string; message: string }> };
                if (validationError.inner) {
                    validationError.inner.forEach((err) => {
                        if (err.path) {
                            fieldErrors[err.path] = err.message;
                        }
                    });
                }
            } else if (error && typeof error === "object" && "message" in error) {
                fieldErrors.general = String(error.message);
            }
            
            return {
                valid: false,
                errors: fieldErrors,
            };
        }
    }
    
    /**
     * Validate push device update input
     */
    async validateUpdateInput(input: PushDeviceUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await pushDeviceValidation.update.validate(input);
            return { valid: true };
        } catch (error: unknown) {
            const fieldErrors: Record<string, string> = {};
            
            if (error && typeof error === "object" && "inner" in error) {
                const validationError = error as { inner?: Array<{ path?: string; message: string }> };
                if (validationError.inner) {
                    validationError.inner.forEach((err) => {
                        if (err.path) {
                            fieldErrors[err.path] = err.message;
                        }
                    });
                }
            } else if (error && typeof error === "object" && "message" in error) {
                fieldErrors.general = String(error.message);
            }
            
            return {
                valid: false,
                errors: fieldErrors,
            };
        }
    }
}

/**
 * MSW handlers factory for push device endpoints
 */
export class PushDeviceMSWHandlers {
    private responseFactory: PushDeviceResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new PushDeviceResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all push device endpoints
     */
    createSuccessHandlers(): RestHandler[] {
        return [
            // Create push device
            rest.post(`${this.responseFactory["baseUrl"]}/api/push-device`, async (req, res, ctx) => {
                const body = await req.json() as PushDeviceCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                // Create push device
                const pushDevice = this.responseFactory.createPushDeviceFromInput(body);
                const response = this.responseFactory.createSuccessResponse(pushDevice);
                
                return res(
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            // Get push device by ID
            rest.get(`${this.responseFactory["baseUrl"]}/api/push-device/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const pushDevice = this.responseFactory.createMockPushDevice({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(pushDevice);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Update push device
            rest.put(`${this.responseFactory["baseUrl"]}/api/push-device/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as PushDeviceUpdateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateUpdateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                const pushDevice = this.responseFactory.createMockPushDevice({ 
                    id: id as string,
                    name: body.name || undefined,
                    updatedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(pushDevice);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Delete push device
            rest.delete(`${this.responseFactory["baseUrl"]}/api/push-device/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // List push devices
            rest.get(`${this.responseFactory["baseUrl"]}/api/push-device`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const includeExpired = url.searchParams.get("includeExpired") === "true";
                
                let pushDevices = this.responseFactory.createPushDevicesForAllPlatforms();
                
                // Filter out expired devices if not including them
                if (!includeExpired) {
                    pushDevices = pushDevices.filter(d => !d.expires || new Date(d.expires) > new Date());
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedDevices = pushDevices.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createPushDeviceListResponse(
                    paginatedDevices,
                    {
                        page,
                        pageSize: limit,
                        totalCount: pushDevices.length,
                    },
                );
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Test push notification
            rest.post(`${this.responseFactory["baseUrl"]}/api/push-device/test`, async (req, res, ctx) => {
                const body = await req.json() as PushDeviceTestInput;
                
                // Simulate success
                const response = this.responseFactory.createPushTestSuccessResponse();
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
        ];
    }
    
    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): RestHandler[] {
        return [
            // Validation error
            rest.post(`${this.responseFactory["baseUrl"]}/api/push-device`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        endpoint: "A valid URL is required",
                        "keys.p256dh": "Public key is required",
                        "keys.auth": "Auth secret is required",
                    })),
                );
            }),
            
            // Not found error
            rest.get(`${this.responseFactory["baseUrl"]}/api/push-device/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string)),
                );
            }),
            
            // Permission error
            rest.post(`${this.responseFactory["baseUrl"]}/api/push-device`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("create")),
                );
            }),
            
            // Duplicate device error
            rest.post(`${this.responseFactory["baseUrl"]}/api/push-device`, (req, res, ctx) => {
                return res(
                    ctx.status(409),
                    ctx.json(this.responseFactory.createDuplicateDeviceErrorResponse("https://fcm.googleapis.com/fcm/send/example")),
                );
            }),
            
            // Invalid subscription error
            rest.post(`${this.responseFactory["baseUrl"]}/api/push-device`, (req, res, ctx) => {
                return res(
                    ctx.status(422),
                    ctx.json(this.responseFactory.createInvalidSubscriptionErrorResponse()),
                );
            }),
            
            // Push test failure
            rest.post(`${this.responseFactory["baseUrl"]}/api/push-device/test`, (req, res, ctx) => {
                return res(
                    ctx.status(500),
                    ctx.json(this.responseFactory.createPushTestFailureResponse("Device is offline")),
                );
            }),
            
            // Server error
            rest.post(`${this.responseFactory["baseUrl"]}/api/push-device`, (req, res, ctx) => {
                return res(
                    ctx.status(500),
                    ctx.json(this.responseFactory.createServerErrorResponse()),
                );
            }),
        ];
    }
    
    /**
     * Create loading simulation handlers
     */
    createLoadingHandlers(delay = 2000): RestHandler[] {
        return [
            rest.post(`${this.responseFactory["baseUrl"]}/api/push-device`, async (req, res, ctx) => {
                const body = await req.json() as PushDeviceCreateInput;
                const pushDevice = this.responseFactory.createPushDeviceFromInput(body);
                const response = this.responseFactory.createSuccessResponse(pushDevice);
                
                return res(
                    ctx.delay(delay),
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            rest.post(`${this.responseFactory["baseUrl"]}/api/push-device/test`, async (req, res, ctx) => {
                const response = this.responseFactory.createPushTestSuccessResponse();
                
                return res(
                    ctx.delay(delay),
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
        ];
    }
    
    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers(): RestHandler[] {
        return [
            rest.post(`${this.responseFactory["baseUrl"]}/api/push-device`, (req, res, ctx) => {
                return res.networkError("Network connection failed");
            }),
            
            rest.get(`${this.responseFactory["baseUrl"]}/api/push-device/:id`, (req, res, ctx) => {
                return res.networkError("Connection timeout");
            }),
            
            rest.post(`${this.responseFactory["baseUrl"]}/api/push-device/test`, (req, res, ctx) => {
                return res.networkError("Unable to reach push notification service");
            }),
        ];
    }
    
    /**
     * Create custom handler with specific configuration
     */
    createCustomHandler(config: {
        endpoint: string;
        method: "GET" | "POST" | "PUT" | "DELETE";
        status: number;
        response: unknown;
        delay?: number;
    }): RestHandler {
        const { endpoint, method, status, response, delay } = config;
        const fullEndpoint = `${this.responseFactory["baseUrl"]}${endpoint}`;
        
        return rest[method.toLowerCase() as keyof typeof rest](fullEndpoint, (req, res, ctx) => {
            const responseCtx = [ctx.status(status), ctx.json(response)];
            
            if (delay) {
                responseCtx.unshift(ctx.delay(delay));
            }
            
            return res(...responseCtx);
        });
    }
}

/**
 * Pre-configured response scenarios
 */
export const pushDeviceResponseScenarios = {
    // Success scenarios
    createSuccess: (pushDevice?: PushDevice) => {
        const factory = new PushDeviceResponseFactory();
        return factory.createSuccessResponse(
            pushDevice || factory.createMockPushDevice(),
        );
    },
    
    listSuccess: (pushDevices?: PushDevice[]) => {
        const factory = new PushDeviceResponseFactory();
        return factory.createPushDeviceListResponse(
            pushDevices || factory.createPushDevicesForAllPlatforms(),
        );
    },
    
    testSuccess: () => {
        const factory = new PushDeviceResponseFactory();
        return factory.createPushTestSuccessResponse();
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new PushDeviceResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                endpoint: "A valid URL is required",
                "keys.p256dh": "Public key is required",
                "keys.auth": "Auth secret is required",
            },
        );
    },
    
    notFoundError: (deviceId?: string) => {
        const factory = new PushDeviceResponseFactory();
        return factory.createNotFoundErrorResponse(
            deviceId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new PushDeviceResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    duplicateError: (endpoint?: string) => {
        const factory = new PushDeviceResponseFactory();
        return factory.createDuplicateDeviceErrorResponse(
            endpoint || "https://fcm.googleapis.com/fcm/send/existing-endpoint",
        );
    },
    
    invalidSubscriptionError: () => {
        const factory = new PushDeviceResponseFactory();
        return factory.createInvalidSubscriptionErrorResponse();
    },
    
    testFailureError: (reason?: string) => {
        const factory = new PushDeviceResponseFactory();
        return factory.createPushTestFailureResponse(
            reason || "Device is offline",
        );
    },
    
    serverError: () => {
        const factory = new PushDeviceResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new PushDeviceMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new PushDeviceMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new PushDeviceMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new PushDeviceMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const pushDeviceResponseFactory = new PushDeviceResponseFactory();
export const pushDeviceMSWHandlers = new PushDeviceMSWHandlers();
