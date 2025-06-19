/**
 * Example tests demonstrating how to use event fixtures
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { MockSocketEmitter, waitForEvent, collectEvents, assertEventEmitted } from "@vrooli/ui/__test/fixtures/events";
import { socketEventFixtures } from "./socketEvents";
import { chatEventFixtures } from "./chatEvents";
import { swarmEventFixtures } from "./swarmEvents";
import { notificationEventFixtures } from "./notificationEvents";
import { collaborationEventFixtures } from "./collaborationEvents";
import { systemEventFixtures } from "./systemEvents";

describe("Event Fixtures Examples", () => {
    let socket: MockSocketEmitter;

    beforeEach(() => {
        socket = new MockSocketEmitter();
    });

    describe("Socket Events", () => {
        it("should handle connection lifecycle", async () => {
            // Test connection
            expect(socket.isConnected()).toBe(true);
            assertEventEmitted(socket, "connect");

            // Test disconnection
            socket.disconnect();
            expect(socket.isConnected()).toBe(false);
            assertEventEmitted(socket, "disconnect");
        });

        it("should handle room events with callbacks", async () => {
            const callback = vi.fn();

            // Join chat room
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
            expect(history).toHaveLength(3);
            expect(history[0].event).toBe("connect");
            expect(history[1].event).toBe("joinUserRoom");
            expect(history[2].event).toBe("joinChatRoom");
        });
    });

    describe("Chat Events", () => {
        it("should handle message events", () => {
            const messageHandler = vi.fn();
            socket.on("messages", messageHandler);

            // Emit text message
            socket.emit("messages", chatEventFixtures.messages.textMessage.data);
            expect(messageHandler).toHaveBeenCalledWith(chatEventFixtures.messages.textMessage.data);

            // Emit message with attachment
            socket.emit("messages", chatEventFixtures.messages.messageWithAttachment.data);
            expect(messageHandler).toHaveBeenCalledTimes(2);
        });

        it("should handle response streaming", async () => {
            const chunks: string[] = [];
            let finalMessage: string | undefined;

            socket.on("responseStream", (data) => {
                if (data.__type === "stream" && data.chunk) {
                    chunks.push(data.chunk);
                } else if (data.__type === "end" && data.finalMessage) {
                    finalMessage = data.finalMessage;
                }
            });

            // Emit stream sequence
            await socket.emitSequence([
                { event: "responseStream", data: chatEventFixtures.responseStream.streamStart.data },
                { delay: 100 },
                { event: "responseStream", data: chatEventFixtures.responseStream.streamChunk.data },
                { delay: 100 },
                { event: "responseStream", data: chatEventFixtures.responseStream.streamEnd.data },
            ]);

            expect(chunks).toHaveLength(2);
            expect(chunks[0]).toBe("I'm thinking about");
            expect(chunks[1]).toBe(" your question...");
            expect(finalMessage).toBe("I'm thinking about your question... Here's my answer.");
        });

        it("should handle bot status updates", () => {
            const statusHandler = vi.fn();
            socket.on("botStatusUpdate", statusHandler);

            // Test thinking status
            socket.emit("botStatusUpdate", chatEventFixtures.botStatus.thinking.data);
            expect(statusHandler).toHaveBeenCalledWith(
                expect.objectContaining({
                    status: "thinking",
                    message: "Processing your request...",
                })
            );

            // Test tool calling
            socket.emit("botStatusUpdate", chatEventFixtures.botStatus.toolCalling.data);
            expect(statusHandler).toHaveBeenCalledWith(
                expect.objectContaining({
                    status: "tool_calling",
                    toolInfo: expect.objectContaining({
                        name: "search",
                    }),
                })
            );
        });

        it("should handle typing indicators", () => {
            const typingHandler = vi.fn();
            socket.on("typing", typingHandler);

            // User starts typing
            socket.emit("typing", chatEventFixtures.typing.userStartTyping.data);
            expect(typingHandler).toHaveBeenCalledWith({
                starting: ["user_123"],
            });

            // User stops typing
            socket.emit("typing", chatEventFixtures.typing.userStopTyping.data);
            expect(typingHandler).toHaveBeenCalledWith({
                stopping: ["user_123"],
            });
        });

        it("should execute chat flow sequences", async () => {
            const messages: any[] = [];
            const botStatuses: any[] = [];

            socket.on("messages", (data) => messages.push(data));
            socket.on("botStatusUpdate", (data) => botStatuses.push(data));

            await socket.emitSequence(chatEventFixtures.sequences.botResponseFlow);

            expect(botStatuses).toHaveLength(3);
            expect(botStatuses[0].status).toBe("thinking");
            expect(botStatuses[2].status).toBe("processing_complete");
        });
    });

    describe("Swarm Events", () => {
        it("should handle swarm state updates", () => {
            const stateHandler = vi.fn();
            socket.on("swarmStateUpdate", stateHandler);

            // Test state transitions
            socket.emit("swarmStateUpdate", swarmEventFixtures.state.uninitialized.data);
            socket.emit("swarmStateUpdate", swarmEventFixtures.state.starting.data);
            socket.emit("swarmStateUpdate", swarmEventFixtures.state.running.data);

            expect(stateHandler).toHaveBeenCalledTimes(3);
            expect(stateHandler).toHaveBeenLastCalledWith(
                expect.objectContaining({
                    state: "RUNNING",
                })
            );
        });

        it("should handle resource updates", () => {
            const resourceHandler = vi.fn();
            socket.on("swarmResourceUpdate", resourceHandler);

            // Test resource consumption
            socket.emit("swarmResourceUpdate", swarmEventFixtures.resources.initialAllocation.data);
            socket.emit("swarmResourceUpdate", swarmEventFixtures.resources.consumptionUpdate.data);

            expect(resourceHandler).toHaveBeenCalledTimes(2);
            expect(resourceHandler).toHaveBeenLastCalledWith(
                expect.objectContaining({
                    resources: {
                        allocated: 10000,
                        consumed: 3500,
                        remaining: 6500,
                    },
                })
            );
        });

        it("should execute swarm lifecycle sequence", async () => {
            const events = await collectEvents(socket, "swarmStateUpdate", 15000);

            // Start lifecycle sequence in background
            socket.emitSequence(swarmEventFixtures.sequences.basicLifecycle);

            // Wait for events
            await new Promise(resolve => setTimeout(resolve, 15000));

            expect(events.length).toBeGreaterThan(0);
        });
    });

    describe("Notification Events", () => {
        it("should handle various notification types", () => {
            const notificationHandler = vi.fn();
            socket.on("notification", notificationHandler);

            // Test different notification types
            socket.emit("notification", notificationEventFixtures.notifications.newMessage.data);
            socket.emit("notification", notificationEventFixtures.notifications.mention.data);
            socket.emit("notification", notificationEventFixtures.notifications.awardEarned.data);

            expect(notificationHandler).toHaveBeenCalledTimes(3);
            expect(notificationHandler).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "ChatMessage",
                })
            );
        });

        it("should handle API credit updates", () => {
            const creditHandler = vi.fn();
            socket.on("apiCredits", creditHandler);

            // Test credit updates
            socket.emit("apiCredits", notificationEventFixtures.apiCredits.creditUpdate.data);
            expect(creditHandler).toHaveBeenCalledWith({
                credits: "1000000",
            });

            // Test low credits
            socket.emit("apiCredits", notificationEventFixtures.apiCredits.lowCredits.data);
            expect(creditHandler).toHaveBeenCalledWith({
                credits: "100",
            });
        });

        it("should execute notification burst sequence", async () => {
            const notifications: any[] = [];
            socket.on("notification", (data) => notifications.push(data));

            await socket.emitSequence(notificationEventFixtures.sequences.notificationBurst);

            expect(notifications).toHaveLength(3);
            expect(notifications[0].type).toBe("ChatMessage");
            expect(notifications[1].type).toBe("Mention");
            expect(notifications[2].type).toBe("TeamInvite");
        });
    });

    describe("Collaboration Events", () => {
        it("should handle run task updates", () => {
            const taskHandler = vi.fn();
            socket.on("runTask", taskHandler);

            // Test task lifecycle
            socket.emit("runTask", collaborationEventFixtures.runTask.taskCreated.data);
            socket.emit("runTask", collaborationEventFixtures.runTask.taskInProgress.data);
            socket.emit("runTask", collaborationEventFixtures.runTask.taskCompleted.data);

            expect(taskHandler).toHaveBeenCalledTimes(3);
            expect(taskHandler).toHaveBeenLastCalledWith(
                expect.objectContaining({
                    status: "completed",
                })
            );
        });

        it("should handle decision requests", async () => {
            const decisionHandler = vi.fn();
            socket.on("runTaskDecisionRequest", decisionHandler);

            // Test simple decision
            socket.emit("runTaskDecisionRequest", collaborationEventFixtures.decisionRequest.simpleDecision.data);

            expect(decisionHandler).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "boolean",
                    title: "Proceed with analysis?",
                })
            );
        });

        it("should execute parallel task sequence", async () => {
            const tasks: any[] = [];
            socket.on("runTask", (data) => tasks.push(data));

            await socket.emitSequence(collaborationEventFixtures.sequences.parallelExecution);

            // Should have multiple tasks running in parallel
            const inProgressTasks = tasks.filter(t => t.status === "in_progress");
            expect(inProgressTasks.length).toBeGreaterThanOrEqual(2);
        });
    });

    describe("System Events", () => {
        it("should handle system status updates", () => {
            const statusHandler = vi.fn();
            socket.on("systemStatus", statusHandler);

            // Test different system states
            socket.emit("systemStatus", systemEventFixtures.status.healthy.data);
            socket.emit("systemStatus", systemEventFixtures.status.degraded.data);
            socket.emit("systemStatus", systemEventFixtures.status.critical.data);

            expect(statusHandler).toHaveBeenCalledTimes(3);
            expect(statusHandler).toHaveBeenLastCalledWith(
                expect.objectContaining({
                    status: "critical",
                    issues: expect.arrayContaining(["Database connection lost"]),
                })
            );
        });

        it("should handle deployment events", () => {
            const deployHandler = vi.fn();
            socket.on("deploymentStatus", deployHandler);

            // Test deployment flow
            socket.emit("deploymentStatus", systemEventFixtures.deployment.started.data);
            socket.emit("deploymentStatus", systemEventFixtures.deployment.progress.data);
            socket.emit("deploymentStatus", systemEventFixtures.deployment.completed.data);

            expect(deployHandler).toHaveBeenCalledTimes(3);
            expect(deployHandler).toHaveBeenLastCalledWith(
                expect.objectContaining({
                    status: "completed",
                    success: true,
                })
            );
        });

        it("should handle security alerts", () => {
            const securityHandler = vi.fn();
            socket.on("securityAlert", securityHandler);

            // Test security alert
            socket.emit("securityAlert", systemEventFixtures.security.alert.data);

            expect(securityHandler).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "suspicious_activity",
                    severity: "medium",
                })
            );
        });
    });

    describe("Factory Functions", () => {
        it("should create custom events using factories", () => {
            // Create custom message event
            const customMessage = chatEventFixtures.factories.createMessageEvent({
                id: "custom_msg",
                content: "Custom message",
            });

            const handler = vi.fn();
            socket.on("messages", handler);
            socket.emit(customMessage.event as any, customMessage.data);

            expect(handler).toHaveBeenCalledWith(
                expect.objectContaining({
                    added: expect.arrayContaining([
                        expect.objectContaining({
                            content: "Custom message",
                        }),
                    ]),
                })
            );
        });

        it("should create custom swarm state updates", () => {
            const customState = swarmEventFixtures.factories.createStateUpdate(
                "swarm_custom",
                "FAILED",
                "Custom failure message"
            );

            const handler = vi.fn();
            socket.on("swarmStateUpdate", handler);
            socket.emit(customState.event as any, customState.data);

            expect(handler).toHaveBeenCalledWith(
                expect.objectContaining({
                    swarmId: "swarm_custom",
                    state: "FAILED",
                    message: "Custom failure message",
                })
            );
        });
    });

    describe("Error Scenarios", () => {
        it("should handle stream errors", () => {
            const errorHandler = vi.fn();
            socket.on("responseStream", (data) => {
                if (data.__type === "error") {
                    errorHandler(data.error);
                }
            });

            socket.emit("responseStream", chatEventFixtures.responseStream.streamError.data);

            expect(errorHandler).toHaveBeenCalledWith(
                expect.objectContaining({
                    code: "LLM_ERROR",
                    retryable: true,
                })
            );
        });

        it("should handle connection errors", () => {
            const errorHandler = vi.fn();
            socket.on("error", errorHandler);

            socket.emit("error", socketEventFixtures.errors.unauthorized.data);
            socket.emit("error", socketEventFixtures.errors.rateLimit.data);

            expect(errorHandler).toHaveBeenCalledTimes(2);
            expect(errorHandler).toHaveBeenCalledWith(
                expect.objectContaining({
                    code: "RATE_LIMIT",
                    retryAfter: 60,
                })
            );
        });
    });

    describe("Waiting for Events", () => {
        it("should wait for specific events", async () => {
            // Emit event after delay
            setTimeout(() => {
                socket.emit("notification", notificationEventFixtures.notifications.taskComplete.data);
            }, 100);

            const notification = await waitForEvent(socket, "notification", 1000);
            expect(notification).toMatchObject({
                type: "RunComplete",
                title: "Task completed successfully",
            });
        });

        it("should timeout when event doesn't arrive", async () => {
            await expect(
                waitForEvent(socket, "notification", 100)
            ).rejects.toThrow("Timeout waiting for event: notification");
        });
    });
});