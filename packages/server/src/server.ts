import http from "http";
import { app } from "./app";

export const server = http.createServer(app);
