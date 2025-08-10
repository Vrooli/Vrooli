import { Request, Response } from 'express';
import { container } from 'tsyringe';
import { TemplatesService } from '../services/templates.service.js';
import { asyncHandler } from '../utils/asyncHandler.js';
import {
  ListTemplatesQuery,
  CreateTemplateRequest,
  ApplyTemplateRequest
} from '../schemas/templates.schema.js';

class TemplatesController {
  /**
   * List all templates
   */
  list = asyncHandler(async (req: Request, res: Response) => {
    const templatesService = container.resolve(TemplatesService);
    
    const { category, search, limit = 20, offset = 0 } = req.query as any;
    const result = await templatesService.getTemplates(category, search, limit, offset);
    
    res.json({
      status: 'success',
      data: result,
      meta: {
        total: result.total,
        limit,
        offset
      }
    });
  });

  /**
   * Get a single template
   */
  getById = asyncHandler(async (req: Request, res: Response) => {
    const templatesService = container.resolve(TemplatesService);
    
    const template = await templatesService.getTemplateById(req.params.id);
    
    res.json({
      status: 'success',
      data: template
    });
  });

  /**
   * Create a new template
   */
  create = asyncHandler(async (req: Request<{}, {}, CreateTemplateRequest['body']>, res: Response) => {
    const templatesService = container.resolve(TemplatesService);
    
    const template = await templatesService.createTemplate(req.body);
    
    res.status(201).json({
      status: 'success',
      data: template,
      message: 'Template created successfully'
    });
  });

  /**
   * Apply a template
   */
  apply = asyncHandler(async (req: Request, res: Response) => {
    const templatesService = container.resolve(TemplatesService);
    
    const result = await templatesService.applyTemplate(req.params.id, req.body);
    
    res.json({
      status: 'success',
      data: result,
      message: 'Template applied successfully'
    });
  });

  /**
   * Get most used templates
   */
  getMostUsed = asyncHandler(async (req: Request, res: Response) => {
    const templatesService = container.resolve(TemplatesService);
    
    const limit = parseInt(req.query.limit as string) || 10;
    const templates = await templatesService.getMostUsedTemplates(limit);
    
    res.json({
      status: 'success',
      data: templates,
      meta: {
        count: templates.length
      }
    });
  });
}

export const templatesController = new TemplatesController();