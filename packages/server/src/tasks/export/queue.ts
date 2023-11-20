import Bull from "bull";
import { HOST, PORT } from "../../redisConn.js";
import { exportProcess } from "./process.js";

export type ExportProcessPayload = {

}

const exportQueue = new Bull<ExportProcessPayload>("export", { redis: { port: PORT, host: HOST } });
exportQueue.process(exportProcess);

export function exportData(data: ExportProcessPayload) {
    exportQueue.add(data); //TODO
}
