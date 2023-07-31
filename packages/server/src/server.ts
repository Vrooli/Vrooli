import http from "http";
import { app } from "./app";

export const SERVER_URL = process.env.VITE_SERVER_LOCATION === "local" ?
    "http://localhost:5329/api" :
    process.env.SERVER_URL ?
        process.env.SERVER_URL :
        `http://${process.env.SITE_IP}:5329/api`;

export const server = http.createServer(app);
