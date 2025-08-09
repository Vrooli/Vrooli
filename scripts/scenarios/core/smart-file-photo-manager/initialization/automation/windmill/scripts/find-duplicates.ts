/**
 * Smart File Photo Manager - Duplicate Detection Script
 * Finds potential duplicate files using hash comparison and embedding similarity
 */

import { Resource } from "windmill-client";

interface QdrantResource {
  url: string;
  api_key?: string;
}

interface PostgresResource {
  host: string;
  port: number;
  database: string;
  username: string;
  password: string;
}

interface DuplicateDetectionInput {
  fileId?: string; // If provided, find duplicates for this specific file
  scanAll?: boolean; // If true, scan all files for duplicates
  similarityThreshold?: number; // Threshold for semantic similarity (default: 0.85)
  exactOnly?: boolean; // If true, only find exact duplicates (hash-based)
}

interface DuplicateDetectionResult {
  exactDuplicates: DuplicateGroup[];
  similarFiles: DuplicateGroup[];
  totalFiles: number;
  duplicateGroups: number;
  processingTime: number;
}

interface DuplicateGroup {
  type: 'exact' | 'similar';
  confidence: number;
  files: DuplicateFile[];
  suggestedAction: string;
  estimatedSpaceSavings?: number;
}

interface DuplicateFile {
  id: string;
  filename: string;
  filePath: string;
  size: number;
  hash: string;
  uploadedAt: string;
  similarity?: number;
}

interface FileRecord {
  id: string;
  filename: string;
  filePath: string;
  size: number;
  hash: string;
  uploadedAt: string;
  embedding?: number[];
  fileType: string;
}

export async function main(
  input: DuplicateDetectionInput,
  qdrant: Resource<"qdrant"> = "f/file_manager/qdrant",
  postgres: Resource<"postgres"> = "f/file_manager/postgres"
): Promise<DuplicateDetectionResult> {
  
  console.log(`Starting duplicate detection with params:`, input);
  const startTime = Date.now();
  
  try {
    const similarityThreshold = input.similarityThreshold || 0.85;
    
    let exactDuplicates: DuplicateGroup[] = [];
    let similarFiles: DuplicateGroup[] = [];
    
    if (input.fileId) {
      // Find duplicates for specific file
      const results = await findDuplicatesForFile(
        postgres as PostgresResource,
        qdrant as QdrantResource,
        input.fileId,
        similarityThreshold,
        input.exactOnly || false
      );
      exactDuplicates = results.exact;
      similarFiles = results.similar;
      
    } else if (input.scanAll) {
      // Scan all files for duplicates
      const results = await findAllDuplicates(
        postgres as PostgresResource,
        qdrant as QdrantResource,
        similarityThreshold,
        input.exactOnly || false
      );
      exactDuplicates = results.exact;
      similarFiles = results.similar;
      
    } else {
      throw new Error('Either fileId or scanAll must be specified');
    }
    
    // Store duplicate relationships in database
    await storeDuplicateRelationships(
      postgres as PostgresResource,
      [...exactDuplicates, ...similarFiles]
    );
    
    // Generate suggestions for duplicate handling
    await generateDuplicateSuggestions(
      postgres as PostgresResource,
      [...exactDuplicates, ...similarFiles]
    );
    
    const processingTime = Date.now() - startTime;
    const totalFiles = exactDuplicates.reduce((sum, group) => sum + group.files.length, 0) +
                      similarFiles.reduce((sum, group) => sum + group.files.length, 0);
    
    const result: DuplicateDetectionResult = {
      exactDuplicates,
      similarFiles,
      totalFiles,
      duplicateGroups: exactDuplicates.length + similarFiles.length,
      processingTime
    };
    
    console.log(`Duplicate detection completed: ${result.duplicateGroups} groups, ${totalFiles} files in ${processingTime}ms`);
    return result;
    
  } catch (error) {
    console.error('Error in duplicate detection:', error);
    throw error;
  }
}

async function findDuplicatesForFile(
  postgres: PostgresResource,
  qdrant: QdrantResource,
  fileId: string,
  similarityThreshold: number,
  exactOnly: boolean
): Promise<{ exact: DuplicateGroup[], similar: DuplicateGroup[] }> {
  
  // Get the target file information
  const targetFile = await getFileById(postgres, fileId);
  if (!targetFile) {
    throw new Error(`File not found: ${fileId}`);
  }
  
  const exact: DuplicateGroup[] = [];
  const similar: DuplicateGroup[] = [];
  
  // Find exact duplicates by hash
  const exactDuplicates = await findExactDuplicatesByHash(postgres, targetFile.hash, fileId);
  
  if (exactDuplicates.length > 0) {
    const spaceSavings = exactDuplicates.reduce((sum, file) => sum + file.size, 0);
    
    exact.push({
      type: 'exact',
      confidence: 1.0,
      files: [
        {
          id: targetFile.id,
          filename: targetFile.filename,
          filePath: targetFile.filePath,
          size: targetFile.size,
          hash: targetFile.hash,
          uploadedAt: targetFile.uploadedAt
        },
        ...exactDuplicates
      ],
      suggestedAction: 'merge_or_delete',
      estimatedSpaceSavings: spaceSavings
    });
  }
  
  // Find similar files by embedding (if not exactOnly and file has embedding)
  if (!exactOnly && targetFile.embedding) {
    const similarFiles = await findSimilarFilesByEmbedding(
      qdrant,
      postgres,
      targetFile.id,
      targetFile.embedding,
      similarityThreshold
    );
    
    if (similarFiles.length > 0) {
      similar.push({
        type: 'similar',
        confidence: similarFiles[0].similarity || 0,
        files: [
          {
            id: targetFile.id,
            filename: targetFile.filename,
            filePath: targetFile.filePath,
            size: targetFile.size,
            hash: targetFile.hash,
            uploadedAt: targetFile.uploadedAt
          },
          ...similarFiles
        ],
        suggestedAction: 'review_and_organize'
      });
    }
  }
  
  return { exact, similar };
}

async function findAllDuplicates(
  postgres: PostgresResource,
  qdrant: QdrantResource,
  similarityThreshold: number,
  exactOnly: boolean
): Promise<{ exact: DuplicateGroup[], similar: DuplicateGroup[] }> {
  
  const exact: DuplicateGroup[] = [];
  const similar: DuplicateGroup[] = [];
  
  // Find all exact duplicates grouped by hash
  const exactGroups = await findAllExactDuplicates(postgres);
  
  for (const group of exactGroups) {
    const spaceSavings = group.files.slice(1).reduce((sum, file) => sum + file.size, 0);
    
    exact.push({
      type: 'exact',
      confidence: 1.0,
      files: group.files,
      suggestedAction: 'merge_or_delete',
      estimatedSpaceSavings: spaceSavings
    });
  }
  
  if (!exactOnly) {
    // Find similar files using batch embedding comparison
    const similarGroups = await findSimilarFilesInBatch(
      postgres,
      qdrant,
      similarityThreshold
    );
    
    similar.push(...similarGroups);
  }
  
  return { exact, similar };
}

async function getFileById(postgres: PostgresResource, fileId: string): Promise<FileRecord | null> {
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
        id, original_name as filename, folder_path as file_path, 
        size_bytes as size, file_hash as hash, uploaded_at,
        content_embedding as embedding, file_type
      FROM files 
      WHERE id = $1
    `;
    
    const result = await client.query(query, [fileId]);
    
    if (result.rows.length === 0) return null;
    
    const row = result.rows[0];
    return {
      id: row.id,
      filename: row.filename,
      filePath: row.file_path,
      size: parseInt(row.size),
      hash: row.hash,
      uploadedAt: row.uploaded_at,
      embedding: row.embedding ? JSON.parse(row.embedding) : undefined,
      fileType: row.file_type
    };
    
  } finally {
    await client.end();
  }
}

async function findExactDuplicatesByHash(
  postgres: PostgresResource, 
  hash: string, 
  excludeFileId: string
): Promise<DuplicateFile[]> {
  
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
        id, original_name as filename, folder_path as file_path,
        size_bytes as size, file_hash as hash, uploaded_at
      FROM files 
      WHERE file_hash = $1 AND id != $2
      ORDER BY uploaded_at DESC
    `;
    
    const result = await client.query(query, [hash, excludeFileId]);
    
    return result.rows.map(row => ({
      id: row.id,
      filename: row.filename,
      filePath: row.file_path,
      size: parseInt(row.size),
      hash: row.hash,
      uploadedAt: row.uploaded_at
    }));
    
  } finally {
    await client.end();
  }
}

async function findAllExactDuplicates(postgres: PostgresResource): Promise<{ files: DuplicateFile[] }[]> {
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
        id, original_name as filename, folder_path as file_path,
        size_bytes as size, file_hash as hash, uploaded_at
      FROM files 
      WHERE file_hash IN (
        SELECT file_hash 
        FROM files 
        GROUP BY file_hash 
        HAVING COUNT(*) > 1
      )
      ORDER BY file_hash, uploaded_at DESC
    `;
    
    const result = await client.query(query);
    
    // Group by hash
    const groups = new Map<string, DuplicateFile[]>();
    
    for (const row of result.rows) {
      const file: DuplicateFile = {
        id: row.id,
        filename: row.filename,
        filePath: row.file_path,
        size: parseInt(row.size),
        hash: row.hash,
        uploadedAt: row.uploaded_at
      };
      
      if (!groups.has(row.hash)) {
        groups.set(row.hash, []);
      }
      groups.get(row.hash)!.push(file);
    }
    
    return Array.from(groups.values()).map(files => ({ files }));
    
  } finally {
    await client.end();
  }
}

async function findSimilarFilesByEmbedding(
  qdrant: QdrantResource,
  postgres: PostgresResource,
  fileId: string,
  embedding: number[],
  threshold: number
): Promise<DuplicateFile[]> {
  
  try {
    // Search for similar embeddings in Qdrant
    const response = await fetch(`${qdrant.url}/collections/file_embeddings/points/search`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        vector: embedding,
        limit: 10,
        score_threshold: threshold,
        with_payload: true,
        filter: {
          must_not: [
            { key: "file_id", match: { value: fileId } }
          ]
        }
      })
    });
    
    if (!response.ok) {
      throw new Error(`Qdrant search error: ${response.statusText}`);
    }
    
    const searchResult = await response.json();
    
    if (!searchResult.result || searchResult.result.length === 0) {
      return [];
    }
    
    // Get detailed file information from PostgreSQL
    const fileIds = searchResult.result.map((hit: any) => hit.payload.file_id);
    const files = await getFilesByIds(postgres, fileIds);
    
    // Combine with similarity scores
    return searchResult.result.map((hit: any) => {
      const file = files.find(f => f.id === hit.payload.file_id);
      return file ? {
        ...file,
        similarity: hit.score
      } : null;
    }).filter(f => f !== null) as (DuplicateFile & { similarity: number })[];
    
  } catch (error) {
    console.error('Error finding similar files:', error);
    return [];
  }
}

async function findSimilarFilesInBatch(
  postgres: PostgresResource,
  qdrant: QdrantResource,
  threshold: number
): Promise<DuplicateGroup[]> {
  
  // This would implement batch similarity search
  // For now, return empty array as this is a complex operation
  console.log('Batch similarity search not implemented yet');
  return [];
}

async function getFilesByIds(postgres: PostgresResource, fileIds: string[]): Promise<DuplicateFile[]> {
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
        id, original_name as filename, folder_path as file_path,
        size_bytes as size, file_hash as hash, uploaded_at
      FROM files 
      WHERE id = ANY($1)
    `;
    
    const result = await client.query(query, [fileIds]);
    
    return result.rows.map(row => ({
      id: row.id,
      filename: row.filename,
      filePath: row.file_path,
      size: parseInt(row.size),
      hash: row.hash,
      uploadedAt: row.uploaded_at
    }));
    
  } finally {
    await client.end();
  }
}

async function storeDuplicateRelationships(
  postgres: PostgresResource,
  duplicateGroups: DuplicateGroup[]
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
    await client.query('BEGIN');
    
    // Clear existing relationships for files in these groups
    const allFileIds = duplicateGroups.flatMap(group => group.files.map(f => f.id));
    
    if (allFileIds.length > 0) {
      await client.query(
        'DELETE FROM file_relationships WHERE file_id = ANY($1) OR related_file_id = ANY($1)',
        [allFileIds]
      );
    }
    
    // Insert new relationships
    for (const group of duplicateGroups) {
      const files = group.files;
      
      for (let i = 0; i < files.length; i++) {
        for (let j = i + 1; j < files.length; j++) {
          await client.query(`
            INSERT INTO file_relationships (file_id, related_file_id, relationship_type, confidence)
            VALUES ($1, $2, $3, $4), ($4, $1, $3, $4)
          `, [files[i].id, files[j].id, group.type, group.confidence]);
        }
      }
    }
    
    await client.query('COMMIT');
    console.log(`Stored relationships for ${duplicateGroups.length} duplicate groups`);
    
  } catch (error) {
    await client.query('ROLLBACK');
    throw error;
  } finally {
    await client.end();
  }
}

async function generateDuplicateSuggestions(
  postgres: PostgresResource,
  duplicateGroups: DuplicateGroup[]
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
    
    for (const group of duplicateGroups) {
      const files = group.files;
      
      if (group.type === 'exact') {
        // Suggest keeping the newest file and removing others
        const newestFile = files.reduce((newest, current) => 
          new Date(current.uploadedAt) > new Date(newest.uploadedAt) ? current : newest
        );
        
        for (const file of files) {
          if (file.id !== newestFile.id) {
            await client.query(`
              INSERT INTO suggestions (file_id, type, status, suggested_value, reason, confidence, similar_file_ids)
              VALUES ($1, $2, 'pending', $3, $4, $5, $6)
            `, [
              file.id,
              'duplicate',
              'delete',
              `Exact duplicate of ${newestFile.filename}. Consider removing to save space.`,
              group.confidence,
              [newestFile.id]
            ]);
          }
        }
      } else if (group.type === 'similar') {
        // Suggest reviewing similar files for potential organization
        for (const file of files) {
          const otherFiles = files.filter(f => f.id !== file.id);
          
          await client.query(`
            INSERT INTO suggestions (file_id, type, status, suggested_value, reason, confidence, similar_file_ids, similarity_scores)
            VALUES ($1, $2, 'pending', $3, $4, $5, $6, $7)
          `, [
            file.id,
            'duplicate',
            'review',
            'Review similar files for potential organization or cleanup',
            `Found ${otherFiles.length} similar files that may be variants or related content`,
            group.confidence,
            otherFiles.map(f => f.id),
            otherFiles.map(f => (f as any).similarity || group.confidence)
          ]);
        }
      }
    }
    
    console.log(`Generated suggestions for ${duplicateGroups.length} duplicate groups`);
    
  } finally {
    await client.end();
  }
}