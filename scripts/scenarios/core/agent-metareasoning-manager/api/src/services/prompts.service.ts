import { injectable, inject } from 'tsyringe';
import { v4 as uuidv4 } from 'uuid';
import { RedisClientType } from 'redis';
import { PromptsRepository, PromptEntity } from '../repositories/prompts.repository.js';
import { NotFoundError } from '../utils/errors.js';
import { logger } from '../utils/logger.js';
import { 
  CreatePromptRequest, 
  UpdatePromptRequest,
  TestPromptRequest 
} from '../schemas/prompts.schema.js';

@injectable()
export class PromptsService {
  private redis: RedisClientType | null = null;

  constructor(
    @inject(PromptsRepository) private promptsRepo: PromptsRepository,
    @inject('RedisClient') private redisPromise: Promise<RedisClientType>
  ) {
    this.initRedis();
  }

  private async initRedis(): Promise<void> {
    this.redis = await this.redisPromise;
  }

  /**
   * Get all prompts with optional filtering
   */
  async getPrompts(
    category?: string,
    limit: number = 20,
    offset: number = 0
  ): Promise<{ prompts: PromptEntity[]; total: number }> {
    const filters = category ? { category, status: 'active' } : { status: 'active' };
    const result = await this.promptsRepo.findPaginated(limit, offset, filters);
    
    return {
      prompts: result.data,
      total: result.total
    };
  }

  /**
   * Get a single prompt by ID
   */
  async getPromptById(id: string): Promise<PromptEntity> {
    // Try cache first
    if (this.redis) {
      const cached = await this.redis.get(`prompt:${id}`);
      if (cached) {
        logger.debug(`Cache hit for prompt ${id}`);
        return JSON.parse(cached);
      }
    }

    const prompt = await this.promptsRepo.findByIdOrThrow(id);
    
    // Cache for 5 minutes
    if (this.redis) {
      await this.redis.setEx(`prompt:${id}`, 300, JSON.stringify(prompt));
    }

    return prompt;
  }

  /**
   * Create a new prompt
   */
  async createPrompt(data: CreatePromptRequest['body']): Promise<PromptEntity> {
    const promptData = {
      id: uuidv4(),
      ...data,
      status: 'active' as const,
      version: 1,
      variables: data.variables || [],
      tags: data.tags || [],
      created_at: new Date(),
      updated_at: new Date()
    };

    const prompt = await this.promptsRepo.create(promptData);
    
    logger.info(`Created prompt ${prompt.id}: ${prompt.name}`);
    
    return prompt;
  }

  /**
   * Update an existing prompt
   */
  async updatePrompt(id: string, data: UpdatePromptRequest['body']): Promise<PromptEntity> {
    // Check if prompt exists
    await this.promptsRepo.findByIdOrThrow(id);

    // If template is being updated, increment version
    if (data.template) {
      await this.promptsRepo.incrementVersion(id);
    }

    const updated = await this.promptsRepo.update(id, data);
    
    // Invalidate cache
    if (this.redis) {
      await this.redis.del(`prompt:${id}`);
    }

    logger.info(`Updated prompt ${id}`);
    
    return updated;
  }

  /**
   * Delete a prompt (soft delete by setting status to archived)
   */
  async deletePrompt(id: string): Promise<void> {
    await this.promptsRepo.update(id, { status: 'archived' });
    
    // Invalidate cache
    if (this.redis) {
      await this.redis.del(`prompt:${id}`);
    }

    logger.info(`Archived prompt ${id}`);
  }

  /**
   * Test a prompt with provided variables
   */
  async testPrompt(id: string, data: TestPromptRequest['body']): Promise<any> {
    const prompt = await this.getPromptById(id);
    
    // Replace variables in template
    let result = prompt.template;
    for (const [key, value] of Object.entries(data.variables)) {
      result = result.replace(new RegExp(`{{${key}}}`, 'g'), value);
    }

    // In a real implementation, this would call the LLM
    // For now, return the processed template
    return {
      prompt_id: id,
      processed_template: result,
      variables_used: data.variables,
      context: data.context,
      // This would be the actual LLM response
      response: `Mock response for prompt: ${prompt.name}`
    };
  }

  /**
   * Search prompts by text
   */
  async searchPrompts(searchTerm: string): Promise<PromptEntity[]> {
    return await this.promptsRepo.search(searchTerm);
  }

  /**
   * Get prompts by category
   */
  async getPromptsByCategory(category: string): Promise<PromptEntity[]> {
    return await this.promptsRepo.findByCategory(category);
  }

  /**
   * Get prompts by tag
   */
  async getPromptsByTag(tag: string): Promise<PromptEntity[]> {
    return await this.promptsRepo.findByTag(tag);
  }
}