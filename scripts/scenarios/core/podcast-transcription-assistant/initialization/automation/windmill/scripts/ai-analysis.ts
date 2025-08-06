// AI Analysis Handler - Manages AI analysis requests and results
// This script interfaces with the n8n AI analysis workflow

interface AnalysisRequest {
  transcriptionId: string;
  analysisType: 'summary' | 'key_insights' | 'custom';
  customPrompt?: string;
  aiModel?: string;
  temperature?: number;
  maxTokens?: number;
  sessionId?: string;
  userIdentifier?: string;
}

interface AnalysisResponse {
  success: boolean;
  message: string;
  analysisId: string;
  transcriptionId: string;
  analysisType: string;
  filename: string;
  result: string;
  aiModel: string;
  processingTime: number;
  tokensUsed: number;
  confidence: number;
  createdAt: string;
  actions: {
    viewTranscription: string;
    copyResult: string;
    exportAnalysis: string;
  };
}

interface AnalysisHistory {
  id: string;
  transcriptionId: string;
  analysisType: string;
  result: string;
  createdAt: string;
  filename: string;
}

// Analysis state management
let currentAnalysis: AnalysisResponse | null = null;
let analysisHistory: AnalysisHistory[] = [];

/**
 * Main function to handle analysis type selection from UI
 */
export async function handleAnalysisType(
  transcriptionId: string, 
  analysisType: 'summary' | 'key_insights' | 'custom',
  customPrompt?: string
): Promise<AnalysisResponse> {
  
  const request: AnalysisRequest = {
    transcriptionId: transcriptionId,
    analysisType: analysisType,
    customPrompt: customPrompt,
    sessionId: generateSessionId(),
    userIdentifier: 'windmill-analysis'
  };
  
  return await requestAnalysis(request);
}

/**
 * Requests AI analysis from the n8n workflow
 */
export async function requestAnalysis(analysisRequest: AnalysisRequest): Promise<AnalysisResponse> {
  try {
    // Validate request
    const validationError = validateAnalysisRequest(analysisRequest);
    if (validationError) {
      throw new Error(validationError);
    }
    
    // Show loading state in UI
    setAnalysisLoading(true);
    
    // Prepare payload for n8n AI analysis workflow
    const payload = {
      transcriptionId: analysisRequest.transcriptionId,
      analysisType: analysisRequest.analysisType,
      customPrompt: analysisRequest.customPrompt,
      aiModel: analysisRequest.aiModel || 'llama3.1:8b',
      temperature: analysisRequest.temperature || 0.7,
      maxTokens: analysisRequest.maxTokens || 2048,
      sessionId: analysisRequest.sessionId || generateSessionId(),
      userIdentifier: analysisRequest.userIdentifier || 'windmill-analysis'
    };
    
    // Call n8n AI analysis workflow
    const response = await fetch('http://localhost:5678/webhook/ai-analysis', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(payload)
    });
    
    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`AI analysis failed: ${response.status} - ${errorText}`);
    }
    
    const result = await response.json() as AnalysisResponse;
    
    // Update current analysis state
    currentAnalysis = result;
    
    // Add to analysis history
    addToAnalysisHistory({
      id: result.analysisId,
      transcriptionId: result.transcriptionId,
      analysisType: result.analysisType,
      result: result.result,
      createdAt: result.createdAt,
      filename: result.filename
    });
    
    // Update UI
    updateAnalysisUI(result);
    
    return result;
    
  } catch (error) {
    console.error('Analysis error:', error);
    setAnalysisLoading(false);
    throw error;
  } finally {
    setAnalysisLoading(false);
  }
}

/**
 * Gets analysis history for a specific transcription
 */
export async function getAnalysisHistory(transcriptionId: string): Promise<AnalysisHistory[]> {
  try {
    // In a real implementation, this would query the database
    const response = await fetch(`http://localhost:3000/api/transcriptions/${transcriptionId}/analyses`, {
      method: 'GET',
      headers: {
        'Accept': 'application/json'
      }
    });
    
    if (!response.ok) {
      throw new Error(`Failed to fetch analysis history: ${response.status}`);
    }
    
    const analyses = await response.json();
    return analyses.map((a: any) => ({
      id: a.id,
      transcriptionId: a.transcription_id,
      analysisType: a.analysis_type,
      result: a.result_text,
      createdAt: a.created_at,
      filename: a.transcription?.filename || 'Unknown'
    }));
    
  } catch (error) {
    console.error('Failed to fetch analysis history:', error);
    return [];
  }
}

/**
 * Saves analysis results to user's collection
 */
export async function saveAnalysis(analysisId: string, name?: string): Promise<{ success: boolean; message: string }> {
  try {
    if (!currentAnalysis || currentAnalysis.analysisId !== analysisId) {
      throw new Error('Analysis not found');
    }
    
    const savedAnalysis = {
      id: analysisId,
      name: name || `${currentAnalysis.analysisType} - ${currentAnalysis.filename}`,
      transcriptionId: currentAnalysis.transcriptionId,
      analysisType: currentAnalysis.analysisType,
      result: currentAnalysis.result,
      filename: currentAnalysis.filename,
      savedAt: new Date().toISOString()
    };
    
    // Save to database via API
    const response = await fetch('http://localhost:3000/api/saved-analyses', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(savedAnalysis)
    });
    
    if (!response.ok) {
      throw new Error(`Save failed: ${response.status}`);
    }
    
    return { success: true, message: 'Analysis saved to your collection' };
    
  } catch (error) {
    console.error('Save analysis error:', error);
    return { 
      success: false, 
      message: error instanceof Error ? error.message : 'Failed to save analysis' 
    };
  }
}

/**
 * Exports analysis in different formats
 */
export async function exportAnalysis(
  analysisId: string, 
  format: 'txt' | 'pdf' | 'docx' | 'json'
): Promise<Blob> {
  
  if (!currentAnalysis || currentAnalysis.analysisId !== analysisId) {
    throw new Error('Analysis not found');
  }
  
  let content: string;
  let mimeType: string;
  
  switch (format) {
    case 'txt':
      content = formatAnalysisAsText(currentAnalysis);
      mimeType = 'text/plain';
      break;
      
    case 'json':
      content = JSON.stringify(currentAnalysis, null, 2);
      mimeType = 'application/json';
      break;
      
    case 'pdf':
    case 'docx':
      // For PDF and DOCX, we'd use a service or library
      // For now, fall back to formatted text
      content = formatAnalysisAsText(currentAnalysis);
      mimeType = 'text/plain';
      break;
      
    default:
      throw new Error(`Unsupported export format: ${format}`);
  }
  
  return new Blob([content], { type: mimeType });
}

/**
 * Compares multiple analyses of the same transcription
 */
export async function compareAnalyses(analysisIds: string[]): Promise<{
  transcriptionId: string;
  filename: string;
  analyses: AnalysisHistory[];
  comparison: string;
}> {
  
  if (analysisIds.length < 2) {
    throw new Error('At least 2 analyses required for comparison');
  }
  
  // Fetch all analyses
  const analyses = await Promise.all(
    analysisIds.map(id => getAnalysisById(id))
  );
  
  // Ensure all analyses are from the same transcription
  const transcriptionIds = [...new Set(analyses.map(a => a.transcriptionId))];
  if (transcriptionIds.length > 1) {
    throw new Error('Cannot compare analyses from different transcriptions');
  }
  
  // Generate comparison using AI
  const comparisonPrompt = `
    Compare and contrast the following analyses of the same transcription:
    
    ${analyses.map((a, i) => `
    Analysis ${i + 1} (${a.analysisType}):
    ${a.result}
    `).join('\n')}
    
    Provide a comparison highlighting:
    1. Key similarities
    2. Important differences
    3. Complementary insights
    4. Overall synthesis
  `;
  
  // Request comparison analysis
  const comparisonResult = await requestAnalysis({
    transcriptionId: analyses[0].transcriptionId,
    analysisType: 'custom',
    customPrompt: comparisonPrompt
  });
  
  return {
    transcriptionId: analyses[0].transcriptionId,
    filename: analyses[0].filename,
    analyses: analyses,
    comparison: comparisonResult.result
  };
}

/**
 * Gets predefined analysis templates
 */
export function getAnalysisTemplates(): Array<{
  name: string;
  type: 'summary' | 'key_insights' | 'custom';
  prompt?: string;
  description: string;
}> {
  return [
    {
      name: 'Executive Summary',
      type: 'summary',
      description: 'Concise overview highlighting main points and conclusions'
    },
    {
      name: 'Key Insights',
      type: 'key_insights',
      description: 'Extract important takeaways and actionable insights'
    },
    {
      name: 'Action Items',
      type: 'custom',
      prompt: 'Extract all action items, tasks, and next steps mentioned in this transcription. Format as a numbered list with clear, actionable items.',
      description: 'Identify specific tasks and action items mentioned'
    },
    {
      name: 'Decision Points',
      type: 'custom',
      prompt: 'Identify all decisions that were made or need to be made based on this transcription. Include context and rationale where available.',
      description: 'Highlight decisions made and pending decisions'
    },
    {
      name: 'Questions Raised',
      type: 'custom',
      prompt: 'Extract all questions that were asked or raised during this conversation, including both answered and unanswered questions.',
      description: 'Catalog questions discussed in the conversation'
    },
    {
      name: 'Technical Details',
      type: 'custom',
      prompt: 'Focus on technical concepts, specifications, and implementation details mentioned in this transcription.',
      description: 'Extract technical information and specifications'
    }
  ];
}

// Private helper functions

/**
 * Validates analysis request
 */
function validateAnalysisRequest(request: AnalysisRequest): string | null {
  if (!request.transcriptionId) {
    return 'Transcription ID is required';
  }
  
  if (!request.analysisType) {
    return 'Analysis type is required';
  }
  
  const validTypes = ['summary', 'key_insights', 'custom'];
  if (!validTypes.includes(request.analysisType)) {
    return `Invalid analysis type: ${request.analysisType}`;
  }
  
  if (request.analysisType === 'custom' && !request.customPrompt) {
    return 'Custom prompt is required for custom analysis';
  }
  
  if (request.customPrompt && request.customPrompt.length > 2000) {
    return 'Custom prompt must be less than 2000 characters';
  }
  
  return null;
}

/**
 * Generates a session ID
 */
function generateSessionId(): string {
  return `analysis-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

/**
 * Sets loading state in UI
 */
function setAnalysisLoading(loading: boolean): void {
  // This would integrate with Windmill's state management
  console.log('Analysis loading:', loading);
}

/**
 * Updates UI with analysis results
 */
function updateAnalysisUI(result: AnalysisResponse): void {
  // This would integrate with Windmill's state management
  console.log('Analysis completed:', {
    type: result.analysisType,
    processingTime: result.processingTime,
    tokensUsed: result.tokensUsed
  });
}

/**
 * Adds analysis to local history
 */
function addToAnalysisHistory(analysis: AnalysisHistory): void {
  // Remove duplicate if exists
  const index = analysisHistory.findIndex(a => a.id === analysis.id);
  if (index > -1) {
    analysisHistory.splice(index, 1);
  }
  
  // Add to beginning
  analysisHistory.unshift(analysis);
  
  // Keep only recent analyses
  if (analysisHistory.length > 50) {
    analysisHistory = analysisHistory.slice(0, 50);
  }
}

/**
 * Gets analysis by ID
 */
async function getAnalysisById(id: string): Promise<AnalysisHistory> {
  const response = await fetch(`http://localhost:3000/api/analyses/${id}`, {
    method: 'GET',
    headers: {
      'Accept': 'application/json'
    }
  });
  
  if (!response.ok) {
    throw new Error(`Analysis not found: ${id}`);
  }
  
  const analysis = await response.json();
  return {
    id: analysis.id,
    transcriptionId: analysis.transcription_id,
    analysisType: analysis.analysis_type,
    result: analysis.result_text,
    createdAt: analysis.created_at,
    filename: analysis.transcription?.filename || 'Unknown'
  };
}

/**
 * Formats analysis as plain text
 */
function formatAnalysisAsText(analysis: AnalysisResponse): string {
  return `
AI Analysis Report
==================

File: ${analysis.filename}
Analysis Type: ${analysis.analysisType}
AI Model: ${analysis.aiModel}
Generated: ${new Date(analysis.createdAt).toLocaleString()}
Processing Time: ${analysis.processingTime}ms
Tokens Used: ${analysis.tokensUsed}
Confidence: ${Math.round(analysis.confidence * 100)}%

Analysis Result
---------------

${analysis.result}

---
Generated by Podcast Transcription Assistant
Analysis ID: ${analysis.analysisId}
`.trim();
}