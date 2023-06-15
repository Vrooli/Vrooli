import { io } from "socket.io-client";
import { webSocketUrlBase } from "./fetchData";

export const socket = io(webSocketUrlBase, { withCredentials: true });
