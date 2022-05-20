import { genErrorCode, logger, LogLevel } from './logger';
import { createClient } from 'redis';

const split = (process.env.REDIS_CONN || 'redis:6379').split(':');
export const HOST = split[0];
export const PORT = Number(split[1]);
export const REDIS_URL = `redis://${HOST}:${PORT}`;

export const redisClient = createClient({ url: REDIS_URL });
redisClient.on('connect', () => {
    logger.log(LogLevel.info, '✅ Connected to Redis.', { code: genErrorCode('0213') });
});
redisClient.on('error', (error) => {
    logger.log(LogLevel.error, 'Error occured while connecting or accessing redis server', { code: genErrorCode('0002'), error });
});
redisClient.on('end', () => {
    logger.log(LogLevel.info, 'Redis client closed.', { code: genErrorCode('0208'), url: REDIS_URL });
});