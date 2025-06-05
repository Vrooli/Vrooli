import { type ReservedSocketEvents, type RoomSocketEvents, type SocketEvent, type SocketEventCallbackPayloads, type SocketEventPayloads } from "@vrooli/shared";
import i18next from "i18next";
import { io } from "socket.io-client";
import { webSocketUrlBase } from "../utils/consts.js";
import { PubSub } from "../utils/pubsub.js";

const shouldUseWebSocket = process.env.NODE_ENV !== "test";
export const socket = io(webSocketUrlBase, {
    withCredentials: true,
    autoConnect: shouldUseWebSocket,
    reconnection: shouldUseWebSocket,
});

export const SERVER_CONNECT_MESSAGE_ID = "AuthMessage";

type EmitSocketEvent = RoomSocketEvents;
type OnSocketEvent = Exclude<SocketEvent, ReservedSocketEvents | RoomSocketEvents>;
enum SocketState {
    Connected = "Connected",
    DisconnectedNoError = "DisconnectedNoError",
    LostConnection = "LostConnection",
}

export class SocketService {
    private static instance: SocketService;

    state = SocketState.DisconnectedNoError;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }
    static get(): SocketService {
        if (!SocketService.instance) {
            SocketService.instance = new SocketService();
        }
        return SocketService.instance;
    }

    handleConnected() {
        console.info("Websocket connected to server");
        PubSub.get().publish("clearSnack", { id: SERVER_CONNECT_MESSAGE_ID });
        if (this.state === SocketState.LostConnection) {
            PubSub.get().publish("snack", { message: i18next.t("ServerReconnected"), severity: "Success" });
        }
        this.state = SocketState.Connected;
    }

    handleDisconnected() {
        if (this.state !== SocketState.Connected) {
            return;
        }
        this.state = SocketState.LostConnection;
        console.info("Websocket disconnected from server");
        PubSub.get().publish("snack", { message: i18next.t("ServerDisconnected"), severity: "Error", id: SERVER_CONNECT_MESSAGE_ID, autoHideDuration: "persist" });
    }

    handleReconnectAttempted() {
        console.info("Websocket attempting to reconnect to server");
        PubSub.get().publish("snack", { message: i18next.t("ServerReconnectAttempt"), severity: "Warning", id: SERVER_CONNECT_MESSAGE_ID, autoHideDuration: "persist" });
    }

    handleReconnected() {
        this.state = SocketState.Connected;
        console.info("Websocket reconnected to server");
        PubSub.get().publish("snack", { message: i18next.t("ServerReconnected"), severity: "Success", id: SERVER_CONNECT_MESSAGE_ID });
    }

    /**
     * Emits a socket event from the client to the server.
     * 
     * @param event - The socket event to emit.
     * @param payload - The payload data to send along with the event.
     * @param callback - An optional callback function to handle the response.
     */
    emitEvent<T extends EmitSocketEvent>(
        event: T,
        payload: SocketEventPayloads[T],
        callback?: (response: SocketEventCallbackPayloads[T]) => void,
    ) {
        if (callback) {
            socket.emit(event, payload, callback);
        } else {
            socket.emit(event, payload);
        }
    }

    /**
     * Registers a socket event listener.
     * 
     * @param socket - The socket object.
     * @param event - The socket event to listen for.
     * @param handler - The event handler function.
     */
    onEvent<T extends OnSocketEvent>(
        event: T,
        handler: (payload: SocketEventPayloads[T]) => unknown,
    ) {
        socket.on(event, handler as never);

        return () => {
            socket.off(event, handler as never);
        };
    }

    disconnect() {
        this.state = SocketState.DisconnectedNoError;
        socket.disconnect();
    }

    connect() {
        socket.connect();
    }

    restart() {
        this.disconnect();
        this.connect();
    }
}
