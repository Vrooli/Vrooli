import Bull from "bull";
import { HOST, PORT } from "../../redisConn.js";
import { pushProcess } from "./process.js";
const pushQueue = new Bull("push", { redis: { port: PORT, host: HOST } });
pushQueue.process(pushProcess);
export function sendPush(subscription, payload, delay = 0) {
    pushQueue.add({
        ...payload,
        ...subscription,
    }, { delay });
}
//# sourceMappingURL=queue.js.map