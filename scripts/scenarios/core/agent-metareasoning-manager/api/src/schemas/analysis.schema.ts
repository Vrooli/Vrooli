import { z } from 'zod';
import { prioritySchema } from './common.schema.js';

// Base analysis request schema
const baseAnalysisSchema = z.object({
  context: z.string().max(5000).optional(),
  timeframe: z.string().optional(),
  constraints: z.array(z.string()).optional(),
  metadata: z.record(z.any()).optional()
});

// Decision analysis request
export const analyzeDecisionSchema = z.object({
  body: baseAnalysisSchema.extend({
    scenario: z.string().min(10).max(5000),
    options: z.array(z.string()).min(2).max(10).optional(),
    criteria: z.array(z.string()).optional(),
    weights: z.record(z.number().min(0).max(1)).optional()
  })
});

// Pros and cons analysis request
export const analyzeProsConsSchema = z.object({
  body: baseAnalysisSchema.extend({
    topic: z.string().min(5).max(1000),
    perspective: z.string().optional(),
    depth: z.enum(['basic', 'detailed', 'comprehensive']).default('detailed')
  })
});

// SWOT analysis request
export const analyzeSwotSchema = z.object({
  body: baseAnalysisSchema.extend({
    target: z.string().min(5).max(1000),
    industry: z.string().optional(),
    competitors: z.array(z.string()).optional(),
    market_conditions: z.string().optional()
  })
});

// Risk analysis request
export const analyzeRisksSchema = z.object({
  body: baseAnalysisSchema.extend({
    proposal: z.string().min(10).max(5000),
    risk_categories: z.array(z.string()).optional(),
    risk_appetite: z.enum(['conservative', 'moderate', 'aggressive']).default('moderate'),
    include_mitigation: z.boolean().default(true)
  })
});

// Analysis result schemas
export const decisionResultSchema = z.object({
  recommendation: z.string(),
  confidence: z.number().min(0).max(1),
  reasoning: z.array(z.string()),
  options_analysis: z.array(z.object({
    option: z.string(),
    score: z.number(),
    pros: z.array(z.string()),
    cons: z.array(z.string())
  })),
  next_steps: z.array(z.string()).optional()
});

export const prosConsResultSchema = z.object({
  pros: z.array(z.object({
    point: z.string(),
    impact: prioritySchema,
    explanation: z.string().optional()
  })),
  cons: z.array(z.object({
    point: z.string(),
    impact: prioritySchema,
    explanation: z.string().optional()
  })),
  summary: z.string(),
  recommendation: z.string().optional()
});

export const swotResultSchema = z.object({
  strengths: z.array(z.string()),
  weaknesses: z.array(z.string()),
  opportunities: z.array(z.string()),
  threats: z.array(z.string()),
  strategic_recommendations: z.array(z.string()),
  priority_actions: z.array(z.object({
    action: z.string(),
    priority: prioritySchema,
    timeframe: z.string()
  }))
});

export const riskResultSchema = z.object({
  risks: z.array(z.object({
    description: z.string(),
    likelihood: prioritySchema,
    impact: prioritySchema,
    category: z.string(),
    mitigation: z.string().optional()
  })),
  overall_risk_level: prioritySchema,
  risk_score: z.number().min(0).max(100),
  recommendations: z.array(z.string())
});

export type AnalyzeDecisionRequest = z.infer<typeof analyzeDecisionSchema>;
export type AnalyzeProsConsRequest = z.infer<typeof analyzeProsConsSchema>;
export type AnalyzeSwotRequest = z.infer<typeof analyzeSwotSchema>;
export type AnalyzeRisksRequest = z.infer<typeof analyzeRisksSchema>;
export type DecisionResult = z.infer<typeof decisionResultSchema>;
export type ProsConsResult = z.infer<typeof prosConsResultSchema>;
export type SwotResult = z.infer<typeof swotResultSchema>;
export type RiskResult = z.infer<typeof riskResultSchema>;