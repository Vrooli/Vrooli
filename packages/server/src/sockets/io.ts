import { ReservedSocketEvents, RoomSocketEvents, SocketEvent, SocketEventHandler, SocketEventPayloads } from "@local/shared";
import { createAdapter } from "@socket.io/redis-adapter";
import { createClient } from "redis";
import { Server, Socket } from "socket.io";
import { AuthTokensService } from "../auth/auth.js";
import { RequestService } from "../auth/request.js";
import { SessionService } from "../auth/session.js";
import { logger } from "../events/logger.js";
import { getRedisUrl } from "../redisConn.js";
import { server } from "../server.js";
import { SessionData } from "../types.js";

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

/**
 * Singleton class to manage socket.io instance and socket maps.
 *
 * Horizontal Scaling:
 * This service supports horizontal scaling across multiple server instances
 * through the use of the `@socket.io/redis-adapter`. When configured, this
 * adapter uses Redis Pub/Sub capabilities to broadcast events and manage
 * socket states (like room memberships) across all connected server instances.
 * This allows a client connected to one server instance to seamlessly
 * receive messages or interact with clients connected to other instances
 * as if they were all on a single server.
 *
 * Key aspects of scaling handled by the adapter:
 * - Broadcasting events to all clients in a room, regardless of which instance they are connected to.
 * - Fetching all sockets in a room across all instances.
 * - Handling disconnections in a way that is recognized cluster-wide.
 *
 * Note: Local maps like `userSockets` and `sessionSockets` still only contain
 * sockets directly connected to *this* specific server instance. The adapter
 * orchestrates communication between instances, not by merging these local maps directly.
 */
export class SocketService {
    private static instance: SocketService | null = null;
    private static state: SocketServiceState = "uninitialized";

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
     * Must be called once during application startup.
     */
    public static async init(): Promise<void> {
        if (SocketService.state === "initialized") {
            logger.warn("SocketService already initialized.");
            return;
        }
        if (SocketService.state === "initializing") {
            logger.warn("SocketService initialization already in progress.");
            // Optional: Could implement a mechanism to wait for the ongoing initialization
            // For now, just return to prevent concurrent initializations
            return;
        }

        SocketService.state = "initializing";
        try {
            // Create the instance internally
            const instance = new SocketService();

            // Initialize properties
            instance.userSockets = new Map<string, Set<string>>();
            instance.sessionSockets = new Map<string, Set<string>>();

            // Get RequestService info
            const requestService = RequestService.get();
            const safeOrigins = requestService.safeOrigins();

            // --- Redis Adapter Setup (for horizontal scaling) ---
            let redisAdapter;
            try {
                // Connect to new Redis client for publishing messages.
                const pubClient = createClient({ url: getRedisUrl() });

                // Duplicate the publisher client for subscribing.
                // Socket.IO Redis adapter requires separate clients for pub and sub operations
                // to avoid Redis Pub/Sub commands from interfering with regular commands on the same connection.
                const subClient = pubClient.duplicate();

                // Connect both clients to the Redis server.
                // Promise.all ensures both connections are established before proceeding.
                await Promise.all([pubClient.connect(), subClient.connect()]);

                // Create the Redis adapter instance, passing in the pub and sub clients.
                // This adapter will now manage inter-server communication.
                redisAdapter = createAdapter(pubClient, subClient);
                logger.info("SocketService: Redis adapter created and clients connected.");

                // TODO: Implement graceful shutdown for pubClient and subClient
                // For example, by adding a close method to SocketService:
                // instance.closeRedisClients = async () => {
                //   await pubClient.quit();
                //   await subClient.quit();
                //   logger.info("SocketService: Redis clients disconnected.");
                // };
                // And calling this during application shutdown.

            } catch (redisError) {
                logger.error("SocketService: Failed to connect Redis clients or create adapter. Socket.IO will run with in-memory adapter.", { error: redisError, trace: "SOCKET_REDIS_ADAPTER_FAIL" });
                // Proceeding without the adapter means it defaults to in-memory, single-instance mode.
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
            // Listener for global user socket disconnection
            instance.io.on("internal:disconnect_user_sockets", (userId: string) => {
                logger.info(`[SocketService] Received command via serverSideEmit to disconnect sockets for user: ${userId} on this instance.`);
                // Ensure 'this' context is correct or use the instance directly if 'this' is problematic in the callback
                // In this case, 'instance' is captured in the closure and is the correct SocketService instance.
                instance._performLocalUserSocketDisconnection(userId);
            });

            // Listener for global session socket disconnection
            instance.io.on("internal:disconnect_session_sockets", (sessionId: string) => {
                logger.info(`[SocketService] Received command via serverSideEmit to disconnect sockets for session: ${sessionId} on this instance.`);
                instance._performLocalSessionSocketDisconnection(sessionId);
            });
            // --- End Server-Side Event Listeners ---

            // Assign the fully initialized instance
            SocketService.instance = instance;
            SocketService.state = "initialized"; // Mark as initialized *after* all setup is done
            logger.info("SocketService initialized successfully.");

        } catch (error) {
            logger.error("Failed to initialize SocketService", { error, trace: "SOCKET_INIT_FAIL" });
            // Reset state on failure to allow potential retry? Or keep as initializing?
            // Let's reset to uninitialized for simplicity, although a 'failed' state might be better.
            SocketService.state = "uninitialized";
            throw error; // Re-throw to signal failure
        }
    }

    /**
     * Get the SocketService instance. Throws an error if not initialized.
     * @throws Error if the service has not been initialized via initialize().
     */
    public static get(): SocketService {
        if (SocketService.state !== "initialized" || !SocketService.instance) {
            // Throw error if not fully initialized or instance is missing
            throw new Error(`SocketService not ready. Current state: ${SocketService.state}. Call SocketService.init() during startup.`);
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
