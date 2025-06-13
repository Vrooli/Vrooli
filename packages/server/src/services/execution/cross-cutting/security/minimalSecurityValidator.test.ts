import { describe, it, expect, beforeEach, vi } from "vitest";
import { MinimalSecurityValidator } from "./minimalSecurityValidator.js";
import { ExecutionEventEmitter } from "../monitoring/ExecutionEventEmitter.js";
import { logger } from "../../../../events/logger.js";

describe("MinimalSecurityValidator", () => {
    let validator: MinimalSecurityValidator;
    let mockEventEmitter: ExecutionEventEmitter;
    
    beforeEach(() => {
        mockEventEmitter = {
            emitExecutionEvent: vi.fn().mockResolvedValue(undefined),
            emitMetric: vi.fn().mockResolvedValue(undefined)
        } as any;
        
        validator = new MinimalSecurityValidator(logger, mockEventEmitter);
    });
    
    describe("checkPermission", () => {
        it("should return true when permission exists", async () => {
            const result = await validator.checkPermission(
                "user123",
                "routine",
                "execute",
                ["routine:read", "routine:execute", "routine:write"]
            );
            
            expect(result).toBe(true);
            expect(mockEventEmitter.emitExecutionEvent).toHaveBeenCalledWith({
                executionId: expect.stringMatching(/^security-\d+$/),
                event: "completed",
                tier: "cross-cutting",
                component: "MinimalSecurityValidator",
                data: {
                    securityEvent: {
                        type: "permission_check",
                        userId: "user123",
                        resource: "routine",
                        action: "execute",
                        permissions: ["routine:read", "routine:execute", "routine:write"],
                        result: true,
                        timestamp: expect.any(Date)
                    }
                }
            });
        });
        
        it("should return false when permission missing", async () => {
            const result = await validator.checkPermission(
                "user123",
                "routine",
                "delete",
                ["routine:read", "routine:execute"]
            );
            
            expect(result).toBe(false);
            expect(mockEventEmitter.emitMetric).toHaveBeenCalledWith({
                tier: "cross-cutting",
                component: "MinimalSecurityValidator",
                metricType: "safety",
                name: "security.permission_check",
                value: 0,
                tags: {
                    resource: "routine",
                    action: "delete",
                    userId: "user123"
                }
            });
        });
        
        it("should handle undefined userId", async () => {
            const result = await validator.checkPermission(
                undefined,
                "routine",
                "read",
                ["routine:read"]
            );
            
            expect(result).toBe(true);
            expect(mockEventEmitter.emitMetric).toHaveBeenCalledWith(
                expect.objectContaining({
                    tags: expect.objectContaining({
                        userId: "anonymous"
                    })
                })
            );
        });
    });
    
    describe("checkAuthentication", () => {
        it("should return true for authenticated user", async () => {
            const result = await validator.checkAuthentication("user123");
            
            expect(result).toBe(true);
            expect(mockEventEmitter.emitExecutionEvent).toHaveBeenCalledWith(
                expect.objectContaining({
                    data: {
                        securityEvent: expect.objectContaining({
                            type: "authentication_check",
                            userId: "user123",
                            result: true
                        })
                    }
                })
            );
        });
        
        it("should return false for unauthenticated user", async () => {
            const result = await validator.checkAuthentication(undefined);
            
            expect(result).toBe(false);
            expect(mockEventEmitter.emitMetric).toHaveBeenCalledWith(
                expect.objectContaining({
                    name: "security.authentication_check",
                    value: 0
                })
            );
        });
    });
    
    describe("checkResourceAccess", () => {
        it("should return true when authenticated and has permission", async () => {
            const result = await validator.checkResourceAccess(
                "user123",
                "team456",
                "api",
                "call",
                ["api:call", "api:read"]
            );
            
            expect(result).toBe(true);
            
            // Should emit 3 events: authentication, permission, and resource access
            expect(mockEventEmitter.emitExecutionEvent).toHaveBeenCalledTimes(3);
            expect(mockEventEmitter.emitMetric).toHaveBeenCalledTimes(3);
            
            // Check resource access event
            expect(mockEventEmitter.emitExecutionEvent).toHaveBeenCalledWith(
                expect.objectContaining({
                    data: {
                        securityEvent: expect.objectContaining({
                            type: "resource_access",
                            userId: "user123",
                            teamId: "team456",
                            resource: "api",
                            action: "call",
                            result: true,
                            metadata: {
                                authenticated: true,
                                permitted: true
                            }
                        })
                    }
                })
            );
        });
        
        it("should return false when not authenticated", async () => {
            const result = await validator.checkResourceAccess(
                undefined,
                "team456",
                "api",
                "call",
                ["api:call"]
            );
            
            expect(result).toBe(false);
            
            // Should only emit authentication check, not permission check
            expect(mockEventEmitter.emitExecutionEvent).toHaveBeenCalledTimes(1);
        });
        
        it("should return false when authenticated but no permission", async () => {
            const result = await validator.checkResourceAccess(
                "user123",
                "team456",
                "api",
                "delete",
                ["api:read", "api:call"]
            );
            
            expect(result).toBe(false);
            
            // Should emit all 3 events even though permission failed
            expect(mockEventEmitter.emitExecutionEvent).toHaveBeenCalledTimes(3);
            
            // Final resource access event should show failure
            expect(mockEventEmitter.emitMetric).toHaveBeenNthCalledWith(
                3,
                expect.objectContaining({
                    name: "security.resource_access",
                    value: 0
                })
            );
        });
    });
    
    describe("event emission", () => {
        it("should emit both execution and metric events", async () => {
            await validator.checkPermission(
                "user123",
                "routine",
                "execute",
                ["routine:execute"]
            );
            
            expect(mockEventEmitter.emitExecutionEvent).toHaveBeenCalledOnce();
            expect(mockEventEmitter.emitMetric).toHaveBeenCalledOnce();
        });
        
        it("should include proper event structure", async () => {
            const timestamp = new Date();
            vi.setSystemTime(timestamp);
            
            await validator.checkPermission(
                "user123",
                "routine",
                "execute",
                ["routine:execute"]
            );
            
            expect(mockEventEmitter.emitExecutionEvent).toHaveBeenCalledWith({
                executionId: expect.stringMatching(/^security-\d+$/),
                event: "completed",
                tier: "cross-cutting",
                component: "MinimalSecurityValidator",
                data: {
                    securityEvent: {
                        type: "permission_check",
                        userId: "user123",
                        resource: "routine",
                        action: "execute",
                        permissions: ["routine:execute"],
                        result: true,
                        timestamp
                    }
                }
            });
            
            vi.useRealTimers();
        });
    });
});