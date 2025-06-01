import { type ReservedSocketEvents, type RoomSocketEvents, type SocketEvent, type SocketEventHandler, type SocketEventPayloads } from "@local/shared";
import { createAdapter } from "@socket.io/redis-adapter";
import IORedis, { type Cluster, type Redis } from "ioredis";
import { Server, type Socket } from "socket.io";
import { AuthTokensService } from "../auth/auth.js";
import { RequestService } from "../auth/request.js";
import { SessionService } from "../auth/session.js";
import { logger } from "../events/logger.js";
import { getRedisUrl } from "../redisConn.js";
import { server } from "../server.js";
import { type SessionData } from "../types.js";

// Define possible states
type SocketServiceState = "uninitialized" | "initializing" | "initialized";

// Moved from events.ts
type EmitSocketEvent = Exclude<SocketEvent, ReservedSocketEvents | RoomSocketEvents>;
type OnSocketEvent = Exclude<SocketEvent, ReservedSocketEvents>;

// Define the structure for health details
export interface SocketHealthDetails {
    connectedClients: number;
    activeRooms: number;
    namespaces: number;
}

/* --------------------------------------------------------------------------
 * socketService.ts – Socket.IO singleton with Redis adapter
 * --------------------------------------------------------------------------
 *
 *  ▌ Overview
 *  • Horizontally-scalable Socket.IO server powered by the @socket.io/redis-adapter
 *    and backed by **ioredis**. Every instance shares Pub/Sub so rooms, broadcasts,
 *    and disconnect commands propagate cluster-wide.
 *  • Local maps (`userSockets`, `sessionSockets`) remain instance-local; the adapter
 *    handles the cross-instance fan-out.
 *  • Extensive inline docs explain the control-flow for future maintainers & AI agents.
 *
 *  ▌ Why ioredis?
 *  • We already depend on ioredis for BullMQ and other services; consolidating drivers
 *    reduces image size, cold-start time, and mental overhead.
 *  • ioredis supports clusters, sentinels, and auto-reconnect out-of-the-box.
 * -------------------------------------------------------------------------- */
export class SocketService {
    private static instance: SocketService | null = null;
    private static state: SocketServiceState = "uninitialized";
    private static initializationPromise: Promise<SocketService> | null = null;
    private static currentSigtermHandler: (() => Promise<void>) | null = null;

    /**
     * Active socket IDs by user ID for clients connected *directly to this server instance*.
     * While this map is local to this instance, the configured Redis adapter ensures that
     * operations spanning multiple instances (e.g., emitting to a room, fetching all
     * sockets in a room, or disconnecting a user's sockets) are correctly coordinated
     * across the entire cluster. For example, when `socket.disconnect(true)` is called
     * via `closeUserSockets`, the adapter helps propagate this state.
     */
    public userSockets!: Map<string, Set<string>>;

    /**
     * Active socket IDs by session ID for clients connected *directly to this server instance*.
     * Similar to `userSockets`, this map is local. The Redis adapter handles the
     * necessary inter-server communication for operations that need a cluster-wide view
     * or effect.
     */
    public sessionSockets!: Map<string, Set<string>>;

    /**
     * Socket.io server instance
     */
    public io!: Server;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    /**
     * Asynchronously initializes the SocketService instance.
     * Ensures RequestService is ready before configuring CORS.
     * This method is idempotent: if called multiple times, it will either return
     * the promise of the ongoing initialization or resolve immediately if already initialized.
     * @returns A promise that resolves to the SocketService instance once initialized.
     */
    public static async init(): Promise<SocketService> {
        if (SocketService.instance) { // Already initialized and instance is set
            logger.debug("SocketService already initialized. Returning existing instance.");
            return SocketService.instance;
        }

        if (SocketService.initializationPromise) {
            logger.debug("SocketService initialization already in progress. Returning existing promise.");
            return SocketService.initializationPromise;
        }

        SocketService.state = "initializing"; // Set state for clarity, though promise is key
        logger.info("SocketService: Starting initialization...");

        SocketService.initializationPromise = (async () => {
            // Moved SIGTERM cleanup to the top of the IIFE.
            // This ensures any handler from a *previous completed or failed init* is cleared
            // before attempting to set up resources for the current initialization attempt.
            if (SocketService.currentSigtermHandler) {
                process.removeListener("SIGTERM", SocketService.currentSigtermHandler);
                logger.debug("SocketService: Removed previous SIGTERM handler during new initialization.");
                SocketService.currentSigtermHandler = null;
            }

            let pubClient: Redis | Cluster | undefined;
            let subClient: Redis | Cluster | undefined;

            try {
                // Create the instance internally first
                const instance = new SocketService();
                instance.userSockets = new Map<string, Set<string>>();
                instance.sessionSockets = new Map<string, Set<string>>();

                const requestService = RequestService.get();
                const safeOrigins = requestService.safeOrigins();

                // --- Redis Adapter Setup (for horizontal scaling) ---
                let redisAdapter;
                try {
                    const redisUrl = getRedisUrl();
                    pubClient = new IORedis(redisUrl, { lazyConnect: true });
                    subClient = pubClient.duplicate();

                    await Promise.all([pubClient.connect(), subClient.connect()]);
                    logger.info("SocketService: ioredis clients connected.");

                    redisAdapter = createAdapter(pubClient, subClient);
                    logger.info("SocketService: Redis adapter initialized (ioredis backend).");

                    // Capture for closure, ensure they are defined if used in SIGTERM
                    const finalPubClient = pubClient;
                    const finalSubClient = subClient;

                    // Define and register the new SIGTERM handler only if Redis setup is successful
                    SocketService.currentSigtermHandler = async () => {
                        logger.info("SocketService: SIGTERM received (Redis mode), attempting to disconnect Redis clients.");
                        const promises: Promise<"OK" | void>[] = [];
                        if (finalPubClient) {
                            promises.push(finalPubClient.quit().catch((e: Error) => { logger.error("SocketService: Error quitting pubClient on SIGTERM", { error: e }); return; }));
                        }
                        if (finalSubClient) {
                            promises.push(finalSubClient.quit().catch((e: Error) => { logger.error("SocketService: Error quitting subClient on SIGTERM", { error: e }); return; }));
                        }
                        if (promises.length > 0) {
                            await Promise.all(promises);
                            logger.info("SocketService: Redis clients disconnection process completed via SIGTERM handler.");
                        } else {
                            logger.info("SocketService: No Redis clients for this SIGTERM handler instance to disconnect (Redis mode).");
                        }
                    };
                    process.on("SIGTERM", SocketService.currentSigtermHandler);
                    logger.info("SocketService: SIGTERM handler for Redis clients set.");

                } catch (redisError: unknown) {
                    if (redisError instanceof Error) {
                        logger.error("SocketService: Failed to connect Redis clients or create adapter. Socket.IO will run with in-memory adapter.", { error: redisError, trace: "SOCKET_REDIS_ADAPTER_FAIL" });
                    } else {
                        logger.error("SocketService: Failed to connect Redis clients or create adapter due to an unknown error type.", { error: String(redisError), trace: "SOCKET_REDIS_ADAPTER_FAIL_UNKNOWN_TYPE" });
                    }
                    // Attempt to clean up clients if they were instantiated during this failed attempt
                    const cleanupPromises: Promise<"OK" | void>[] = [];
                    if (pubClient) {
                        cleanupPromises.push(pubClient.quit().catch((e: Error) => { logger.error("SocketService: CRITICAL - Error quitting pubClient during adapter setup failure. Potential resource leak.", { error: e }); return; }));
                    }
                    if (subClient) {
                        cleanupPromises.push(subClient.quit().catch((e: Error) => { logger.error("SocketService: CRITICAL - Error quitting subClient during adapter setup failure. Potential resource leak.", { error: e }); return; }));
                    }
                    if (cleanupPromises.length > 0) {
                        await Promise.all(cleanupPromises); // Await cleanup before proceeding
                        logger.info("SocketService: Attempted cleanup of partially initialized Redis clients after adapter setup failure.");
                    }
                    // redisAdapter will remain undefined, Socket.IO uses the default in-memory adapter
                    // No new SIGTERM handler for Redis is set; any old one was cleared at the start of the IIFE.
                }
                // --- End Redis Adapter Setup ---

                // Create the Server instance with the required CORS config
                instance.io = new Server(server, {
                    cors: {
                        origin: safeOrigins,
                        methods: ["GET", "POST"],
                        credentials: true,
                    },
                    connectionStateRecovery: {}, // Existing option
                    ...(redisAdapter && { adapter: redisAdapter }), // Add adapter if successfully created
                });

                // --- Setup Server-Side Event Listeners for Global Actions ---
                instance.io.on("internal:disconnect_user_sockets", (userId: string) => {
                    logger.info(`[SocketService] Received command via serverSideEmit to disconnect sockets for user: ${userId} on this instance.`);
                    instance._performLocalUserSocketDisconnection(userId);
                });

                instance.io.on("internal:disconnect_session_sockets", (sessionId: string) => {
                    logger.info(`[SocketService] Received command via serverSideEmit to disconnect sockets for session: ${sessionId} on this instance.`);
                    instance._performLocalSessionSocketDisconnection(sessionId);
                });
                // --- End Server-Side Event Listeners ---

                SocketService.instance = instance; // Assign the fully initialized instance
                SocketService.state = "initialized"; // Mark as initialized
                logger.info(`SocketService initialized successfully ${redisAdapter ? "with Redis adapter" : "with in-memory adapter"}.`);
                return instance; // Resolve the promise with the instance

            } catch (error: unknown) { // Outer catch for the entire initialization IIFE
                if (error instanceof Error) {
                    logger.error("Failed to initialize SocketService", { error, trace: "SOCKET_INIT_FAIL" });
                } else {
                    logger.error("Failed to initialize SocketService due to an unknown error type", { error: String(error), trace: "SOCKET_INIT_FAIL_UNKNOWN_TYPE" });
                }

                // Cleanup Redis clients if they were initialized in this attempt but a subsequent step failed
                // (e.g. Redis setup was successful, pubClient/subClient are defined, but new Server() failed).
                const cleanupPromises: Promise<"OK" | void>[] = [];
                if (pubClient) { // pubClient is from the IIFE's scope
                    cleanupPromises.push(pubClient.quit().catch((e: Error) => { logger.error("SocketService: CRITICAL - Error quitting pubClient in outer catch during init failure. Potential resource leak.", { error: e }); return; }));
                }
                if (subClient) { // subClient is from the IIFE's scope
                    cleanupPromises.push(subClient.quit().catch((e: Error) => { logger.error("SocketService: CRITICAL - Error quitting subClient in outer catch during init failure. Potential resource leak.", { error: e }); return; }));
                }

                if (cleanupPromises.length > 0) {
                    // Fire-and-forget cleanup efforts, logging outcomes.
                    Promise.all(cleanupPromises)
                        .then(() => logger.info("SocketService: Attempted cleanup of Redis clients in outer catch due to init failure."))
                        .catch(e => logger.error("SocketService: Error during Redis client cleanup in outer catch.", { error: e }));
                }

                SocketService.state = "uninitialized"; // Reset state on failure
                SocketService.instance = null; // Clear instance on failure
                SocketService.initializationPromise = null; // Clear promise to allow retry

                // If init fails, the SIGTERM handler that was set up for *this attempt's Redis part*
                // (if Redis setup was successful before another part failed) should ideally remain
                // to allow cleanup if the process terminates before a retry.
                // The next call to init() will clean up this currentSigtermHandler if it's still set (at the start of its IIFE).
                // If Redis setup itself failed, no new handler was set by this attempt, and any old one was cleared at IIFE start.
                throw error; // Re-throw to reject the initializationPromise
            }
        })();

        return SocketService.initializationPromise;
    }

    /**
     * Get the SocketService instance. Throws an error if not initialized.
     * @throws Error if the service has not been initialized via initialize().
     */
    public static get(): SocketService {
        if (SocketService.state !== "initialized" || !SocketService.instance) {
            // Throw error if not fully initialized or instance is missing
            throw new Error(`SocketService not ready or not initialized. Current state: ${SocketService.state}. Ensure SocketService.init() is called and awaited at application startup.`);
        }
        return SocketService.instance;
    }

    /**
     * Adds a connected socket's ID to the local user and session maps for *this server instance*.
     * This method is invoked when a socket establishes a connection with the current server instance.
     * The Redis adapter ensures that room memberships are synchronized across the cluster when
     * sockets join or leave rooms, but these `userSockets` and `sessionSockets` maps remain local.
     *
     * @param socket The connected socket instance.
     */
    public addSocket(socket: Socket): void {
        const user = SessionService.getUser(socket);
        const userId = user?.id;
        const sessionId = user?.session.id;

        if (userId && sessionId) {
            if (!this.userSockets.has(userId)) {
                this.userSockets.set(userId, new Set<string>());
            }
            this.userSockets.get(userId)?.add(socket.id);

            if (!this.sessionSockets.has(sessionId)) {
                this.sessionSockets.set(sessionId, new Set<string>());
            }
            this.sessionSockets.get(sessionId)?.add(socket.id);

            logger.debug(`Socket ${socket.id} added for user ${userId}, session ${sessionId}`);
        }
    }

    /**
     * Removes a disconnected socket's ID from the local user and session maps for *this server instance*.
     * This is called when a socket disconnects from the current server instance.
     * The Redis adapter ensures that the socket's departure from any rooms is communicated
     * across the cluster.
     *
     * @param socket The disconnected socket instance.
     */
    public removeSocket(socket: Socket): void {
        const user = SessionService.getUser(socket); // Potentially get user/session info if needed, or maybe it's stored elsewhere
        const userId = user?.id;
        const sessionId = user?.session.id;

        if (userId) {
            const userSet = this.userSockets.get(userId);
            if (userSet) {
                userSet.delete(socket.id);
                if (userSet.size === 0) {
                    this.userSockets.delete(userId);
                }
            }
        }

        if (sessionId) {
            const sessionSet = this.sessionSockets.get(sessionId);
            if (sessionSet) {
                sessionSet.delete(socket.id);
                if (sessionSet.size === 0) {
                    this.sessionSockets.delete(sessionId);
                }
            }
        }
        logger.debug(`Socket ${socket.id} removed for user ${userId}, session ${sessionId}`);
    }

    /**
     * Emits a socket event to all clients in a specific room across all server instances.
     * This method leverages the Redis adapter to achieve cluster-wide emission.
     *
     * How it works with the Redis Adapter:
     * 1. `this.io.in(roomId).fetchSockets()`: This command, when an adapter is configured,
     *    sends a request (via Redis) to all server instances to get a list of all socket IDs
     *    present in the specified `roomId` across the entire cluster.
     * 2. The method then iterates through these (potentially remote) sockets.
     * 3. `socket.emit(event, payload)`: For each socket, this emit is routed by the adapter
     *    (again, via Redis) to the specific server instance that is currently managing that
     *    socket's connection. That instance then delivers the event to the client.
     *
     * The per-socket check for `AuthTokensService.isAccessTokenExpired` is performed before emitting.
     * If a more direct broadcast without per-socket fetching is desired (and per-socket checks
     * are handled differently), `this.io.to(roomId).emit(event, payload)` could be used, relying
     * entirely on the adapter's optimized broadcast.
     * 
     * @param event The custom socket event to emit.
     * @param roomId The ID of the room (e.g. chat) to emit the event to.
     * @param payload The payload data to send along with the event.
     */
    public emitSocketEvent<T extends EmitSocketEvent>(event: T, roomId: string, payload: SocketEventPayloads[T]): void {
        this.io.in(roomId).fetchSockets().then((sockets) => {
            for (const socket of sockets) {
                const session = (socket as { session?: SessionData }).session;
                if (!session) {
                    socket.emit(event, payload);
                    continue; // Continue to next socket if no session
                }
                const isExpired = AuthTokensService.isAccessTokenExpired(session);
                if (isExpired) {
                    socket.disconnect();
                    // Also remove the socket from session socket maps
                    // No need to call removeSocket here as the disconnect event will trigger it
                } else {
                    socket.emit(event, payload);
                }
            }
        }).catch(err => {
            logger.error("Error fetching sockets for emit", { event, roomId, error: err, trace: "SOCKET_EMIT_FETCH_ERR" });
        });
    }

    /**
     * Registers a socket event listener on a specific socket connected *to this server instance*.
     * Event handlers registered this way will only be triggered by events occurring on sockets
     * directly managed by the current server instance. This behavior is local and not directly
     * altered by the Redis adapter, as it pertains to individual socket object listeners.
     * 
     * @param socket - The socket object (must be a local socket).
     * @param event - The socket event to listen for.
     * @param handler - The event handler function.
     */
    public onSocketEvent<T extends OnSocketEvent>(socket: Socket, event: T, handler: SocketEventHandler<T>): void {
        socket.on(event, handler as never);
    }

    /**
     * Checks if a specific room has any open connections *across all server instances*.
     * With the Redis adapter configured, `this.io.sockets.adapter.rooms.get(roomId)`
     * queries the cluster-wide state of the room. The `.size` property will reflect
     * the total number of sockets in that room across the entire cluster.
     *
     * This is suitable for checks like determining if a push notification is needed
     * versus a WebSocket event if any client is actively connected to the room anywhere.
     *
     * @param roomId - The ID of the room to check.
     * @returns true if the room has one or more open connections cluster-wide, false otherwise.
     */
    public roomHasOpenConnections(roomId: string): boolean {
        const room = this.io.sockets.adapter.rooms.get(roomId);
        return room ? room.size > 0 : false;
    }

    /**
     * Closes all socket connections for a specific user that are connected *to this server instance*.
     * This is the local execution part of a global disconnection command.
     * Useful when the user logs out or is banned, to clear their local connections.
     *
     * Scope: This method operates on sockets stored in the local `this.userSockets` map.
     * It will only disconnect sockets directly managed by the server instance where this method is called.
     * When `socket.disconnect(true)` is executed, the Redis adapter ensures this disconnection
     * is recognized cluster-wide (e.g., the socket is removed from any rooms it was in across the cluster).
     *
     * @param userId The ID of the user whose local sockets should be closed.
     */
    private _performLocalUserSocketDisconnection(userId: string): void {
        const socketsIds = this.userSockets.get(userId);
        if (socketsIds) {
            logger.info(`[SocketService] Closing ${socketsIds.size} local sockets for user ${userId}`);
            for (const socketId of socketsIds) {
                const socket = this.io.sockets.sockets.get(socketId);
                if (socket && socket.connected) {
                    socket.disconnect(true); // Pass true to close the underlying connection
                }
            }
            // No need to delete from userSockets here, removeSocket on disconnect handles it
        }
    }

    /**
     * Initiates a cluster-wide command to close all socket connections for a specific user.
     * This method sends a server-side event that all instances listen for, prompting
     * them to close any connections they manage for the specified user.
     *
     * @param userId The ID of the user whose sockets should be closed across all instances.
     */
    public closeUserSockets(userId: string): void {
        if (!userId) {
            logger.warn("[SocketService] closeUserSockets called with null or undefined userId.");
            return;
        }
        logger.info(`[SocketService] Initiating global disconnection for user ${userId} via serverSideEmit.`);
        this.io.serverSideEmit("internal:disconnect_user_sockets", userId);
    }

    /**
     * Closes all socket connections for a specific session that are connected *to this server instance*.
     * This is the local execution part of a global disconnection command.
     * Useful when a session is revoked or expires, to clear its local connections.
     *
     * Scope: This method operates on sockets stored in the local `this.sessionSockets` map.
     * It will only disconnect sockets directly managed by the server instance where this method is called.
     * The Redis adapter ensures the disconnections are recognized cluster-wide.
     *
     * @param sessionId The ID of the session whose local sockets should be closed.
     */
    private _performLocalSessionSocketDisconnection(sessionId: string): void {
        const socketsIds = this.sessionSockets.get(sessionId);
        if (socketsIds) {
            logger.info(`[SocketService] Closing ${socketsIds.size} local sockets for session ${sessionId}`);
            for (const socketId of socketsIds) {
                const socket = this.io.sockets.sockets.get(socketId);
                if (socket && socket.connected) {
                    socket.disconnect(true); // Pass true to close the underlying connection
                }
            }
            // No need to delete from sessionSockets here, removeSocket on disconnect handles it
        }
    }

    /**
     * Initiates a cluster-wide command to close all socket connections for a specific session.
     * This method sends a server-side event that all instances listen for, prompting
     * them to close any connections they manage for the specified session.
     *
     * @param sessionId The ID of the session whose sockets should be closed across all instances.
     */
    public closeSessionSockets(sessionId: string): void {
        if (!sessionId) {
            logger.warn("[SocketService] closeSessionSockets called with null or undefined sessionId.");
            return;
        }
        logger.info(`[SocketService] Initiating global disconnection for session ${sessionId} via serverSideEmit.`);
        this.io.serverSideEmit("internal:disconnect_session_sockets", sessionId);
    }

    /**
     * Retrieves health details about the socket server.
     * Note: Some metrics are instance-specific, while others can reflect cluster state
     * when a Redis adapter is in use.
     *
     * - `connectedClients` (`this.io.engine.clientsCount`): This typically reflects the number
     *   of clients connected *directly to this server instance*.
     *   To get a cluster-wide client count, adapter-specific methods or a custom aggregation
     *   strategy (e.g., using `io.serverSideEmit` for each server to report its count)
     *   would be required.
     * - `activeRooms` (`this.io.sockets.adapter.rooms?.size`): With the Redis adapter, this
     *   reflects the total number of unique active rooms *across the entire cluster*.
     * - `namespaces` (`this.io.sockets.adapter.nsp?.size`): This usually refers to the
     *   number of configured namespaces, which is typically static and consistent across instances.
     *
     * @returns An object containing health details, or null if the server is not ready.
     */
    public getHealthDetails(): SocketHealthDetails | null {
        // Check if io and its properties are available (robustness check)
        if (!this.io?.engine || !this.io?.sockets?.adapter) {
            logger.warn("Attempted to get socket health details before io was fully ready.");
            return null;
        }

        try {
            return {
                connectedClients: this.io.engine.clientsCount ?? 0,
                activeRooms: this.io.sockets.adapter.rooms?.size ?? 0,
                namespaces: this.io.sockets.adapter.nsp?.size ?? 0,
            };
        } catch (error) {
            logger.error("Error retrieving socket health details", { error, trace: "SOCKET_HEALTH_DETAILS_ERR" });
            return null; // Return null on error to indicate issue
        }
    }
}
