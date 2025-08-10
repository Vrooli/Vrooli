import { Pool } from 'pg';
import { createClient, RedisClientType } from 'redis';
import { config } from './index.js';

// PostgreSQL connection pool
export const createDatabasePool = (): Pool => {
  return new Pool(config.database);
};

// Redis client factory
export const createRedisClient = async (): Promise<RedisClientType> => {
  const client = createClient({
    url: config.redis.url
  }) as RedisClientType;
  
  client.on('error', (err) => {
    console.error('Redis Client Error:', err);
  });
  
  client.on('connect', () => {
    console.log('Redis Client Connected');
  });
  
  await client.connect();
  return client;
};

// Export singleton instances
let dbPool: Pool | null = null;
let redisClient: RedisClientType | null = null;

export const getDbPool = (): Pool => {
  if (!dbPool) {
    dbPool = createDatabasePool();
  }
  return dbPool;
};

export const getRedisClient = async (): Promise<RedisClientType> => {
  if (!redisClient) {
    redisClient = await createRedisClient();
  }
  return redisClient;
};

export const closeDatabaseConnections = async (): Promise<void> => {
  if (dbPool) {
    await dbPool.end();
    dbPool = null;
  }
  if (redisClient) {
    await redisClient.quit();
    redisClient = null;
  }
};