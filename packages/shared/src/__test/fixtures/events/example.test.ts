/**
 * Example tests demonstrating how to use the enhanced event fixtures
 * Shows both legacy patterns and new factory-based approaches
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { MockSocketEmitter } from "./MockSocketEmitter.js";
import { waitForEvent, collectEvents, generateCorrelationId, EventCorrelator, TimingAnalyzer, networkPresets } from "./eventUtils.js";
import { socketEventFixtures, connectionFactory, roomFactory, reconnectionFactory } from "./socketEvents.js";
import { chatEventFixtures } from "./chatEvents.js";
import { swarmEventFixtures } from "./swarmEvents.js";
import { notificationEventFixtures } from "./notificationEvents.js";
import { collaborationEventFixtures } from "./collaborationEvents.js";
import { systemEventFixtures } from "./systemEvents.js";
import { userJourneySequences, systemReliabilitySequences, comprehensiveScenarios, SequenceOrchestrator } from "./comprehensiveSequences.js";

describe("Enhanced Event Fixtures Examples", () => {
    let socket: MockSocketEmitter;

    beforeEach(() => {
        socket = new MockSocketEmitter({
            simulateNetwork: false,
            autoAcknowledge: true,
            stateTracking: true,
            correlationTracking: true
        });
    });

    describe("Legacy Event Patterns (Backward Compatibility)", () => {
        it("should handle connection lifecycle", async () => {
            socket.connect();
            expect(socket.isConnected()).toBe(true);
            socket.assertEventEmitted("connect");

            socket.disconnect();
            expect(socket.isConnected()).toBe(false);
            socket.assertEventEmitted("disconnect");
        });

        it("should handle room events with callbacks", async () => {
            const callback = vi.fn();

            socket.emitWithAck(
                "joinChatRoom",
                socketEventFixtures.room.joinChatSuccess.data,
                callback
            );

            await new Promise(resolve => setTimeout(resolve, 150));
            expect(callback).toHaveBeenCalledWith({ success: true });
            expect(socket.inRoom("chat:chat_123")).toBe(true);
        });

        it("should execute event sequences", async () => {
            await socket.emitSequence(socketEventFixtures.sequences.connectionFlow);

            const history = socket.getEmitHistory();
            expect(history.length).toBeGreaterThanOrEqual(3);
            socket.assertEventOrder(["connect", "joinUserRoom", "joinChatRoom"]);
        });
    });

    describe("Enhanced Factory Patterns", () => {
        it("should use connection factory for dynamic events", () => {
            // Create custom connection event
            const customConnection = connectionFactory.create({
                socketId: "custom_socket_456",
                transport: "polling"
            });

            const handler = vi.fn();
            socket.on("connect", handler);
            socket.emit(customConnection.event, customConnection.data);

            expect(handler).toHaveBeenCalledWith(
                expect.objectContaining({
                    socketId: "custom_socket_456",
                    transport: "polling"
                })
            );
        });

        it("should use room factory for various room types", () => {
            const roomHandler = vi.fn();
            socket.on("joinChatRoom", roomHandler);

            // Create chat room join event
            const chatJoin = roomFactory.create({
                roomType: "chat",
                roomId: "custom_chat_789",
                userId: "user_456"
            });

            socket.emit(chatJoin.event, chatJoin.data);

            expect(roomHandler).toHaveBeenCalledWith(
                expect.objectContaining({
                    roomType: "chat",
                    roomId: "custom_chat_789",
                    userId: "user_456"
                })
            );
        });

        it("should create event sequences with factory patterns", () => {
            // Create escalating reconnection sequence
            const reconnectSequence = reconnectionFactory.createSequence("escalating", { count: 3 });
            
            expect(reconnectSequence).toHaveLength(3);
            expect(reconnectSequence[0].data.attempt).toBe(1);
            expect(reconnectSequence[2].data.attempt).toBe(3);
            
            // Verify exponential backoff
            expect(reconnectSequence[1].data.delay).toBeGreaterThan(reconnectSequence[0].data.delay);
            expect(reconnectSequence[2].data.delay).toBeGreaterThan(reconnectSequence[1].data.delay);
        });

        it("should create correlated events", () => {
            const correlationId = generateCorrelationId();
            
            const correlatedEvents = connectionFactory.createCorrelated(correlationId, [
                connectionFactory.single,
                roomFactory.variants.joinUserSuccess as any,
                roomFactory.variants.joinChatSuccess as any
            ]);

            expect(correlatedEvents).toHaveLength(3);
            correlatedEvents.forEach(event => {
                expect(event.metadata.correlationId).toBe(correlationId);
            });
            
            // First event should not have causedBy
            expect(correlatedEvents[0].metadata.causedBy).toBeUndefined();
            
            // Second event should be caused by first
            expect(correlatedEvents[1].metadata.causedBy).toBe(correlatedEvents[0].event);
        });
    });

    describe("Enhanced MockSocketEmitter Features", () => {
        it("should track state changes", async () => {
            socket.setState({ connected: false, roomsJoined: 0 });

            await socket.emit("connect", { socketId: "test_socket" });
            await socket.emit("joinChatRoom", { chatId: "chat_123" });

            const state = socket.getState();
            expect(state.lastEvent).toBe("joinChatRoom");
            expect(state.lastEventTime).toBeDefined();
        });

        it("should simulate network conditions", async () => {
            // Enable network simulation with poor conditions
            socket.setNetworkCondition(networkPresets.mobile3G);

            const startTime = Date.now();
            await socket.emit("messages", { content: "Test message" });
            const endTime = Date.now();

            // Should have network delay
            expect(endTime - startTime).toBeGreaterThan(100); // At least some delay
        });

        it("should support parallel event emission", async () => {
            const messageHandler = vi.fn();
            socket.on("messages", messageHandler);

            await socket.emitParallel([
                { event: "messages", data: { added: [{ id: "msg_1", content: "Message 1" }] } },
                { event: "messages", data: { added: [{ id: "msg_2", content: "Message 2" }] } },
                { event: "messages", data: { added: [{ id: "msg_3", content: "Message 3" }] } }
            ]);

            expect(messageHandler).toHaveBeenCalledTimes(3);
        });

        it("should provide detailed emit history", () => {
            socket.emit("connect", {});
            socket.emit("joinChatRoom", { chatId: "test" });
            socket.emit("messages", { content: "test" });

            const timedHistory = socket.getTimedHistory();
            expect(timedHistory).toHaveLength(3);
            
            // First event has no delta
            expect(timedHistory[0].delta).toBe(0);
            
            // Subsequent events have timing deltas
            expect(timedHistory[1].delta).toBeGreaterThanOrEqual(0);
            expect(timedHistory[2].delta).toBeGreaterThanOrEqual(0);
        });

        it("should support assertion helpers", () => {
            socket.emit("connect", {});
            socket.emit("joinChatRoom", { chatId: "test_123" });
            socket.emit("messages", { content: "Hello" });
            socket.emit("disconnect", {});

            // Test various assertions
            socket.assertEventEmitted("connect");
            socket.assertEventNotEmitted("error");
            socket.assertEventCount("messages", 1);
            socket.assertEventOrder(["connect", "joinChatRoom", "messages", "disconnect"]);
        });
    });

    describe("Comprehensive Sequences", () => {
        it("should execute new user onboarding journey", async () => {
            const events: any[] = [];
            
            // Capture all events
            ["connect", "joinUserRoom", "joinChatRoom", "messages", "notification"].forEach(eventType => {
                socket.on(eventType, (data) => events.push({ type: eventType, data }));
            });

            await socket.emitSequence(userJourneySequences.newUserOnboarding);

            // Verify journey completion
            expect(events.length).toBeGreaterThan(10);
            
            const eventTypes = events.map(e => e.type);
            expect(eventTypes).toContain("connect");
            expect(eventTypes).toContain("joinUserRoom");
            expect(eventTypes).toContain("joinChatRoom");
            expect(eventTypes).toContain("messages");
            expect(eventTypes).toContain("notification");
        });

        it("should handle advanced user workflow", async () => {
            const taskEvents: any[] = [];
            const swarmEvents: any[] = [];
            
            socket.on("runTask", (data) => taskEvents.push(data));
            socket.on("swarmStateUpdate", (data) => swarmEvents.push(data));

            await socket.emitSequence(userJourneySequences.advancedUserWorkflow);

            expect(taskEvents.length).toBeGreaterThan(0);
            expect(swarmEvents.length).toBeGreaterThan(0);
            
            // Verify workflow progression
            const taskStatuses = taskEvents.map(t => t.status);
            expect(taskStatuses).toContain("created");
            expect(taskStatuses).toContain("in_progress");
            expect(taskStatuses).toContain("completed");
        });

        it("should simulate system degradation and recovery", async () => {
            const systemEvents: any[] = [];
            const errorEvents: any[] = [];
            
            socket.on("systemStatus", (data) => systemEvents.push(data));
            socket.on("responseStream", (data) => {
                if (data.__type === "error") errorEvents.push(data);
            });

            await socket.emitSequence(systemReliabilitySequences.gracefulDegradation);

            expect(systemEvents.length).toBeGreaterThan(0);
            expect(errorEvents.length).toBeGreaterThan(0);

            // Verify degradation and recovery cycle
            const healthStatuses = systemEvents.map(e => e.status);
            expect(healthStatuses).toContain("healthy");
            expect(healthStatuses).toContain("degraded");
            expect(healthStatuses).toContain("recovering");
        });
    });

    describe("Event Correlation and Analysis", () => {
        it("should track event correlations", async () => {
            const correlator = new EventCorrelator();
            const correlationId = generateCorrelationId();

            // Create correlated events
            const events = connectionFactory.createCorrelated(correlationId, [
                connectionFactory.single,
                roomFactory.variants.joinUserSuccess as any
            ]);

            // Track events
            events.forEach(event => correlator.track(event));

            // Verify correlation
            const correlatedEvents = correlator.getCorrelatedEvents(correlationId);
            expect(correlatedEvents).toHaveLength(2);

            const chain = correlator.getEventChain(correlationId);
            expect(chain).toEqual(["connect", "joinUserRoom"]);

            // Test pattern matching
            expect(correlator.matchesPattern(correlationId, ["connect", "joinUserRoom"])).toBe(true);
            expect(correlator.matchesPattern(correlationId, ["connect", "*"])).toBe(true);
            expect(correlator.matchesPattern(correlationId, ["disconnect", "joinUserRoom"])).toBe(false);
        });

        it("should analyze event timing", async () => {
            const analyzer = new TimingAnalyzer();

            // Record events with timing
            analyzer.record("connect");
            await new Promise(resolve => setTimeout(resolve, 100));
            analyzer.record("joinRoom");
            await new Promise(resolve => setTimeout(resolve, 50));
            analyzer.record("message");

            const stats = analyzer.getStats();
            expect(stats.eventCount).toBe(3);
            expect(stats.totalDuration).toBeGreaterThan(140); // At least 150ms
            expect(stats.averageInterval).toBeGreaterThan(0);
            expect(stats.eventFrequency.get("connect")).toBe(1);

            // Test timing relationships
            expect(analyzer.withinWindow("connect", "joinRoom", 200)).toBe(true);
            expect(analyzer.withinWindow("connect", "message", 50)).toBe(false);
        });
    });

    describe("Sequence Orchestration", () => {
        it("should orchestrate complex scenarios", async () => {
            const orchestrator = new SequenceOrchestrator({
                simulation: { timing: "fast", network: "fast" }
            });

            const result = await orchestrator.executeSequence([
                { event: "connect", data: {} },
                { delay: 100 },
                { event: "joinChatRoom", data: { chatId: "test" } },
                { delay: 200 },
                { event: "messages", data: { content: "Hello" } }
            ]);

            expect(result.success).toBe(true);
            expect(result.events).toHaveLength(3);
            expect(result.correlationId).toBeDefined();
            expect(result.duration).toBeGreaterThan(0);
        });

        it("should combine multiple sequences", () => {
            const orchestrator = new SequenceOrchestrator();

            const combined = orchestrator.combineSequences([
                { 
                    name: "connection", 
                    sequence: [{ event: "connect", data: {} }], 
                    delay: 1000 
                },
                { 
                    name: "chat", 
                    sequence: [{ event: "joinChatRoom", data: { chatId: "test" } }], 
                    delay: 500 
                }
            ]);

            expect(combined).toHaveLength(4); // connect + delay + joinChatRoom + delay
            expect(combined[0].event).toBe("connect");
            expect(combined[1].delay).toBe(1000);
            expect(combined[2].event).toBe("joinChatRoom");
        });

        it("should create parallel scenarios", () => {
            const orchestrator = new SequenceOrchestrator();

            const parallel = orchestrator.createParallelScenario([
                { 
                    name: "user1", 
                    sequence: [{ event: "joinChat", data: { userId: "user_1" } }],
                    userContext: "user_1"
                },
                { 
                    name: "user2", 
                    sequence: [{ event: "joinChat", data: { userId: "user_2" } }],
                    userContext: "user_2"
                }
            ]);

            expect(parallel[0].parallel).toHaveLength(2);
            expect(parallel[0].parallel![0].data).toMatchObject({ scenario: "user1", user: "user_1" });
        });
    });

    describe("Advanced Chat Scenarios", () => {
        it("should handle bot tool approval workflow", async () => {
            const toolEvents: any[] = [];
            const statusEvents: any[] = [];
            
            socket.on("tool_approval_required", (data) => toolEvents.push({ type: "required", data }));
            socket.on("tool_approval_rejected", (data) => toolEvents.push({ type: "rejected", data }));
            socket.on("botStatusUpdate", (data) => statusEvents.push(data));

            await socket.emitSequence(chatEventFixtures.sequences.toolApprovalFlow);

            expect(toolEvents.length).toBeGreaterThan(0);
            expect(statusEvents.length).toBeGreaterThan(0);

            // Verify approval workflow
            const approvalRequired = toolEvents.find(e => e.type === "required");
            expect(approvalRequired).toBeDefined();
            expect(approvalRequired.data.toolName).toBe("execute_code");

            const approvalRejected = toolEvents.find(e => e.type === "rejected");
            expect(approvalRejected).toBeDefined();
            expect(approvalRejected.data.reason).toContain("declined");
        });

        it("should handle streaming with error recovery", async () => {
            const streamEvents: any[] = [];
            
            socket.on("responseStream", (data) => streamEvents.push(data));

            await socket.emitSequence(chatEventFixtures.sequences.errorRecoveryFlow);

            // Should have stream start, error, then recovery
            expect(streamEvents.length).toBeGreaterThan(3);
            
            const errorEvent = streamEvents.find(e => e.__type === "error");
            expect(errorEvent).toBeDefined();
            expect(errorEvent.error.retryable).toBe(true);

            // Should have retry after error
            const streamAfterError = streamEvents.slice(streamEvents.indexOf(errorEvent) + 1);
            expect(streamAfterError.some(e => e.__type === "stream")).toBe(true);
        });
    });

    describe("System Monitoring Scenarios", () => {
        it("should handle maintenance window", async () => {
            const maintenanceEvents: any[] = [];
            const notificationEvents: any[] = [];
            
            socket.on("maintenanceStatus", (data) => maintenanceEvents.push(data));
            socket.on("notification", (data) => notificationEvents.push(data));

            await socket.emitSequence(systemReliabilitySequences.plannedMaintenance);

            expect(maintenanceEvents.length).toBeGreaterThan(0);
            expect(notificationEvents.length).toBeGreaterThan(0);

            // Verify maintenance progression
            const statuses = maintenanceEvents.map(e => e.status);
            expect(statuses).toContain("scheduled");
            expect(statuses).toContain("in_progress");
            expect(statuses).toContain("completed");
        });

        it("should handle security incident", async () => {
            const securityEvents: any[] = [];
            const errorEvents: any[] = [];
            
            socket.on("securityAlert", (data) => securityEvents.push(data));
            socket.on("error", (data) => errorEvents.push(data));

            await socket.emitSequence(systemReliabilitySequences.securityIncident);

            expect(securityEvents.length).toBeGreaterThan(0);
            
            // Verify incident escalation
            const severities = securityEvents.map(e => e.severity);
            expect(severities).toContain("low");
            expect(severities).toContain("high");

            // Should have lockdown error
            const lockdownError = errorEvents.find(e => e.code === "SECURITY_LOCKDOWN");
            expect(lockdownError).toBeDefined();
        });
    });

    describe("Performance Testing Utilities", () => {
        it("should test event waiting with timeout", async () => {
            // Emit event after delay
            setTimeout(() => {
                socket.emit("notification", { type: "test", message: "Delayed notification" });
            }, 100);

            const notification = await waitForEvent(socket, "notification", 1000);
            expect(notification.type).toBe("test");
            expect(notification.message).toBe("Delayed notification");
        });

        it("should collect events over time period", async () => {
            // Start emitting events
            const emitInterval = setInterval(() => {
                socket.emit("heartbeat", { timestamp: Date.now() });
            }, 50);

            const events = await collectEvents(socket, "heartbeat", 250);
            clearInterval(emitInterval);

            expect(events.length).toBeGreaterThanOrEqual(4); // At least 4 heartbeats in 250ms
            expect(events.length).toBeLessThanOrEqual(6); // But not too many
        });

        it("should timeout when waiting for events that don't arrive", async () => {
            await expect(
                waitForEvent(socket, "never-emitted-event", 100)
            ).rejects.toThrow("Timeout waiting for event: never-emitted-event");
        });
    });

    describe("Full-Scale Integration Test", () => {
        it("should execute comprehensive scenario", async () => {
            // Set up comprehensive monitoring
            const allEvents: any[] = [];
            const eventTypes = [
                "connect", "disconnect", "joinChatRoom", "leaveChatRoom",
                "messages", "responseStream", "botStatusUpdate", "typing",
                "participants", "notification", "runTask", "swarmStateUpdate",
                "systemStatus", "error"
            ];
            
            eventTypes.forEach(eventType => {
                socket.on(eventType, (data) => allEvents.push({ 
                    type: eventType, 
                    data, 
                    timestamp: Date.now() 
                }));
            });

            // Execute comprehensive scenario with timing
            const startTime = Date.now();
            await socket.emitSequence(comprehensiveScenarios.fullScale);
            const endTime = Date.now();

            // Verify comprehensive execution
            expect(allEvents.length).toBeGreaterThan(20);
            expect(endTime - startTime).toBeGreaterThan(100);

            // Verify we have events from multiple systems
            const uniqueEventTypes = [...new Set(allEvents.map(e => e.type))];
            expect(uniqueEventTypes.length).toBeGreaterThan(5);
            
            // Should include connection events
            expect(uniqueEventTypes).toContain("connect");
            expect(uniqueEventTypes).toContain("joinChatRoom");

            // Verify event ordering makes sense
            const connectIndex = allEvents.findIndex(e => e.type === "connect");
            const joinRoomIndex = allEvents.findIndex(e => e.type === "joinChatRoom");
            expect(joinRoomIndex).toBeGreaterThan(connectIndex);
        });
    });
});