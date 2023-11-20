import Bull from "bull";
import { HOST, PORT } from "../../redisConn.js";
import { smsProcess } from "./process.js";

export type SmsProcessPayload = {
    to: string[];
    body: string;
}

const smsQueue = new Bull<SmsProcessPayload>("sms", { redis: { port: PORT, host: HOST } });
smsQueue.process(smsProcess);

export function sendSms(to = [], body: string) {
    smsQueue.add({ to, body });
}
