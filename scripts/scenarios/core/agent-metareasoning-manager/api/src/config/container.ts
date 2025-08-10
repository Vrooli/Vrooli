import 'reflect-metadata';
import { container } from 'tsyringe';
import { Pool } from 'pg';
import { RedisClientType } from 'redis';
import { getDbPool, getRedisClient } from './database.js';

// Import services (will be created next)
import { PromptsService } from '../services/prompts.service.js';
import { WorkflowsService } from '../services/workflows.service.js';
import { AnalysisService } from '../services/analysis.service.js';
import { TemplatesService } from '../services/templates.service.js';
import { OrchestrationService } from '../services/orchestration.service.js';

// Import repositories (will be created next)
import { PromptsRepository } from '../repositories/prompts.repository.js';
import { TemplatesRepository } from '../repositories/templates.repository.js';
import { WorkflowsRepository } from '../repositories/workflows.repository.js';

/**
 * Configure dependency injection container
 */
export async function configureContainer(): Promise<void> {
  // Register database connections
  container.register<Pool>('DbPool', {
    useFactory: () => getDbPool()
  });

  container.register<Promise<RedisClientType>>('RedisClient', {
    useFactory: () => getRedisClient()
  });

  // Register repositories
  container.registerSingleton<PromptsRepository>(PromptsRepository);
  container.registerSingleton<TemplatesRepository>(TemplatesRepository);
  container.registerSingleton<WorkflowsRepository>(WorkflowsRepository);

  // Register services
  container.registerSingleton<PromptsService>(PromptsService);
  container.registerSingleton<WorkflowsService>(WorkflowsService);
  container.registerSingleton<AnalysisService>(AnalysisService);
  container.registerSingleton<TemplatesService>(TemplatesService);
  container.registerSingleton<OrchestrationService>(OrchestrationService);
}

export { container };