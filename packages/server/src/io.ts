import { Server } from "socket.io";
import { server } from "./server";
import { safeOrigins } from "./utils";

// Create the WebSocket server and attach it to the HTTP server
export const io = new Server(server, {
    // Requires its own cors settings
    cors: {
        origin: safeOrigins(),
        methods: ["GET", "POST"],
        credentials: true,
    },
});
