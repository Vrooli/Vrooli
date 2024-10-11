import { socket } from "api/socket";
import { useEffect } from "react";
import { PubSub } from "utils/pubsub";

export function useSocketConnect() {
    useEffect(() => {
        const events = [
            ["connect", () => {
                console.info("Websocket connected to server");
            }],
            ["disconnect", () => {
                PubSub.get().publish("snack", { messageKey: "ServerDisconnected", severity: "Error", id: "ServerDisconnected", autoHideDuration: 15000 });
            }],
            ["reconnect_attempt", () => {
                PubSub.get().publish("snack", { messageKey: "ServerReconnectAttempt", severity: "Warning", id: "ServerReconnectAttempt", autoHideDuration: 10000 });
            }],
            ["reconnect", () => {
                PubSub.get().publish("snack", { messageKey: "ServerReconnected", severity: "Success", id: "ServerReconnected" });
            }],
        ] as const;

        events.forEach(([event, handler]) => { socket.on(event, handler); });

        return () => {
            events.forEach(([event, handler]) => { socket.off(event, handler); });
            socket.disconnect();
        };
    }, []);
}
