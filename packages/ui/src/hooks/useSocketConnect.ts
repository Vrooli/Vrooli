import { useEffect } from "react";
import { SocketService, socket } from "../api/socket.js";

export function useSocketConnect() {
    useEffect(() => {
        // Instead of using SocketService functions directly, we have to 
        // wrap them in closures to ensure that the SocketService instance is updated when calling them
        const events = [
            ["connect", () => { SocketService.get().handleConnected(); }],
            ["disconnect", () => { SocketService.get().handleDisconnected(); }],
            ["reconnect_attempt", () => { SocketService.get().handleReconnectAttempted(); }],
            ["reconnect", () => { SocketService.get().handleReconnected(); }],
        ] as const;

        events.forEach(([event, handler]) => { socket.on(event, handler); });

        return () => {
            events.forEach(([event, handler]) => { socket.off(event, handler); });
            socket.disconnect();
        };
    }, []);
}
