import { Request, Response } from 'express';
import { container } from 'tsyringe';
import { AnalysisService } from '../services/analysis.service.js';
import { asyncHandler } from '../utils/asyncHandler.js';
import {
  AnalyzeDecisionRequest,
  AnalyzeProsConsRequest,
  AnalyzeSwotRequest,
  AnalyzeRisksRequest
} from '../schemas/analysis.schema.js';

class AnalysisController {
  /**
   * Analyze a decision
   */
  analyzeDecision = asyncHandler(async (req: Request<{}, {}, AnalyzeDecisionRequest['body']>, res: Response) => {
    const analysisService = container.resolve(AnalysisService);
    
    const result = await analysisService.analyzeDecision(req.body);
    
    res.json({
      status: 'success',
      data: result,
      meta: {
        analysis_type: 'decision',
        timestamp: new Date().toISOString()
      }
    });
  });

  /**
   * Analyze pros and cons
   */
  analyzeProscons = asyncHandler(async (req: Request<{}, {}, AnalyzeProsConsRequest['body']>, res: Response) => {
    const analysisService = container.resolve(AnalysisService);
    
    const result = await analysisService.analyzeProscons(req.body);
    
    res.json({
      status: 'success',
      data: result,
      meta: {
        analysis_type: 'pros-cons',
        timestamp: new Date().toISOString()
      }
    });
  });

  /**
   * Perform SWOT analysis
   */
  analyzeSwot = asyncHandler(async (req: Request<{}, {}, AnalyzeSwotRequest['body']>, res: Response) => {
    const analysisService = container.resolve(AnalysisService);
    
    const result = await analysisService.analyzeSwot(req.body);
    
    res.json({
      status: 'success',
      data: result,
      meta: {
        analysis_type: 'swot',
        timestamp: new Date().toISOString()
      }
    });
  });

  /**
   * Analyze risks
   */
  analyzeRisks = asyncHandler(async (req: Request<{}, {}, AnalyzeRisksRequest['body']>, res: Response) => {
    const analysisService = container.resolve(AnalysisService);
    
    const result = await analysisService.analyzeRisks(req.body);
    
    res.json({
      status: 'success',
      data: result,
      meta: {
        analysis_type: 'risks',
        timestamp: new Date().toISOString()
      }
    });
  });

  /**
   * Execute analysis chain
   */
  executeChain = asyncHandler(async (req: Request, res: Response) => {
    const analysisService = container.resolve(AnalysisService);
    
    const { analyses } = req.body;
    
    if (!Array.isArray(analyses) || analyses.length === 0) {
      res.status(400).json({
        status: 'error',
        message: 'Analyses array is required'
      });
      return;
    }
    
    const result = await analysisService.chainAnalyses(analyses);
    
    res.json({
      status: 'success',
      data: result,
      meta: {
        chain_length: analyses.length,
        timestamp: new Date().toISOString()
      }
    });
  });
}

export const analysisController = new AnalysisController();