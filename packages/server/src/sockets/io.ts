import { Server } from "socket.io";
import { RequestService } from "../auth/request.js";
import { server } from "../server.js";

/**
 * Active socket IDs by user ID
 */
export const userSockets = new Map<string, Set<string>>();

/**
 * Active socket IDs by session ID
 */
export const sessionSockets = new Map<string, Set<string>>();

// Create the WebSocket server and attach it to the HTTP server
export const io = new Server(server, {
    // Requires its own cors settings
    cors: {
        origin: RequestService.get().safeOrigins(),
        methods: ["GET", "POST"],
        credentials: true,
    },
});
