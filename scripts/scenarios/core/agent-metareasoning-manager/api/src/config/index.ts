import { z } from 'zod';

// Environment variable schema
const envSchema = z.object({
  NODE_ENV: z.enum(['development', 'production', 'test']).default('development'),
  METAREASONING_API_PORT: z.string().transform(Number).default('8093'),
  METAREASONING_API_HOST: z.string().default('localhost'),
  
  // CORS
  CORS_ORIGINS: z.string().optional().transform(val => 
    val ? val.split(',') : ['http://localhost:3000', 'http://localhost:8001']
  ),
  
  // Database
  POSTGRES_HOST: z.string().default('localhost'),
  POSTGRES_PORT: z.string().transform(Number).default('5432'),
  POSTGRES_DB: z.string().default('postgres'),
  POSTGRES_USER: z.string().default('postgres'),
  POSTGRES_PASSWORD: z.string().default(''),
  
  // Redis
  REDIS_HOST: z.string().default('localhost'),
  REDIS_PORT: z.string().transform(Number).default('6379'),
  
  // n8n
  N8N_BASE_URL: z.string().optional(),
  N8N_PORT: z.string().transform(Number).default('5678'),
  N8N_WEBHOOK_BASE: z.string().optional(),
  
  // Windmill
  WINDMILL_BASE_URL: z.string().optional(),
  WINDMILL_PORT: z.string().transform(Number).default('8000'),
  WINDMILL_WORKSPACE: z.string().default('demo'),
  
  // Security
  JWT_SECRET: z.string().optional(),
  API_KEY_SALT: z.string().optional()
});

// Parse and validate environment variables
const env = envSchema.parse(process.env);

// Construct configuration object
export const config = {
  env: env.NODE_ENV,
  port: env.METAREASONING_API_PORT,
  host: env.METAREASONING_API_HOST,
  
  cors: {
    origin: env.CORS_ORIGINS,
    credentials: true
  },
  
  database: {
    host: env.POSTGRES_HOST,
    port: env.POSTGRES_PORT,
    database: env.POSTGRES_DB,
    user: env.POSTGRES_USER,
    password: env.POSTGRES_PASSWORD,
    max: 20,
    idleTimeoutMillis: 30000
  },
  
  redis: {
    url: `redis://${env.REDIS_HOST}:${env.REDIS_PORT}`,
    keyPrefix: 'metareasoning:',
    retryDelayOnFailover: 100,
    socket: {
      connectTimeout: 5000,
      lazyConnect: true
    }
  },
  
  n8n: {
    baseUrl: env.N8N_BASE_URL || `http://localhost:${env.N8N_PORT}`,
    webhookBase: env.N8N_WEBHOOK_BASE || `http://localhost:${env.N8N_PORT}/webhook`
  },
  
  windmill: {
    baseUrl: env.WINDMILL_BASE_URL || `http://localhost:${env.WINDMILL_PORT}`,
    workspace: env.WINDMILL_WORKSPACE
  },
  
  rateLimiting: {
    windowMs: 60 * 1000, // 1 minute
    max: 100, // requests per window
    message: 'Too many requests, please try again later'
  },
  
  security: {
    jwtSecret: env.JWT_SECRET || 'dev-secret-change-in-production',
    apiKeySalt: env.API_KEY_SALT || 'dev-salt-change-in-production'
  }
} as const;

export type Config = typeof config;