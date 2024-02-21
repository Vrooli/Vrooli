import { SocketEvent, SocketEventPayloads } from "@local/shared";
import { io } from "socket.io-client";
import { webSocketUrlBase } from "./fetchData";

export const socket = io(webSocketUrlBase, { withCredentials: true });

type ReservedEventNames = "connect" | "connect_error" | "disconnect";
type CustomSocketEvent = Exclude<SocketEvent, ReservedEventNames>;

export const listenEvent = <T extends CustomSocketEvent>(
    event: T,
    handler: (payload: SocketEventPayloads[T]) => void,
) => {
    socket.on(event, handler as any);
};
