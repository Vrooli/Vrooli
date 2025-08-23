/**
 * Smart File Photo Manager - Organization Suggestion Script
 * Generates AI-powered suggestions for file organization, tagging, and folder structure
 */

import { Resource } from "windmill-client";

interface OllamaResource {
  base_url: string;
  api_key?: string;
}

interface PostgresResource {
  host: string;
  port: number;
  database: string;
  username: string;
  password: string;
}

interface OrganizationInput {
  fileId?: string; // If provided, generate suggestions for this specific file
  scanUnorganized?: boolean; // If true, process files in root directory
  folderPath?: string; // If provided, process files in this folder
  batchSize?: number; // Number of files to process at once
}

interface OrganizationResult {
  processedFiles: number;
  suggestionsGenerated: number;
  folderSuggestions: number;
  tagSuggestions: number;
  renameSuggestions: number;
  processingTime: number;
}

interface FileInfo {
  id: string;
  filename: string;
  currentPath: string;
  fileType: string;
  size: number;
  description?: string;
  ocrText?: string;
  detectedObjects?: string[];
  tags: string[];
  uploadedAt: string;
}

interface OrganizationSuggestion {
  type: 'folder' | 'tag' | 'rename';
  suggestedValue: string;
  reason: string;
  confidence: number;
  currentValue?: string;
}

export async function main(
  input: OrganizationInput,
  ollama: Resource<"ollama"> = "f/file_manager/ollama",
  postgres: Resource<"postgres"> = "f/file_manager/postgres"
): Promise<OrganizationResult> {
  
  console.log('Starting organization suggestion generation:', input);
  const startTime = Date.now();
  
  try {
    const batchSize = input.batchSize || 10;
    let files: FileInfo[] = [];
    
    // Get files to process based on input parameters
    if (input.fileId) {
      const file = await getFileById(postgres as PostgresResource, input.fileId);
      if (file) files = [file];
    } else if (input.scanUnorganized) {
      files = await getUnorganizedFiles(postgres as PostgresResource);
    } else if (input.folderPath) {
      files = await getFilesInFolder(postgres as PostgresResource, input.folderPath);
    } else {
      throw new Error('Must specify fileId, scanUnorganized, or folderPath');
    }
    
    if (files.length === 0) {
      return {
        processedFiles: 0,
        suggestionsGenerated: 0,
        folderSuggestions: 0,
        tagSuggestions: 0,
        renameSuggestions: 0,
        processingTime: Date.now() - startTime
      };
    }
    
    console.log(`Processing ${files.length} files for organization suggestions`);
    
    let totalSuggestions = 0;
    let folderSuggestions = 0;
    let tagSuggestions = 0;
    let renameSuggestions = 0;
    
    // Process files in batches
    for (let i = 0; i < files.length; i += batchSize) {
      const batch = files.slice(i, i + batchSize);
      
      const batchResults = await Promise.all(
        batch.map(file => generateSuggestionsForFile(
          ollama as OllamaResource,
          postgres as PostgresResource,
          file
        ))
      );
      
      // Count suggestions by type
      for (const result of batchResults) {
        totalSuggestions += result.length;
        folderSuggestions += result.filter(s => s.type === 'folder').length;
        tagSuggestions += result.filter(s => s.type === 'tag').length;
        renameSuggestions += result.filter(s => s.type === 'rename').length;
      }
      
      console.log(`Processed batch ${Math.floor(i / batchSize) + 1}/${Math.ceil(files.length / batchSize)}`);
    }
    
    const processingTime = Date.now() - startTime;
    
    const result: OrganizationResult = {
      processedFiles: files.length,
      suggestionsGenerated: totalSuggestions,
      folderSuggestions,
      tagSuggestions,
      renameSuggestions,
      processingTime
    };
    
    console.log(`Organization suggestions completed:`, result);
    return result;
    
  } catch (error) {
    console.error('Error generating organization suggestions:', error);
    throw error;
  }
}

async function getFileById(postgres: PostgresResource, fileId: string): Promise<FileInfo | null> {
  const { Client } = await import('pg');
  const client = new Client({
    host: postgres.host,
    port: postgres.port,
    database: postgres.database,
    user: postgres.username,
    password: postgres.password
  });
  
  try {
    await client.connect();
    
    const query = `
      SELECT 
        id, original_name as filename, folder_path as current_path,
        file_type, size_bytes as size, description, ocr_text,
        detected_objects, tags, uploaded_at
      FROM files 
      WHERE id = $1
    `;
    
    const result = await client.query(query, [fileId]);
    
    if (result.rows.length === 0) return null;
    
    const row = result.rows[0];
    return {
      id: row.id,
      filename: row.filename,
      currentPath: row.current_path,
      fileType: row.file_type,
      size: parseInt(row.size),
      description: row.description,
      ocrText: row.ocr_text,
      detectedObjects: row.detected_objects ? JSON.parse(row.detected_objects) : [],
      tags: row.tags || [],
      uploadedAt: row.uploaded_at
    };
    
  } finally {
    await client.end();
  }
}

async function getUnorganizedFiles(postgres: PostgresResource): Promise<FileInfo[]> {
  const { Client } = await import('pg');
  const client = new Client({
    host: postgres.host,
    port: postgres.port,
    database: postgres.database,
    user: postgres.username,
    password: postgres.password
  });
  
  try {
    await client.connect();
    
    const query = `
      SELECT 
        id, original_name as filename, folder_path as current_path,
        file_type, size_bytes as size, description, ocr_text,
        detected_objects, tags, uploaded_at
      FROM files 
      WHERE folder_path = '/' 
        OR array_length(tags, 1) IS NULL 
        OR array_length(tags, 1) = 0
        OR description IS NULL
      ORDER BY uploaded_at DESC
      LIMIT 100
    `;
    
    const result = await client.query(query);
    
    return result.rows.map(row => ({
      id: row.id,
      filename: row.filename,
      currentPath: row.current_path,
      fileType: row.file_type,
      size: parseInt(row.size),
      description: row.description,
      ocrText: row.ocr_text,
      detectedObjects: row.detected_objects ? JSON.parse(row.detected_objects) : [],
      tags: row.tags || [],
      uploadedAt: row.uploaded_at
    }));
    
  } finally {
    await client.end();
  }
}

async function getFilesInFolder(postgres: PostgresResource, folderPath: string): Promise<FileInfo[]> {
  const { Client } = await import('pg');
  const client = new Client({
    host: postgres.host,
    port: postgres.port,
    database: postgres.database,
    user: postgres.username,
    password: postgres.password
  });
  
  try {
    await client.connect();
    
    const query = `
      SELECT 
        id, original_name as filename, folder_path as current_path,
        file_type, size_bytes as size, description, ocr_text,
        detected_objects, tags, uploaded_at
      FROM files 
      WHERE folder_path = $1
      ORDER BY uploaded_at DESC
    `;
    
    const result = await client.query(query, [folderPath]);
    
    return result.rows.map(row => ({
      id: row.id,
      filename: row.filename,
      currentPath: row.current_path,
      fileType: row.file_type,
      size: parseInt(row.size),
      description: row.description,
      ocrText: row.ocr_text,
      detectedObjects: row.detected_objects ? JSON.parse(row.detected_objects) : [],
      tags: row.tags || [],
      uploadedAt: row.uploaded_at
    }));
    
  } finally {
    await client.end();
  }
}

async function generateSuggestionsForFile(
  ollama: OllamaResource,
  postgres: PostgresResource,
  file: FileInfo
): Promise<OrganizationSuggestion[]> {
  
  try {
    const suggestions: OrganizationSuggestion[] = [];
    
    // Build context for AI analysis
    const context = buildFileContext(file);
    
    // Generate folder suggestions
    const folderSuggestion = await generateFolderSuggestion(ollama, file, context);
    if (folderSuggestion) suggestions.push(folderSuggestion);
    
    // Generate tag suggestions
    const tagSuggestions = await generateTagSuggestions(ollama, file, context);
    suggestions.push(...tagSuggestions);
    
    // Generate rename suggestions
    const renameSuggestion = await generateRenameSuggestion(ollama, file, context);
    if (renameSuggestion) suggestions.push(renameSuggestion);
    
    // Store suggestions in database
    await storeSuggestions(postgres, file.id, suggestions);
    
    return suggestions;
    
  } catch (error) {
    console.error(`Error generating suggestions for file ${file.id}:`, error);
    return [];
  }
}

function buildFileContext(file: FileInfo): string {
  const contextParts = [
    `Filename: ${file.filename}`,
    `File type: ${file.fileType}`,
    `Current location: ${file.currentPath}`,
    `File size: ${(file.size / 1024 / 1024).toFixed(2)} MB`,
    `Upload date: ${file.uploadedAt}`
  ];
  
  if (file.description) {
    contextParts.push(`Description: ${file.description}`);
  }
  
  if (file.ocrText) {
    const textPreview = file.ocrText.slice(0, 500);
    contextParts.push(`Content: ${textPreview}${file.ocrText.length > 500 ? '...' : ''}`);
  }
  
  if (file.detectedObjects && file.detectedObjects.length > 0) {
    contextParts.push(`Detected objects: ${file.detectedObjects.join(', ')}`);
  }
  
  if (file.tags && file.tags.length > 0) {
    contextParts.push(`Current tags: ${file.tags.join(', ')}`);
  }
  
  return contextParts.join('\n');
}

async function generateFolderSuggestion(
  ollama: OllamaResource,
  file: FileInfo,
  context: string
): Promise<OrganizationSuggestion | null> {
  
  if (file.currentPath !== '/') {
    // File is already organized, check if current location is optimal
    const reviewPrompt = `
Given this file information:
${context}

Is the current folder location "${file.currentPath}" optimal for this file? 
If not, suggest a better folder path. Consider:
- Content type and purpose
- Semantic organization
- Common file organization patterns

Respond with either "KEEP_CURRENT" or suggest a new path starting with "/".
Be specific and practical. Example: "/Documents/Reports" or "/Photos/2024/Vacation"
`;
  } else {
    // File is unorganized, suggest appropriate folder
    const prompt = `
Given this file information:
${context}

Suggest an appropriate folder path for organizing this file. Consider:
- File type and content
- Business vs personal context
- Date-based organization if relevant
- Semantic categories

Respond with only the folder path starting with "/". Be specific and practical.
Examples: "/Documents/Reports", "/Photos/2024/Family", "/Work/Projects/WebsiteRedesign"
`;
    
    try {
      const response = await fetch(`${ollama.base_url}/api/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          model: 'llama3.2',
          prompt,
          stream: false,
          options: {
            temperature: 0.1,
            top_p: 0.9
          }
        })
      });
      
      if (!response.ok) return null;
      
      const result = await response.json();
      const suggestedPath = result.response.trim();
      
      if (suggestedPath && suggestedPath.startsWith('/') && suggestedPath !== file.currentPath) {
        return {
          type: 'folder',
          suggestedValue: suggestedPath,
          reason: `Based on file content and type, this file would be better organized in ${suggestedPath}`,
          confidence: 0.8,
          currentValue: file.currentPath
        };
      }
      
    } catch (error) {
      console.error('Error generating folder suggestion:', error);
    }
  }
  
  return null;
}

async function generateTagSuggestions(
  ollama: OllamaResource,
  file: FileInfo,
  context: string
): Promise<OrganizationSuggestion[]> {
  
  const prompt = `
Given this file information:
${context}

Suggest 3-8 relevant tags for this file. Consider:
- Content and subject matter
- Context and purpose
- File type specific tags
- Searchability and organization

Current tags: ${file.tags.join(', ') || 'none'}

Provide new tags that would be useful but are not already assigned. 
Respond with comma-separated tags in lowercase. Focus on descriptive, searchable terms.
Examples: work, financial, presentation, vacation, family, project, quarterly, meeting
`;
  
  try {
    const response = await fetch(`${ollama.base_url}/api/generate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        model: 'llama3.2',
        prompt,
        stream: false,
        options: {
          temperature: 0.1,
          top_p: 0.9
        }
      })
    });
    
    if (!response.ok) return [];
    
    const result = await response.json();
    const suggestedTags = result.response
      .split(',')
      .map(tag => tag.trim().toLowerCase())
      .filter(tag => tag.length > 0 && !file.tags.includes(tag))
      .slice(0, 5); // Limit to 5 new tags
    
    return suggestedTags.map(tag => ({
      type: 'tag',
      suggestedValue: tag,
      reason: `This tag would improve searchability and organization based on file content`,
      confidence: 0.75
    }));
    
  } catch (error) {
    console.error('Error generating tag suggestions:', error);
    return [];
  }
}

async function generateRenameSuggestion(
  ollama: OllamaResource,
  file: FileInfo,
  context: string
): Promise<OrganizationSuggestion | null> {
  
  // Only suggest rename if filename is generic or unclear
  const genericPatterns = [
    /^untitled/i,
    /^document\d*\./i,
    /^image\d*\./i,
    /^img_\d+/i,
    /^scan\d*\./i,
    /^screenshot/i,
    /^new\s/i
  ];
  
  const isGeneric = genericPatterns.some(pattern => pattern.test(file.filename));
  const hasContent = file.description || file.ocrText;
  
  if (!isGeneric && hasContent) return null; // Filename seems descriptive enough
  
  const prompt = `
Given this file information:
${context}

Current filename: ${file.filename}

Suggest a more descriptive filename that:
- Reflects the actual content
- Follows good naming conventions
- Includes key identifying information
- Maintains the original file extension

Respond with only the suggested filename. Be specific but concise.
Examples: "Q4_Financial_Report_2024.pdf", "Team_Meeting_Notes_Jan15.docx", "Vacation_Beach_Photo.jpg"
`;
  
  try {
    const response = await fetch(`${ollama.base_url}/api/generate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        model: 'llama3.2',
        prompt,
        stream: false,
        options: {
          temperature: 0.1,
          top_p: 0.9
        }
      })
    });
    
    if (!response.ok) return null;
    
    const result = await response.json();
    const suggestedName = result.response.trim();
    
    if (suggestedName && suggestedName !== file.filename) {
      return {
        type: 'rename',
        suggestedValue: suggestedName,
        reason: `Suggested filename better reflects the file content and follows naming conventions`,
        confidence: 0.7,
        currentValue: file.filename
      };
    }
    
  } catch (error) {
    console.error('Error generating rename suggestion:', error);
  }
  
  return null;
}

async function storeSuggestions(
  postgres: PostgresResource,
  fileId: string,
  suggestions: OrganizationSuggestion[]
): Promise<void> {
  
  if (suggestions.length === 0) return;
  
  const { Client } = await import('pg');
  const client = new Client({
    host: postgres.host,
    port: postgres.port,
    database: postgres.database,
    user: postgres.username,
    password: postgres.password
  });
  
  try {
    await client.connect();
    
    // Remove existing suggestions for this file (for the types we're generating)
    await client.query(`
      DELETE FROM suggestions 
      WHERE file_id = $1 AND type IN ('folder', 'tag', 'rename')
    `, [fileId]);
    
    // Insert new suggestions
    for (const suggestion of suggestions) {
      await client.query(`
        INSERT INTO suggestions (
          file_id, type, status, suggested_value, current_value,
          reason, confidence, created_at
        ) VALUES ($1, $2, 'pending', $3, $4, $5, $6, CURRENT_TIMESTAMP)
      `, [
        fileId,
        suggestion.type,
        suggestion.suggestedValue,
        suggestion.currentValue || null,
        suggestion.reason,
        suggestion.confidence
      ]);
    }
    
    console.log(`Stored ${suggestions.length} suggestions for file ${fileId}`);
    
  } finally {
    await client.end();
  }
}