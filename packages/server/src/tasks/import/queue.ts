import Bull from "bull";
import { HOST, PORT } from "../../redisConn.js";
import { importProcess } from "./process.js";

const importQueue = new Bull("import", { redis: { port: PORT, host: HOST } });
importQueue.process(importProcess);

export function importData(data: any) {
    importQueue.add(data); //TODO
}
