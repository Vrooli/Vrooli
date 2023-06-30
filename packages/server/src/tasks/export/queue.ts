import Bull from "bull";
import { HOST, PORT } from "../../redisConn.js";
import { exportProcess } from "./process.js";

const exportQueue = new Bull("export", { redis: { port: PORT, host: HOST } });
exportQueue.process(exportProcess);

export function exporttData(data: any) {
    exportQueue.add(data); //TODO
}
