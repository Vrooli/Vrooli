import { injectable, inject } from 'tsyringe';
import { PromptsService } from './prompts.service.js';
import { WorkflowsService } from './workflows.service.js';
import { AnalysisService } from './analysis.service.js';
import { TemplateEntity } from '../repositories/templates.repository.js';
import { logger } from '../utils/logger.js';

@injectable()
export class OrchestrationService {
  constructor(
    @inject(PromptsService) private promptsService: PromptsService,
    @inject(WorkflowsService) private workflowsService: WorkflowsService,
    @inject(AnalysisService) private analysisService: AnalysisService
  ) {}

  /**
   * Execute a template with all its steps
   */
  async executeTemplate(
    template: TemplateEntity,
    context: string,
    variables: Record<string, string>,
    options?: {
      async?: boolean;
      includeIntermediateResults?: boolean;
    }
  ): Promise<any[]> {
    const results: any[] = [];
    const processedContext = this.processVariables(context, variables);

    for (const step of template.structure.sequence) {
      const stepStart = Date.now();
      let stepResult: any = null;

      try {
        switch (step.type) {
          case 'prompt':
            stepResult = await this.executePromptStep(
              step.id,
              processedContext,
              variables,
              step.config
            );
            break;

          case 'workflow':
            stepResult = await this.executeWorkflowStep(
              step.id,
              processedContext,
              variables,
              step.config
            );
            break;

          case 'analysis':
            stepResult = await this.executeAnalysisStep(
              step.id,
              processedContext,
              variables,
              step.config
            );
            break;

          default:
            logger.warn(`Unknown step type: ${step.type}`);
            continue;
        }

        results.push({
          step: step.id,
          type: step.type,
          result: stepResult,
          duration_ms: Date.now() - stepStart
        });

        // Use previous step result as input for next step if configured
        if (step.config?.use_previous_result && stepResult) {
          variables['previous_result'] = JSON.stringify(stepResult);
        }

      } catch (error: any) {
        logger.error(`Failed to execute template step`, { step, error });
        
        results.push({
          step: step.id,
          type: step.type,
          result: null,
          error: error.message,
          duration_ms: Date.now() - stepStart
        });

        // Stop execution on error unless configured to continue
        if (!step.config?.continue_on_error) {
          throw error;
        }
      }
    }

    return results;
  }

  /**
   * Execute a prompt step
   */
  private async executePromptStep(
    promptId: string,
    context: string,
    variables: Record<string, string>,
    config?: any
  ): Promise<any> {
    const prompt = await this.promptsService.getPromptById(promptId);
    
    // Combine context with prompt template
    const fullContext = `${context}\n\n${prompt.template}`;
    const processedPrompt = this.processVariables(fullContext, variables);

    // In real implementation, would call LLM here
    logger.debug('Executing prompt step', { promptId, processedPrompt });

    return {
      prompt_id: promptId,
      processed: processedPrompt,
      response: `Mock response for prompt ${prompt.name}`
    };
  }

  /**
   * Execute a workflow step
   */
  private async executeWorkflowStep(
    workflowId: string,
    context: string,
    variables: Record<string, string>,
    config?: any
  ): Promise<any> {
    const input = {
      context,
      variables,
      ...config?.additional_inputs
    };

    const result = await this.workflowsService.executeWorkflow(workflowId, {
      input,
      async: false,
      timeout: config?.timeout || 30000
    });

    return result.output;
  }

  /**
   * Execute an analysis step
   */
  private async executeAnalysisStep(
    analysisType: string,
    context: string,
    variables: Record<string, string>,
    config?: any
  ): Promise<any> {
    const data = {
      context,
      ...variables,
      ...config
    };

    switch (analysisType) {
      case 'decision':
        return await this.analysisService.analyzeDecision({
          scenario: context,
          ...data
        });

      case 'pros-cons':
        return await this.analysisService.analyzeProscons({
          topic: context,
          ...data
        });

      case 'swot':
        return await this.analysisService.analyzeSwot({
          target: context,
          ...data
        });

      case 'risks':
        return await this.analysisService.analyzeRisks({
          proposal: context,
          ...data
        });

      default:
        throw new Error(`Unknown analysis type: ${analysisType}`);
    }
  }

  /**
   * Replace variables in text
   */
  private processVariables(text: string, variables: Record<string, string>): string {
    let processed = text;
    
    for (const [key, value] of Object.entries(variables)) {
      // Replace {{variable}} syntax
      processed = processed.replace(new RegExp(`{{${key}}}`, 'g'), value);
      // Also replace ${variable} syntax
      processed = processed.replace(new RegExp(`\\$\\{${key}\\}`, 'g'), value);
    }
    
    return processed;
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
          result = await this.analysisService.analyzeDecision(enrichedData);
          break;
        case 'pros-cons':
          result = await this.analysisService.analyzeProscons(enrichedData);
          break;
        case 'swot':
          result = await this.analysisService.analyzeSwot(enrichedData);
          break;
        case 'risks':
          result = await this.analysisService.analyzeRisks(enrichedData);
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