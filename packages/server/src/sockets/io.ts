import { ReservedSocketEvents, RoomSocketEvents, SocketEvent, SocketEventHandler, SocketEventPayloads } from "@local/shared";
import { Server, Socket } from "socket.io";
import { AuthTokensService } from "../auth/auth.js";
import { RequestService } from "../auth/request.js";
import { SessionService } from "../auth/session.js";
import { logger } from "../events/logger.js";
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
 * Singleton class to manage socket.io instance and socket maps
 */
export class SocketService {
    private static instance: SocketService | null = null;
    private static state: SocketServiceState = "uninitialized";

    /**
     * Active socket IDs by user ID
     */
    public userSockets!: Map<string, Set<string>>;

    /**
     * Active socket IDs by session ID
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

            // Create the Server instance with the required CORS config
            instance.io = new Server(server, {
                cors: {
                    origin: safeOrigins,
                    methods: ["GET", "POST"],
                    credentials: true,
                },
            });

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
     * Adds a connected socket's ID to the user and session maps.
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
     * Removes a disconnected socket's ID from the user and session maps.
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
     * Emits a socket event to all clients in a specific room.
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
     * Registers a socket event listener on a specific socket.
     * 
     * @param socket - The socket object.
     * @param event - The socket event to listen for.
     * @param handler - The event handler function.
     */
    public onSocketEvent<T extends OnSocketEvent>(socket: Socket, event: T, handler: SocketEventHandler<T>): void {
        socket.on(event, handler as never);
    }

    /**
     * Checks if a specific room has open connections.
     *
     * @param roomId - The ID of the room to check.
     * @returns true if the room has one or more open connections, false otherwise.
     */
    public roomHasOpenConnections(roomId: string): boolean {
        const room = this.io.sockets.adapter.rooms.get(roomId);
        return room ? room.size > 0 : false;
    }

    /**
     * Closes all socket connections for a specific user.
     * Useful when the user logs out or is banned.
     * @param userId The ID of the user whose sockets should be closed.
     */
    public closeUserSockets(userId: string): void {
        const socketsIds = this.userSockets.get(userId);
        if (socketsIds) {
            logger.info(`Closing ${socketsIds.size} sockets for user ${userId}`);
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
     * Closes all socket connections for a specific session.
     * Useful when the user logs out or revokes open sessions.
     * @param sessionId The ID of the session whose sockets should be closed.
     */
    public closeSessionSockets(sessionId: string): void {
        const socketsIds = this.sessionSockets.get(sessionId);
        if (socketsIds) {
            logger.info(`Closing ${socketsIds.size} sockets for session ${sessionId}`);
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
     * Retrieves health details about the socket server.
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
