import Bull from 'bull';
import { smsProcess } from './process.js';
import { HOST, PORT } from '../../redisConn';

const smsQueue = new Bull('email', { redis: { port: PORT, host: HOST } });
smsQueue.process(smsProcess);

export function sendSms(to=[], body: string) {
    smsQueue.add({
        to: to,
        body: body
    });
}