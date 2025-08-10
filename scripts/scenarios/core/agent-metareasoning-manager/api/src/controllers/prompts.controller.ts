import { Request, Response } from 'express';
import { container } from 'tsyringe';
import { PromptsService } from '../services/prompts.service.js';
import { asyncHandler } from '../utils/asyncHandler.js';
import { 
  ListPromptsQuery,
  CreatePromptRequest,
  UpdatePromptRequest,
  TestPromptRequest
} from '../schemas/prompts.schema.js';

class PromptsController {
  /**
   * List all prompts
   */
  list = asyncHandler(async (req: Request, res: Response) => {
    const promptsService = container.resolve(PromptsService);
    
    const { category, limit = 20, offset = 0 } = req.query as any;
    const result = await promptsService.getPrompts(category, limit, offset);
    
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
   * Get a single prompt
   */
  getById = asyncHandler(async (req: Request, res: Response) => {
    const promptsService = container.resolve(PromptsService);
    
    const prompt = await promptsService.getPromptById(req.params.id);
    
    res.json({
      status: 'success',
      data: prompt
    });
  });

  /**
   * Create a new prompt
   */
  create = asyncHandler(async (req: Request<{}, {}, CreatePromptRequest['body']>, res: Response) => {
    const promptsService = container.resolve(PromptsService);
    
    const prompt = await promptsService.createPrompt(req.body);
    
    res.status(201).json({
      status: 'success',
      data: prompt,
      message: 'Prompt created successfully'
    });
  });

  /**
   * Update a prompt
   */
  update = asyncHandler(async (req: Request, res: Response) => {
    const promptsService = container.resolve(PromptsService);
    
    const prompt = await promptsService.updatePrompt(req.params.id, req.body);
    
    res.json({
      status: 'success',
      data: prompt,
      message: 'Prompt updated successfully'
    });
  });

  /**
   * Delete a prompt
   */
  delete = asyncHandler(async (req: Request, res: Response) => {
    const promptsService = container.resolve(PromptsService);
    
    await promptsService.deletePrompt(req.params.id);
    
    res.json({
      status: 'success',
      message: 'Prompt archived successfully'
    });
  });

  /**
   * Test a prompt
   */
  test = asyncHandler(async (req: Request, res: Response) => {
    const promptsService = container.resolve(PromptsService);
    
    const result = await promptsService.testPrompt(req.params.id, req.body);
    
    res.json({
      status: 'success',
      data: result
    });
  });

  /**
   * Search prompts
   */
  search = asyncHandler(async (req: Request, res: Response) => {
    const promptsService = container.resolve(PromptsService);
    
    const { q } = req.query;
    if (!q) {
      res.status(400).json({
        status: 'error',
        message: 'Search query is required'
      });
      return;
    }
    
    const prompts = await promptsService.searchPrompts(q as string);
    
    res.json({
      status: 'success',
      data: prompts,
      meta: {
        count: prompts.length
      }
    });
  });
}

export const promptsController = new PromptsController();