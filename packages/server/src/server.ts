import http from "http";
import { app } from "./app.js";

const SERVER_PORT = "5329";
export const SERVER_URL_LOCAL = `http://localhost:${SERVER_PORT}/api`;
export const SERVER_URL_REMOTE = process.env.SERVER_URL ?? `http://${process.env.SITE_IP}:${SERVER_PORT}/api`;
export const SERVER_URL = process.env.VITE_SERVER_LOCATION === "local" ? SERVER_URL_LOCAL : SERVER_URL_REMOTE;

const UI_PORT = "3000";
export const UI_URL_LOCAL = `http://localhost:${UI_PORT}`;
export const UI_URL_REMOTE = process.env.UI_URL ?
    process.env.UI_URL :
    process.env.SERVER_URL ?
        new URL(SERVER_URL).origin :
        `http://${process.env.SITE_IP}:${UI_PORT}`;
export const UI_URL = process.env.VITE_SERVER_LOCATION === "local" ? UI_URL_LOCAL : UI_URL_REMOTE;

export const server = http.createServer(app);
