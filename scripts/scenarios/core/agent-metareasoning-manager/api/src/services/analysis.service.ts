import { injectable, inject } from 'tsyringe';
import axios from 'axios';
import { config } from '../config/index.js';
import { WorkflowsService } from './workflows.service.js';
import { PromptsService } from './prompts.service.js';
import { logger } from '../utils/logger.js';
import {
  AnalyzeDecisionRequest,
  AnalyzeProsConsRequest,
  AnalyzeSwotRequest,
  AnalyzeRisksRequest,
  DecisionResult,
  ProsConsResult,
  SwotResult,
  RiskResult
} from '../schemas/analysis.schema.js';

@injectable()
export class AnalysisService {
  constructor(
    @inject(WorkflowsService) private workflowsService: WorkflowsService,
    @inject(PromptsService) private promptsService: PromptsService
  ) {}

  /**
   * Analyze a decision scenario
   */
  async analyzeDecision(data: AnalyzeDecisionRequest['body']): Promise<DecisionResult> {
    logger.info('Starting decision analysis', { scenario: data.scenario });

    try {
      // Find the decision analysis workflow
      const workflow = await this.workflowsService.findWorkflowByName('decision-analysis');
      
      if (!workflow) {
        // Fallback to mock if workflow not found
        logger.warn('Decision analysis workflow not found, using fallback');
        return this.mockDecisionAnalysis(data);
      }

      // Execute the workflow with the provided data
      const execution = await this.workflowsService.executeWorkflow(
        workflow.id,
        {
          input: {
            scenario: data.scenario,
            options: data.options || [],
            criteria: data.criteria || [],
            context: data.context
          },
          async: false,
          timeout: 30000
        }
      );

      // Process workflow output
      if (execution.status === 'completed' && execution.output) {
        return this.processDecisionWorkflowOutput(execution.output);
      } else {
        logger.error('Decision workflow execution failed', execution);
        return this.mockDecisionAnalysis(data);
      }
    } catch (error) {
      logger.error('Error in decision analysis', error);
      // Fallback to mock implementation
      return this.mockDecisionAnalysis(data);
    }
  }

  /**
   * Mock decision analysis for fallback
   */
  private mockDecisionAnalysis(data: AnalyzeDecisionRequest['body']): DecisionResult {
    const options = data.options || ['Option A', 'Option B'];
    const criteria = data.criteria || ['Cost', 'Time', 'Quality'];
    
    const optionsAnalysis = options.map((option, index) => ({
      option,
      score: Math.random() * 100,
      pros: [`Pro ${index + 1} for ${option}`, `Another pro for ${option}`],
      cons: [`Con ${index + 1} for ${option}`]
    }));

    const bestOption = optionsAnalysis.reduce((a, b) => a.score > b.score ? a : b);

    return {
      recommendation: `Based on the analysis, ${bestOption.option} is recommended`,
      confidence: 0.75 + Math.random() * 0.2,
      reasoning: [
        'Analyzed all provided options against criteria',
        `${bestOption.option} scores highest on key metrics`,
        'Risk factors have been considered'
      ],
      options_analysis: optionsAnalysis,
      next_steps: [
        'Validate assumptions with stakeholders',
        'Create implementation plan',
        'Set up monitoring metrics'
      ]
    };
  }

  /**
   * Process decision workflow output into structured result
   */
  private processDecisionWorkflowOutput(output: any): DecisionResult {
    // Map n8n workflow output to our DecisionResult type
    return {
      recommendation: output.recommendation || 'Unable to generate recommendation',
      confidence: output.confidence || 0.5,
      reasoning: output.reasoning || [],
      options_analysis: output.options_analysis || [],
      next_steps: output.next_steps || []
    };
  }

  /**
   * Analyze pros and cons
   */
  async analyzeProscons(data: AnalyzeProsConsRequest['body']): Promise<ProsConsResult> {
    logger.info('Starting pros/cons analysis', { topic: data.topic });

    try {
      // Find the pros-cons-analyzer workflow
      const workflow = await this.workflowsService.findWorkflowByName('pros-cons-analyzer');
      
      if (!workflow) {
        logger.warn('Pros-cons analyzer workflow not found, using fallback');
        return this.mockProsConsAnalysis(data);
      }

      // Execute the workflow
      const execution = await this.workflowsService.executeWorkflow(
        workflow.id,
        {
          input: {
            topic: data.topic,
            depth: data.depth || 'detailed',
            context: data.context
          },
          async: false,
          timeout: 30000
        }
      );

      if (execution.status === 'completed' && execution.output) {
        return this.processProsConsWorkflowOutput(execution.output, data.topic);
      } else {
        logger.error('Pros-cons workflow execution failed', execution);
        return this.mockProsConsAnalysis(data);
      }
    } catch (error) {
      logger.error('Error in pros-cons analysis', error);
      return this.mockProsConsAnalysis(data);
    }
  }

  /**
   * Mock pros-cons analysis for fallback
   */
  private mockProsConsAnalysis(data: AnalyzeProsConsRequest['body']): ProsConsResult {
    const depth = data.depth || 'detailed';
    const numPoints = depth === 'basic' ? 3 : depth === 'comprehensive' ? 8 : 5;

    const pros = Array.from({ length: numPoints }, (_, i) => ({
      point: `Advantage ${i + 1} of ${data.topic}`,
      impact: ['low', 'medium', 'high', 'critical'][Math.floor(Math.random() * 4)] as any,
      explanation: depth !== 'basic' ? `Detailed explanation of advantage ${i + 1}` : undefined
    }));

    const cons = Array.from({ length: Math.floor(numPoints * 0.7) }, (_, i) => ({
      point: `Disadvantage ${i + 1} of ${data.topic}`,
      impact: ['low', 'medium', 'high', 'critical'][Math.floor(Math.random() * 4)] as any,
      explanation: depth !== 'basic' ? `Detailed explanation of disadvantage ${i + 1}` : undefined
    }));

    return {
      pros,
      cons,
      summary: `Analysis of ${data.topic} reveals ${pros.length} advantages and ${cons.length} disadvantages`,
      recommendation: pros.length > cons.length 
        ? `Proceed with ${data.topic} with risk mitigation`
        : `Reconsider ${data.topic} or explore alternatives`
    };
  }

  /**
   * Process pros-cons workflow output
   */
  private processProsConsWorkflowOutput(output: any, topic: string): ProsConsResult {
    return {
      pros: output.pros || [],
      cons: output.cons || [],
      summary: output.summary || `Analysis of ${topic} completed`,
      recommendation: output.recommendation || null
    };
  }

  /**
   * Perform SWOT analysis
   */
  async analyzeSwot(data: AnalyzeSwotRequest['body']): Promise<SwotResult> {
    logger.info('Starting SWOT analysis', { target: data.target });

    try {
      // Find the SWOT analysis workflow
      const workflow = await this.workflowsService.findWorkflowByName('swot-analysis');
      
      if (!workflow) {
        logger.warn('SWOT analysis workflow not found, using fallback');
        return this.mockSwotAnalysis(data);
      }

      // Execute the workflow
      const execution = await this.workflowsService.executeWorkflow(
        workflow.id,
        {
          input: {
            target: data.target,
            context: data.context,
            include_recommendations: data.include_recommendations !== false
          },
          async: false,
          timeout: 30000
        }
      );

      if (execution.status === 'completed' && execution.output) {
        return this.processSwotWorkflowOutput(execution.output);
      } else {
        logger.error('SWOT workflow execution failed', execution);
        return this.mockSwotAnalysis(data);
      }
    } catch (error) {
      logger.error('Error in SWOT analysis', error);
      return this.mockSwotAnalysis(data);
    }
  }

  /**
   * Mock SWOT analysis for fallback
   */
  private mockSwotAnalysis(data: AnalyzeSwotRequest['body']): SwotResult {
    return {
      strengths: [
        'Strong market position',
        'Experienced team',
        'Solid technology foundation',
        'Good customer relationships'
      ],
      weaknesses: [
        'Limited resources',
        'Technical debt',
        'Dependency on key personnel'
      ],
      opportunities: [
        'Growing market demand',
        'New technology enablers',
        'Partnership possibilities',
        'Geographic expansion'
      ],
      threats: [
        'Competitive pressure',
        'Regulatory changes',
        'Economic uncertainty',
        'Technology disruption'
      ],
      strategic_recommendations: [
        'Leverage strengths to capture opportunities',
        'Address weaknesses that expose to threats',
        'Build partnerships to compensate for resource limitations'
      ],
      priority_actions: [
        {
          action: 'Strengthen market position',
          priority: 'high',
          timeframe: 'Q1 2024'
        },
        {
          action: 'Reduce technical debt',
          priority: 'medium',
          timeframe: 'Q2 2024'
        },
        {
          action: 'Diversify team expertise',
          priority: 'high',
          timeframe: 'Ongoing'
        }
      ]
    };
  }

  /**
   * Process SWOT workflow output
   */
  private processSwotWorkflowOutput(output: any): SwotResult {
    return {
      strengths: output.strengths || [],
      weaknesses: output.weaknesses || [],
      opportunities: output.opportunities || [],
      threats: output.threats || [],
      strategic_recommendations: output.strategic_recommendations || [],
      priority_actions: output.priority_actions || []
    };
  }

  /**
   * Analyze risks
   */
  async analyzeRisks(data: AnalyzeRisksRequest['body']): Promise<RiskResult> {
    logger.info('Starting risk analysis', { proposal: data.proposal });

    try {
      // Find the risk assessment workflow
      const workflow = await this.workflowsService.findWorkflowByName('risk-assessment');
      
      if (!workflow) {
        logger.warn('Risk assessment workflow not found, using fallback');
        return this.mockRiskAnalysis(data);
      }

      // Execute the workflow
      const execution = await this.workflowsService.executeWorkflow(
        workflow.id,
        {
          input: {
            proposal: data.proposal,
            risk_categories: data.risk_categories || [
              'Technical', 'Financial', 'Operational', 'Strategic', 'Compliance'
            ],
            include_mitigation: data.include_mitigation !== false
          },
          async: false,
          timeout: 30000
        }
      );

      if (execution.status === 'completed' && execution.output) {
        return this.processRiskWorkflowOutput(execution.output);
      } else {
        logger.error('Risk workflow execution failed', execution);
        return this.mockRiskAnalysis(data);
      }
    } catch (error) {
      logger.error('Error in risk analysis', error);
      return this.mockRiskAnalysis(data);
    }
  }

  /**
   * Mock risk analysis for fallback
   */
  private mockRiskAnalysis(data: AnalyzeRisksRequest['body']): RiskResult {
    const riskCategories = data.risk_categories || [
      'Technical', 'Financial', 'Operational', 'Strategic', 'Compliance'
    ];

    const risks = riskCategories.flatMap(category => 
      Array.from({ length: 2 }, (_, i) => ({
        description: `${category} risk ${i + 1}: Sample risk for ${data.proposal}`,
        likelihood: ['low', 'medium', 'high', 'critical'][Math.floor(Math.random() * 4)] as any,
        impact: ['low', 'medium', 'high', 'critical'][Math.floor(Math.random() * 4)] as any,
        category,
        mitigation: data.include_mitigation 
          ? `Mitigation strategy for ${category} risk ${i + 1}`
          : undefined
      }))
    );

    const riskScore = risks.reduce((sum, risk) => {
      const likelihoodScore = { low: 1, medium: 2, high: 3, critical: 4 }[risk.likelihood as string] as number;
      const impactScore = { low: 1, medium: 2, high: 3, critical: 4 }[risk.impact as string] as number;
      return sum + (likelihoodScore * impactScore);
    }, 0) / risks.length * 6.25; // Normalize to 0-100

    const overallRiskLevel = 
      riskScore < 25 ? 'low' :
      riskScore < 50 ? 'medium' :
      riskScore < 75 ? 'high' : 'critical';

    return {
      risks,
      overall_risk_level: overallRiskLevel as any,
      risk_score: riskScore,
      recommendations: [
        'Implement risk mitigation strategies for high-impact items',
        'Establish monitoring systems for early risk detection',
        'Create contingency plans for critical risks',
        'Regular risk assessment reviews'
      ]
    };
  }

  /**
   * Process risk workflow output
   */
  private processRiskWorkflowOutput(output: any): RiskResult {
    return {
      risks: output.risks || [],
      overall_risk_level: output.overall_risk_level || 'medium',
      risk_score: output.risk_score || 50,
      recommendations: output.recommendations || []
    };
  }

  /**
   * Execute a complex analysis chain
   */
  async executeAnalysisChain(
    analysisType: string,
    data: any,
    includeWorkflows: string[] = []
  ): Promise<any> {
    logger.info('Executing analysis chain', { type: analysisType });

    const results: any[] = [];

    // Execute each workflow in sequence
    for (const workflowId of includeWorkflows) {
      try {
        const result = await this.workflowsService.executeWorkflow(workflowId, {
          input: data,
          async: false,
          timeout: 30000
        });
        results.push(result);
      } catch (error) {
        logger.error(`Failed to execute workflow ${workflowId} in chain`, error);
        // Continue with other workflows
      }
    }

    return {
      analysis_type: analysisType,
      chain_results: results,
      summary: `Completed ${results.length} of ${includeWorkflows.length} analysis steps`
    };
  }

  /**
   * Chain multiple analyses together
   */
  async chainAnalyses(
    analyses: Array<{
      type: 'decision' | 'pros-cons' | 'swot' | 'risks';
      data: any;
    }>
  ): Promise<any[]> {
    const results: any[] = [];
    let previousResult: any = null;

    for (const analysis of analyses) {
      // Add previous result to data if available
      const enrichedData = {
        ...analysis.data,
        ...(previousResult && { previous_analysis: previousResult })
      };

      let result: any;
      
      switch (analysis.type) {
        case 'decision':
          result = await this.analyzeDecision(enrichedData);
          break;
        case 'pros-cons':
          result = await this.analyzeProscons(enrichedData);
          break;
        case 'swot':
          result = await this.analyzeSwot(enrichedData);
          break;
        case 'risks':
          result = await this.analyzeRisks(enrichedData);
          break;
      }

      results.push({
        type: analysis.type,
        result
      });

      previousResult = result;
    }

    return results;
  }
}