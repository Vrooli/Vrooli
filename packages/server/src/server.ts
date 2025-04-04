import http from "http";
import { app } from "./app.js";

// Server URL
export const SERVER_PORT = 5329;
const SERVER_URL_LOCAL = `http://localhost:${SERVER_PORT}`;
const SERVER_URL_REMOTE = process.env.SERVER_URL ?? `http://${process.env.SITE_IP}:${SERVER_PORT}`;
export const SERVER_URL = process.env.VITE_SERVER_LOCATION === "local" ? SERVER_URL_LOCAL : SERVER_URL_REMOTE;

// UI URL
const UI_PORT = 3000;
const UI_URL_LOCAL = `http://localhost:${UI_PORT}`;
export const UI_URL_REMOTE = process.env.UI_URL ?
    process.env.UI_URL :
    process.env.SERVER_URL ?
        new URL(SERVER_URL).origin :
        `http://${process.env.SITE_IP}:${UI_PORT}`;
export const UI_URL = process.env.VITE_SERVER_LOCATION === "local" ? UI_URL_LOCAL : UI_URL_REMOTE;


// API URLs
export const API_URL_SLUG = "/api";
export const API_URL = `${SERVER_URL}${API_URL_SLUG}`;

// MCP URLs
export const MCP_SITE_WIDE_URL_SLUG = "/mcp";
export const MCP_SITE_WIDE_URL = `${SERVER_URL}${MCP_SITE_WIDE_URL_SLUG}`;
export const MCP_TOOL_URL_SLUG = `${MCP_SITE_WIDE_URL_SLUG}/tool`;
export const MCP_TOOL_URL = `${SERVER_URL}${MCP_TOOL_URL_SLUG}`;

export const server = http.createServer(app);
