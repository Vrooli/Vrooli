import Bull from 'bull';
import { smsProcess } from './process';

const smsQueue = new Bull('sms', { redis: { 
    port: process.env.REDIS_CONN.split(':')[1],
    host: process.env.REDIS_CONN.split(':')[0]
} });
smsQueue.process(smsProcess);

export function sendSms(to=[], body) {
    smsQueue.add({
        to: to,
        body: body
    });
}