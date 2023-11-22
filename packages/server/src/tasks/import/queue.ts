import Bull from "bull";
import { HOST, PORT } from "../../redisConn.js";
import { importProcess } from "./process.js";

export type ImportProcessPayload = {

}

const importQueue = new Bull<ImportProcessPayload>("import", { redis: { port: PORT, host: HOST } });
importQueue.process(importProcess);

export function importData(data: ImportProcessPayload) {
    importQueue.add(data); //TODO
}
