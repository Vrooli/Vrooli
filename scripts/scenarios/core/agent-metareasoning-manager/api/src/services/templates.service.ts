import { injectable, inject } from 'tsyringe';
import { v4 as uuidv4 } from 'uuid';
import { TemplatesRepository, TemplateEntity } from '../repositories/templates.repository.js';
import { OrchestrationService } from './orchestration.service.js';
import { logger } from '../utils/logger.js';
import {
  CreateTemplateRequest,
  ApplyTemplateRequest,
  TemplateResult
} from '../schemas/templates.schema.js';

@injectable()
export class TemplatesService {
  constructor(
    @inject(TemplatesRepository) private templatesRepo: TemplatesRepository,
    @inject(OrchestrationService) private orchestration: OrchestrationService
  ) {}

  /**
   * Get all templates with optional filtering
   */
  async getTemplates(
    category?: string,
    search?: string,
    limit: number = 20,
    offset: number = 0
  ): Promise<{ templates: TemplateEntity[]; total: number }> {
    if (search) {
      const templates = await this.templatesRepo.search(search);
      return {
        templates: templates.slice(offset, offset + limit),
        total: templates.length
      };
    }

    const filters = category ? { category } : {};
    const result = await this.templatesRepo.findPaginated(limit, offset, filters);
    
    return {
      templates: result.data,
      total: result.total
    };
  }

  /**
   * Get a single template by ID
   */
  async getTemplateById(id: string): Promise<TemplateEntity> {
    return await this.templatesRepo.findByIdOrThrow(id);
  }

  /**
   * Create a new template
   */
  async createTemplate(data: CreateTemplateRequest['body']): Promise<TemplateEntity> {
    const templateData = {
      id: uuidv4(),
      ...data,
      variables: data.variables || [],
      tags: data.tags || [],
      usage_count: 0,
      created_at: new Date(),
      updated_at: new Date()
    };

    const template = await this.templatesRepo.create(templateData);
    
    logger.info(`Created template ${template.id}: ${template.name}`);
    
    return template;
  }

  /**
   * Apply a template
   */
  async applyTemplate(
    id: string,
    data: ApplyTemplateRequest['body']
  ): Promise<TemplateResult> {
    const template = await this.getTemplateById(id);
    const executionId = uuidv4();
    const startTime = Date.now();

    // Log template application
    await this.templatesRepo.logApplication(
      id,
      executionId,
      data.context,
      data.variables
    );

    // Increment usage count
    await this.templatesRepo.incrementUsage(id);

    logger.info(`Applying template ${id}`, { executionId });

    try {
      // Execute template sequence
      const results = await this.orchestration.executeTemplate(
        template,
        data.context,
        data.variables,
        data.options
      );

      const totalDuration = Date.now() - startTime;
      const summary = this.generateSummary(template, results);

      // Update application result
      await this.templatesRepo.updateApplicationResult(
        executionId,
        'completed',
        results,
        summary
      );

      return {
        template_id: id,
        execution_id: executionId,
        status: 'completed',
        results,
        summary,
        total_duration_ms: totalDuration
      };
    } catch (error: any) {
      logger.error(`Template application failed`, { executionId, error });

      // Update application result with failure
      await this.templatesRepo.updateApplicationResult(
        executionId,
        'failed',
        [],
        `Template application failed: ${error.message}`
      );

      return {
        template_id: id,
        execution_id: executionId,
        status: 'failed',
        results: [],
        summary: error.message,
        total_duration_ms: Date.now() - startTime
      };
    }
  }

  /**
   * Get most used templates
   */
  async getMostUsedTemplates(limit: number = 10): Promise<TemplateEntity[]> {
    return await this.templatesRepo.getMostUsed(limit);
  }

  /**
   * Generate summary from template results
   */
  private generateSummary(template: TemplateEntity, results: any[]): string {
    const successfulSteps = results.filter(r => r.result !== null).length;
    const totalSteps = results.length;
    
    if (successfulSteps === totalSteps) {
      return `Successfully completed all ${totalSteps} steps of ${template.name}`;
    } else if (successfulSteps > 0) {
      return `Partially completed ${template.name}: ${successfulSteps} of ${totalSteps} steps succeeded`;
    } else {
      return `Failed to execute ${template.name}`;
    }
  }
}