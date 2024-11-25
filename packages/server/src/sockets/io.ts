import { Server } from "socket.io";
import { RequestService } from "../auth/request";
import { server } from "../server";

// Create the WebSocket server and attach it to the HTTP server
export const io = new Server(server, {
    // Requires its own cors settings
    cors: {
        origin: RequestService.get().safeOrigins(),
        methods: ["GET", "POST"],
        credentials: true,
    },
});
