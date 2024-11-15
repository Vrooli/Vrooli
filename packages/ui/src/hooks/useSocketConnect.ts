import { socket } from "api/socket";
import { useEffect } from "react";
import { PubSub } from "utils/pubsub";

export const SERVER_CONNECT_MESSAGE_ID = "AuthMessage";

let wasDisconnected = false;

export function useSocketConnect() {
    useEffect(() => {
        const events = [
            ["connect", () => {
                console.info("Websocket connected to server");
                PubSub.get().publish("clearSnack", { id: SERVER_CONNECT_MESSAGE_ID });
                if (wasDisconnected) {
                    PubSub.get().publish("snack", { messageKey: "ServerReconnected", severity: "Success" });
                }
            }],
            ["disconnect", () => {
                wasDisconnected = true;
                console.info("Websocket disconnected from server");
                PubSub.get().publish("snack", { messageKey: "ServerDisconnected", severity: "Error", id: SERVER_CONNECT_MESSAGE_ID, autoHideDuration: "persist" });
            }],
            ["reconnect_attempt", () => {
                console.info("Websocket attempting to reconnect to server");
                PubSub.get().publish("snack", { messageKey: "ServerReconnectAttempt", severity: "Warning", id: SERVER_CONNECT_MESSAGE_ID, autoHideDuration: "persist" });
            }],
            ["reconnect", () => {
                console.info("Websocket reconnected to server");
                PubSub.get().publish("snack", { messageKey: "ServerReconnected", severity: "Success", id: SERVER_CONNECT_MESSAGE_ID });
            }],
        ] as const;

        events.forEach(([event, handler]) => { socket.on(event, handler); });

        return () => {
            events.forEach(([event, handler]) => { socket.off(event, handler); });
            socket.disconnect();
        };
    }, []);
}
