import { describe, it, expect, beforeEach, vi } from "vitest";
import { MinimalSecurityValidator } from "../../cross-cutting/security/minimalSecurityValidator.js";
import { ExecutionEventEmitter } from "../../cross-cutting/monitoring/ExecutionEventEmitter.js";
import { logger } from "../../../../events/logger.js";

/**
 * Simple tests for MinimalSecurityValidator focusing on core functionality
 */
describe("MinimalSecurityValidator - Core Functionality", () => {
    let validator: MinimalSecurityValidator;
    let mockEventEmitter: ExecutionEventEmitter;
    
    beforeEach(() => {
        // Mock the event emitter completely
        mockEventEmitter = {
            emitExecutionEvent: vi.fn().mockResolvedValue(undefined),
            emitMetric: vi.fn().mockResolvedValue(undefined)
        } as any;
        
        validator = new MinimalSecurityValidator(logger, mockEventEmitter);
    });
    
    describe("checkPermission", () => {
        it("should return true when user has required permission", async () => {
            const result = await validator.checkPermission(
                "user123",
                "routine",
                "execute",
                ["routine:read", "routine:execute", "routine:write"]
            );
            
            expect(result).toBe(true);
            
            // Should emit both execution event and metric
            expect(mockEventEmitter.emitExecutionEvent).toHaveBeenCalledOnce();
            expect(mockEventEmitter.emitMetric).toHaveBeenCalledOnce();
            
            // Check metric was emitted with success
            expect(mockEventEmitter.emitMetric).toHaveBeenCalledWith(
                expect.objectContaining({
                    name: "security.permission_check",
                    value: 1, // Success
                    metricType: "safety"
                })
            );
        });
        
        it("should return false when user lacks required permission", async () => {
            const result = await validator.checkPermission(
                "user123",
                "admin",
                "delete",
                ["user:read", "user:write"] // No admin permissions
            );
            
            expect(result).toBe(false);
            
            // Should still emit events for failed attempts
            expect(mockEventEmitter.emitExecutionEvent).toHaveBeenCalledOnce();
            expect(mockEventEmitter.emitMetric).toHaveBeenCalledWith(
                expect.objectContaining({
                    name: "security.permission_check",
                    value: 0, // Failure
                    tags: expect.objectContaining({
                        resource: "admin",
                        action: "delete"
                    })
                })
            );
        });
    });
    
    describe("checkAuthentication", () => {
        it("should return true for authenticated user", async () => {
            const result = await validator.checkAuthentication("user123");
            
            expect(result).toBe(true);
            expect(mockEventEmitter.emitMetric).toHaveBeenCalledWith(
                expect.objectContaining({
                    name: "security.authentication_check",
                    value: 1
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
        it("should check both authentication and permission", async () => {
            const result = await validator.checkResourceAccess(
                "user123",
                "team456",
                "api",
                "call",
                ["api:call", "api:read"]
            );
            
            expect(result).toBe(true);
            
            // Should emit 3 events total:
            // 1. Authentication check
            // 2. Permission check  
            // 3. Resource access summary
            expect(mockEventEmitter.emitExecutionEvent).toHaveBeenCalledTimes(3);
            expect(mockEventEmitter.emitMetric).toHaveBeenCalledTimes(3);
        });
        
        it("should fail if user not authenticated", async () => {
            const result = await validator.checkResourceAccess(
                undefined, // No user
                "team456",
                "api",
                "call",
                ["api:call"]
            );
            
            expect(result).toBe(false);
            
            // Should only check authentication, not permission
            expect(mockEventEmitter.emitExecutionEvent).toHaveBeenCalledOnce();
        });
        
        it("should fail if user lacks permission", async () => {
            const result = await validator.checkResourceAccess(
                "user123",
                "team456",
                "admin",
                "delete",
                ["user:read", "user:write"] // No admin:delete
            );
            
            expect(result).toBe(false);
            
            // Should still emit all events
            expect(mockEventEmitter.emitExecutionEvent).toHaveBeenCalledTimes(3);
            
            // Final metric should show failure
            const calls = (mockEventEmitter.emitMetric as any).mock.calls;
            const resourceAccessCall = calls.find((call: any[]) => 
                call[0].name === "security.resource_access"
            );
            expect(resourceAccessCall[0].value).toBe(0);
        });
    });
    
    describe("Event emission patterns for agents", () => {
        it("should provide sufficient data for threat detection", async () => {
            // Simulate suspicious activity
            const suspiciousActions = [
                { resource: "system", action: "shutdown" },
                { resource: "database", action: "drop" },
                { resource: "admin", action: "elevate" }
            ];
            
            for (const { resource, action } of suspiciousActions) {
                await validator.checkPermission(
                    "potential-attacker",
                    resource,
                    action,
                    ["user:read"] // Basic permission only
                );
            }
            
            // Check that all failed attempts were logged
            expect(mockEventEmitter.emitMetric).toHaveBeenCalledTimes(3);
            
            // Each call should indicate failure with proper context
            const metricCalls = (mockEventEmitter.emitMetric as any).mock.calls;
            metricCalls.forEach((call: any[], index: number) => {
                expect(call[0]).toMatchObject({
                    name: "security.permission_check",
                    value: 0, // All failed
                    tags: {
                        resource: suspiciousActions[index].resource,
                        action: suspiciousActions[index].action,
                        userId: "potential-attacker"
                    }
                });
            });
        });
        
        it("should emit events that agents can correlate", async () => {
            const userId = "user123";
            const teamId = "team456";
            
            // User tries to access multiple resources
            await validator.checkResourceAccess(userId, teamId, "api", "read", ["api:read"]);
            await validator.checkResourceAccess(userId, teamId, "api", "write", ["api:read"]); // Will fail
            
            // Agents can correlate these events by userId and see access patterns
            const executionCalls = (mockEventEmitter.emitExecutionEvent as any).mock.calls;
            const userEvents = executionCalls.filter((call: any[]) => 
                call[0].data?.securityEvent?.userId === userId
            );
            
            expect(userEvents.length).toBeGreaterThan(0);
        });
    });
});