/**
 * Smart File Photo Manager - Batch Organization Script
 * Performs batch operations for organizing files based on AI suggestions and rules
 */

import { Resource } from "windmill-client";

interface PostgresResource {
  host: string;
  port: number;
  database: string;
  username: string;
  password: string;
}

interface MinioResource {
  endpoint: string;
  access_key: string;
  secret_key: string;
  region?: string;
}

interface BatchOrganizeInput {
  operation: 'organize_by_suggestions' | 'organize_by_type' | 'organize_by_date' | 'apply_tags' | 'cleanup_duplicates';
  fileIds?: string[]; // Specific files to organize
  folderPath?: string; // Organize all files in this folder
  filters?: {
    fileTypes?: string[];
    dateRange?: { start: string; end: string };
    minConfidence?: number;
  };
  dryRun?: boolean; // If true, return what would be done without executing
  autoApprove?: boolean; // If true, automatically approve high-confidence suggestions
  maxFiles?: number; // Limit number of files to process
}

interface BatchOrganizeResult {
  operation: string;
  totalFiles: number;
  processedFiles: number;
  successfulMoves: number;
  failedMoves: number;
  createdFolders: number;
  appliedTags: number;
  dryRun: boolean;
  changes: OrganizationChange[];
  processingTime: number;
  errors: string[];
}

interface OrganizationChange {
  fileId: string;
  filename: string;
  action: string;
  from?: string;
  to?: string;
  tags?: string[];
  success: boolean;
  error?: string;
}

interface FileToProcess {
  id: string;
  filename: string;
  currentPath: string;
  fileType: string;
  size: number;
  uploadedAt: string;
  suggestions?: Suggestion[];
}

interface Suggestion {
  id: string;
  type: string;
  suggestedValue: string;
  currentValue?: string;
  confidence: number;
  status: string;
}

export async function main(
  input: BatchOrganizeInput,
  postgres: Resource<"postgres"> = "f/file_manager/postgres",
  minio: Resource<"minio"> = "f/file_manager/minio"
): Promise<BatchOrganizeResult> {
  
  console.log(`Starting batch organization:`, input);
  const startTime = Date.now();
  
  try {
    const changes: OrganizationChange[] = [];
    const errors: string[] = [];
    let createdFolders = 0;
    let appliedTags = 0;
    
    // Get files to process
    const files = await getFilesToProcess(postgres as PostgresResource, input);
    
    if (files.length === 0) {
      return {
        operation: input.operation,
        totalFiles: 0,
        processedFiles: 0,
        successfulMoves: 0,
        failedMoves: 0,
        createdFolders: 0,
        appliedTags: 0,
        dryRun: input.dryRun || false,
        changes: [],
        processingTime: Date.now() - startTime,
        errors: ['No files found matching the criteria']
      };
    }
    
    console.log(`Found ${files.length} files to process`);
    
    // Process files based on operation type
    let processedFiles = 0;
    let successfulMoves = 0;
    let failedMoves = 0;
    
    for (const file of files) {
      try {
        const fileChanges = await processFile(
          postgres as PostgresResource,
          minio as MinioResource,
          file,
          input
        );
        
        changes.push(...fileChanges);
        processedFiles++;
        
        // Count different types of changes
        for (const change of fileChanges) {
          if (change.success) {
            if (change.action === 'move') successfulMoves++;
            if (change.action === 'create_folder') createdFolders++;
            if (change.action === 'apply_tags') appliedTags++;
          } else {
            if (change.action === 'move') failedMoves++;
            if (change.error) errors.push(`${file.filename}: ${change.error}`);
          }
        }
        
      } catch (error) {
        errors.push(`Error processing ${file.filename}: ${error instanceof Error ? error.message : String(error)}`);
        failedMoves++;
      }
    }
    
    const processingTime = Date.now() - startTime;
    
    const result: BatchOrganizeResult = {
      operation: input.operation,
      totalFiles: files.length,
      processedFiles,
      successfulMoves,
      failedMoves,
      createdFolders,
      appliedTags,
      dryRun: input.dryRun || false,
      changes,
      processingTime,
      errors
    };
    
    console.log(`Batch organization completed:`, result);
    return result;
    
  } catch (error) {
    console.error('Error in batch organization:', error);
    throw error;
  }
}

async function getFilesToProcess(
  postgres: PostgresResource,
  input: BatchOrganizeInput
): Promise<FileToProcess[]> {
  
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
    
    let whereClause = '1=1';
    const params: any[] = [];
    let paramIndex = 1;
    
    // Build query based on input filters
    if (input.fileIds && input.fileIds.length > 0) {
      whereClause += ` AND f.id = ANY($${paramIndex})`;
      params.push(input.fileIds);
      paramIndex++;
    }
    
    if (input.folderPath) {
      whereClause += ` AND f.folder_path = $${paramIndex}`;
      params.push(input.folderPath);
      paramIndex++;
    }
    
    if (input.filters?.fileTypes && input.filters.fileTypes.length > 0) {
      whereClause += ` AND f.file_type = ANY($${paramIndex})`;
      params.push(input.filters.fileTypes);
      paramIndex++;
    }
    
    if (input.filters?.dateRange) {
      whereClause += ` AND f.uploaded_at BETWEEN $${paramIndex} AND $${paramIndex + 1}`;
      params.push(input.filters.dateRange.start, input.filters.dateRange.end);
      paramIndex += 2;
    }
    
    const query = `
      SELECT 
        f.id, f.original_name as filename, f.folder_path as current_path,
        f.file_type, f.size_bytes as size, f.uploaded_at,
        COALESCE(
          json_agg(
            json_build_object(
              'id', s.id,
              'type', s.type,
              'suggested_value', s.suggested_value,
              'current_value', s.current_value,
              'confidence', s.confidence,
              'status', s.status
            )
          ) FILTER (WHERE s.id IS NOT NULL), 
          '[]'::json
        ) as suggestions
      FROM files f
      LEFT JOIN suggestions s ON f.id = s.file_id 
        AND s.status = 'pending'
        ${input.filters?.minConfidence ? `AND s.confidence >= $${paramIndex}` : ''}
      WHERE ${whereClause}
      GROUP BY f.id, f.original_name, f.folder_path, f.file_type, f.size_bytes, f.uploaded_at
      ORDER BY f.uploaded_at DESC
      ${input.maxFiles ? `LIMIT ${input.maxFiles}` : ''}
    `;
    
    if (input.filters?.minConfidence) {
      params.push(input.filters.minConfidence);
    }
    
    const result = await client.query(query, params);
    
    return result.rows.map(row => ({
      id: row.id,
      filename: row.filename,
      currentPath: row.current_path,
      fileType: row.file_type,
      size: parseInt(row.size),
      uploadedAt: row.uploaded_at,
      suggestions: Array.isArray(row.suggestions) ? row.suggestions : []
    }));
    
  } finally {
    await client.end();
  }
}

async function processFile(
  postgres: PostgresResource,
  minio: MinioResource,
  file: FileToProcess,
  input: BatchOrganizeInput
): Promise<OrganizationChange[]> {
  
  const changes: OrganizationChange[] = [];
  
  try {
    switch (input.operation) {
      case 'organize_by_suggestions':
        changes.push(...await organizeBysuggestions(postgres, minio, file, input));
        break;
      case 'organize_by_type':
        changes.push(...await organizeByType(postgres, minio, file, input));
        break;
      case 'organize_by_date':
        changes.push(...await organizeByDate(postgres, minio, file, input));
        break;
      case 'apply_tags':
        changes.push(...await applyTags(postgres, file, input));
        break;
      case 'cleanup_duplicates':
        changes.push(...await cleanupDuplicates(postgres, minio, file, input));
        break;
      default:
        throw new Error(`Unknown operation: ${input.operation}`);
    }
    
  } catch (error) {
    changes.push({
      fileId: file.id,
      filename: file.filename,
      action: 'error',
      success: false,
      error: error instanceof Error ? error.message : String(error)
    });
  }
  
  return changes;
}

async function organizeBysuggestions(
  postgres: PostgresResource,
  minio: MinioResource,
  file: FileToProcess,
  input: BatchOrganizeInput
): Promise<OrganizationChange[]> {
  
  const changes: OrganizationChange[] = [];
  
  if (!file.suggestions || file.suggestions.length === 0) {
    return changes;
  }
  
  // Process folder suggestions
  const folderSuggestions = file.suggestions.filter(s => s.type === 'folder');
  for (const suggestion of folderSuggestions) {
    if (input.autoApprove || suggestion.confidence >= (input.filters?.minConfidence || 0.8)) {
      const moveResult = await moveFile(postgres, minio, file, suggestion.suggestedValue, input.dryRun || false);
      changes.push(moveResult);
      
      if (moveResult.success && !input.dryRun) {
        await updateSuggestionStatus(postgres, suggestion.id, 'accepted');
      }
    }
  }
  
  // Process tag suggestions
  const tagSuggestions = file.suggestions.filter(s => s.type === 'tag');
  if (tagSuggestions.length > 0) {
    const newTags = tagSuggestions
      .filter(s => input.autoApprove || s.confidence >= (input.filters?.minConfidence || 0.7))
      .map(s => s.suggestedValue);
    
    if (newTags.length > 0) {
      const tagResult = await addTags(postgres, file.id, newTags, input.dryRun || false);
      changes.push(tagResult);
      
      if (tagResult.success && !input.dryRun) {
        // Mark tag suggestions as accepted
        for (const suggestion of tagSuggestions) {
          if (newTags.includes(suggestion.suggestedValue)) {
            await updateSuggestionStatus(postgres, suggestion.id, 'accepted');
          }
        }
      }
    }
  }
  
  // Process rename suggestions
  const renameSuggestions = file.suggestions.filter(s => s.type === 'rename');
  for (const suggestion of renameSuggestions) {
    if (input.autoApprove || suggestion.confidence >= (input.filters?.minConfidence || 0.8)) {
      const renameResult = await renameFile(postgres, minio, file, suggestion.suggestedValue, input.dryRun || false);
      changes.push(renameResult);
      
      if (renameResult.success && !input.dryRun) {
        await updateSuggestionStatus(postgres, suggestion.id, 'accepted');
      }
    }
  }
  
  return changes;
}

async function organizeByType(
  postgres: PostgresResource,
  minio: MinioResource,
  file: FileToProcess,
  input: BatchOrganizeInput
): Promise<OrganizationChange[]> {
  
  const typeToFolder = {
    'image': '/Photos',
    'document': '/Documents',
    'video': '/Videos',
    'audio': '/Audio',
    'archive': '/Archives',
    'other': '/Other'
  };
  
  const targetFolder = typeToFolder[file.fileType as keyof typeof typeToFolder] || '/Other';
  
  if (file.currentPath === targetFolder || file.currentPath.startsWith(targetFolder + '/')) {
    return []; // Already in correct location
  }
  
  const moveResult = await moveFile(postgres, minio, file, targetFolder, input.dryRun || false);
  return [moveResult];
}

async function organizeByDate(
  postgres: PostgresResource,
  minio: MinioResource,
  file: FileToProcess,
  input: BatchOrganizeInput
): Promise<OrganizationChange[]> {
  
  const uploadDate = new Date(file.uploadedAt);
  const year = uploadDate.getFullYear();
  const month = uploadDate.toLocaleString('default', { month: 'long' });
  
  const targetFolder = `/${file.fileType === 'image' ? 'Photos' : 'Files'}/${year}/${month}`;
  
  if (file.currentPath === targetFolder) {
    return []; // Already in correct location
  }
  
  const moveResult = await moveFile(postgres, minio, file, targetFolder, input.dryRun || false);
  return [moveResult];
}

async function applyTags(
  postgres: PostgresResource,
  file: FileToProcess,
  input: BatchOrganizeInput
): Promise<OrganizationChange[]> {
  
  const changes: OrganizationChange[] = [];
  
  // Apply suggested tags
  if (file.suggestions) {
    const tagSuggestions = file.suggestions.filter(s => s.type === 'tag');
    if (tagSuggestions.length > 0) {
      const newTags = tagSuggestions
        .filter(s => s.confidence >= (input.filters?.minConfidence || 0.7))
        .map(s => s.suggestedValue);
      
      if (newTags.length > 0) {
        const tagResult = await addTags(postgres, file.id, newTags, input.dryRun || false);
        changes.push(tagResult);
      }
    }
  }
  
  return changes;
}

async function cleanupDuplicates(
  postgres: PostgresResource,
  minio: MinioResource,
  file: FileToProcess,
  input: BatchOrganizeInput
): Promise<OrganizationChange[]> {
  
  // This would implement duplicate cleanup logic
  // For now, return empty array
  console.log(`Duplicate cleanup not yet implemented for file ${file.id}`);
  return [];
}

async function moveFile(
  postgres: PostgresResource,
  minio: MinioResource,
  file: FileToProcess,
  targetPath: string,
  dryRun: boolean
): Promise<OrganizationChange> {
  
  try {
    if (dryRun) {
      return {
        fileId: file.id,
        filename: file.filename,
        action: 'move',
        from: file.currentPath,
        to: targetPath,
        success: true
      };
    }
    
    // Create target folder if it doesn't exist
    await ensureFolderExists(postgres, targetPath);
    
    // Update file path in database
    const { Client } = await import('pg');
    const client = new Client({
      host: postgres.host,
      port: postgres.port,
      database: postgres.database,
      user: postgres.username,
      password: postgres.password
    });
    
    await client.connect();
    
    const query = `
      UPDATE files 
      SET folder_path = $1, updated_at = CURRENT_TIMESTAMP
      WHERE id = $2
    `;
    
    await client.query(query, [targetPath, file.id]);
    await client.end();
    
    return {
      fileId: file.id,
      filename: file.filename,
      action: 'move',
      from: file.currentPath,
      to: targetPath,
      success: true
    };
    
  } catch (error) {
    return {
      fileId: file.id,
      filename: file.filename,
      action: 'move',
      from: file.currentPath,
      to: targetPath,
      success: false,
      error: error instanceof Error ? error.message : String(error)
    };
  }
}

async function renameFile(
  postgres: PostgresResource,
  minio: MinioResource,
  file: FileToProcess,
  newName: string,
  dryRun: boolean
): Promise<OrganizationChange> {
  
  try {
    if (dryRun) {
      return {
        fileId: file.id,
        filename: file.filename,
        action: 'rename',
        from: file.filename,
        to: newName,
        success: true
      };
    }
    
    // Update filename in database
    const { Client } = await import('pg');
    const client = new Client({
      host: postgres.host,
      port: postgres.port,
      database: postgres.database,
      user: postgres.username,
      password: postgres.password
    });
    
    await client.connect();
    
    const query = `
      UPDATE files 
      SET current_name = $1, updated_at = CURRENT_TIMESTAMP
      WHERE id = $2
    `;
    
    await client.query(query, [newName, file.id]);
    await client.end();
    
    return {
      fileId: file.id,
      filename: file.filename,
      action: 'rename',
      from: file.filename,
      to: newName,
      success: true
    };
    
  } catch (error) {
    return {
      fileId: file.id,
      filename: file.filename,
      action: 'rename',
      from: file.filename,
      to: newName,
      success: false,
      error: error instanceof Error ? error.message : String(error)
    };
  }
}

async function addTags(
  postgres: PostgresResource,
  fileId: string,
  newTags: string[],
  dryRun: boolean
): Promise<OrganizationChange> {
  
  try {
    if (dryRun) {
      return {
        fileId,
        filename: '',
        action: 'apply_tags',
        tags: newTags,
        success: true
      };
    }
    
    const { Client } = await import('pg');
    const client = new Client({
      host: postgres.host,
      port: postgres.port,
      database: postgres.database,
      user: postgres.username,
      password: postgres.password
    });
    
    await client.connect();
    
    const query = `
      UPDATE files 
      SET tags = array(SELECT DISTINCT unnest(COALESCE(tags, ARRAY[]::text[]) || $1::text[])),
          updated_at = CURRENT_TIMESTAMP
      WHERE id = $2
    `;
    
    await client.query(query, [newTags, fileId]);
    await client.end();
    
    return {
      fileId,
      filename: '',
      action: 'apply_tags',
      tags: newTags,
      success: true
    };
    
  } catch (error) {
    return {
      fileId,
      filename: '',
      action: 'apply_tags',
      tags: newTags,
      success: false,
      error: error instanceof Error ? error.message : String(error)
    };
  }
}

async function ensureFolderExists(postgres: PostgresResource, folderPath: string): Promise<void> {
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
      INSERT INTO folders (name, path, parent_path, is_smart, created_at)
      VALUES ($1, $2, $3, false, CURRENT_TIMESTAMP)
      ON CONFLICT (path) DO NOTHING
    `;
    
    const pathParts = folderPath.split('/').filter(p => p.length > 0);
    const folderName = pathParts[pathParts.length - 1];
    const parentPath = pathParts.length > 1 ? '/' + pathParts.slice(0, -1).join('/') : '/';
    
    await client.query(query, [folderName, folderPath, parentPath]);
    
  } finally {
    await client.end();
  }
}

async function updateSuggestionStatus(
  postgres: PostgresResource,
  suggestionId: string,
  status: string
): Promise<void> {
  
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
      UPDATE suggestions 
      SET status = $1, applied_at = CURRENT_TIMESTAMP
      WHERE id = $2
    `;
    
    await client.query(query, [status, suggestionId]);
    
  } finally {
    await client.end();
  }
}