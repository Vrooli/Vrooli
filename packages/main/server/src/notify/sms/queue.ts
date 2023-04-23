import Bull from "bull";
import { HOST, PORT } from "../../redisConn.js";
import { smsProcess } from "./process.js";

const smsQueue = new Bull("sms", { redis: { port: PORT, host: HOST } });
smsQueue.process(smsProcess);

export function sendSms(to = [], body: string) {
    smsQueue.add({
        to,
        body,
    });
}
