/**
 * Smart File Photo Manager - Embeddings Generation Script
 * Generates semantic embeddings for files and content chunks using Ollama
 */

import { Resource } from "windmill-client";

interface OllamaResource {
  base_url: string;
  api_key?: string;
}

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

interface EmbeddingGenerationInput {
  fileId: string;
  processContentChunks?: boolean;
  batchSize?: number;
}

interface EmbeddingResult {
  fileEmbeddingGenerated: boolean;
  chunkEmbeddingsGenerated: number;
  totalEmbeddings: number;
  processingTime: number;
}

interface FileData {
  id: string;
  filename: string;
  fileType: string;
  description?: string;
  ocrText?: string;
  tags: string[];
  folderPath: string;
}

interface ContentChunk {
  id: string;
  fileId: string;
  chunkIndex: number;
  content: string;
  chunkType: string;
  pageNumber?: number;
}

export async function main(
  input: EmbeddingGenerationInput,
  ollama: Resource<"ollama"> = "f/file_manager/ollama",
  qdrant: Resource<"qdrant"> = "f/file_manager/qdrant",
  postgres: Resource<"postgres"> = "f/file_manager/postgres"
): Promise<EmbeddingResult> {
  
  console.log(`Generating embeddings for file: ${input.fileId}`);
  const startTime = Date.now();
  
  try {
    const batchSize = input.batchSize || 10;
    let totalEmbeddings = 0;
    
    // Step 1: Get file data from database
    const fileData = await getFileData(postgres as PostgresResource, input.fileId);
    
    if (!fileData) {
      throw new Error(`File not found: ${input.fileId}`);
    }
    
    // Step 2: Generate and store file-level embedding
    const fileEmbeddingGenerated = await generateFileEmbedding(
      ollama as OllamaResource,
      qdrant as QdrantResource,
      postgres as PostgresResource,
      fileData
    );
    
    if (fileEmbeddingGenerated) totalEmbeddings++;
    
    // Step 3: Generate embeddings for content chunks (if requested and applicable)
    let chunkEmbeddingsGenerated = 0;
    
    if (input.processContentChunks && (fileData.fileType === 'document' || fileData.ocrText)) {
      const chunks = await getContentChunks(postgres as PostgresResource, input.fileId);
      
      if (chunks.length > 0) {
        chunkEmbeddingsGenerated = await generateChunkEmbeddings(
          ollama as OllamaResource,
          qdrant as QdrantResource,
          postgres as PostgresResource,
          chunks,
          batchSize
        );
        totalEmbeddings += chunkEmbeddingsGenerated;
      }
    }
    
    // Step 4: Update file processing status
    await updateFileProcessingStatus(
      postgres as PostgresResource,
      input.fileId,
      'embedded'
    );
    
    const processingTime = Date.now() - startTime;
    
    const result: EmbeddingResult = {
      fileEmbeddingGenerated,
      chunkEmbeddingsGenerated,
      totalEmbeddings,
      processingTime
    };
    
    console.log(`Embedding generation completed for ${input.fileId}: ${totalEmbeddings} embeddings in ${processingTime}ms`);
    return result;
    
  } catch (error) {
    console.error(`Error generating embeddings for ${input.fileId}:`, error);
    throw error;
  }
}

async function getFileData(postgres: PostgresResource, fileId: string): Promise<FileData | null> {
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
        id, original_name as filename, file_type, description, 
        ocr_text, tags, folder_path
      FROM files 
      WHERE id = $1
    `;
    
    const result = await client.query(query, [fileId]);
    
    if (result.rows.length === 0) return null;
    
    const row = result.rows[0];
    return {
      id: row.id,
      filename: row.filename,
      fileType: row.file_type,
      description: row.description,
      ocrText: row.ocr_text,
      tags: row.tags || [],
      folderPath: row.folder_path
    };
    
  } finally {
    await client.end();
  }
}

async function getContentChunks(postgres: PostgresResource, fileId: string): Promise<ContentChunk[]> {
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
      SELECT id, file_id, chunk_index, content, chunk_type, page_number
      FROM content_chunks 
      WHERE file_id = $1 
      ORDER BY chunk_index
    `;
    
    const result = await client.query(query, [fileId]);
    
    return result.rows.map(row => ({
      id: row.id,
      fileId: row.file_id,
      chunkIndex: row.chunk_index,
      content: row.content,
      chunkType: row.chunk_type,
      pageNumber: row.page_number
    }));
    
  } finally {
    await client.end();
  }
}

async function generateFileEmbedding(
  ollama: OllamaResource,
  qdrant: QdrantResource,
  postgres: PostgresResource,
  fileData: FileData
): Promise<boolean> {
  
  try {
    // Prepare content for embedding
    const contentParts = [
      `Filename: ${fileData.filename}`,
      `Type: ${fileData.fileType}`,
      `Location: ${fileData.folderPath}`
    ];
    
    if (fileData.description) {
      contentParts.push(`Description: ${fileData.description}`);
    }
    
    if (fileData.ocrText) {
      // Use first 1000 characters of extracted text
      const textPreview = fileData.ocrText.slice(0, 1000);
      contentParts.push(`Content: ${textPreview}`);
    }
    
    if (fileData.tags && fileData.tags.length > 0) {
      contentParts.push(`Tags: ${fileData.tags.join(', ')}`);
    }
    
    const content = contentParts.join('\n');
    
    // Generate embedding
    const embedding = await generateEmbedding(ollama, content);
    
    // Store in PostgreSQL
    await storeFileEmbedding(postgres, fileData.id, embedding);
    
    // Store in Qdrant
    await storeFileEmbeddingInQdrant(qdrant, fileData, embedding);
    
    return true;
    
  } catch (error) {
    console.error(`Error generating file embedding for ${fileData.id}:`, error);
    return false;
  }
}

async function generateChunkEmbeddings(
  ollama: OllamaResource,
  qdrant: QdrantResource,
  postgres: PostgresResource,
  chunks: ContentChunk[],
  batchSize: number
): Promise<number> {
  
  let processedCount = 0;
  
  for (let i = 0; i < chunks.length; i += batchSize) {
    const batch = chunks.slice(i, i + batchSize);
    
    try {
      // Process batch in parallel
      const batchPromises = batch.map(async (chunk) => {
        try {
          const embedding = await generateEmbedding(ollama, chunk.content);
          
          // Store embedding in PostgreSQL
          await storeChunkEmbedding(postgres, chunk.id, embedding);
          
          // Store embedding in Qdrant
          await storeChunkEmbeddingInQdrant(qdrant, chunk, embedding);
          
          return true;
        } catch (error) {
          console.error(`Error processing chunk ${chunk.id}:`, error);
          return false;
        }
      });
      
      const results = await Promise.all(batchPromises);
      processedCount += results.filter(r => r).length;
      
      console.log(`Processed batch ${Math.floor(i / batchSize) + 1}: ${results.filter(r => r).length}/${batch.length} successful`);
      
      // Small delay between batches to avoid overwhelming the API
      if (i + batchSize < chunks.length) {
        await new Promise(resolve => setTimeout(resolve, 100));
      }
      
    } catch (error) {
      console.error(`Error processing batch starting at ${i}:`, error);
    }
  }
  
  return processedCount;
}

async function generateEmbedding(ollama: OllamaResource, content: string): Promise<number[]> {
  try {
    const response = await fetch(`${ollama.base_url}/api/embeddings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        model: 'nomic-embed-text',
        prompt: content
      })
    });
    
    if (!response.ok) {
      throw new Error(`Embedding API error: ${response.statusText}`);
    }
    
    const result = await response.json();
    return result.embedding;
    
  } catch (error) {
    console.error('Error generating embedding:', error);
    throw error;
  }
}

async function storeFileEmbedding(
  postgres: PostgresResource,
  fileId: string,
  embedding: number[]
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
      UPDATE files 
      SET content_embedding = $2
      WHERE id = $1
    `;
    
    await client.query(query, [fileId, `[${embedding.join(',')}]`]);
    
  } finally {
    await client.end();
  }
}

async function storeChunkEmbedding(
  postgres: PostgresResource,
  chunkId: string,
  embedding: number[]
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
      UPDATE content_chunks 
      SET embedding = $2
      WHERE id = $1
    `;
    
    await client.query(query, [chunkId, `[${embedding.join(',')}]`]);
    
  } finally {
    await client.end();
  }
}

async function storeFileEmbeddingInQdrant(
  qdrant: QdrantResource,
  fileData: FileData,
  embedding: number[]
): Promise<void> {
  
  try {
    const response = await fetch(`${qdrant.url}/collections/file_embeddings/points`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        points: [{
          id: fileData.id,
          vector: embedding,
          payload: {
            file_id: fileData.id,
            filename: fileData.filename,
            file_type: fileData.fileType,
            folder_path: fileData.folderPath,
            tags: fileData.tags,
            has_description: !!fileData.description,
            has_content: !!fileData.ocrText,
            indexed_at: new Date().toISOString()
          }
        }]
      })
    });
    
    if (!response.ok) {
      throw new Error(`Qdrant file embedding error: ${response.statusText}`);
    }
    
  } catch (error) {
    console.error('Error storing file embedding in Qdrant:', error);
    throw error;
  }
}

async function storeChunkEmbeddingInQdrant(
  qdrant: QdrantResource,
  chunk: ContentChunk,
  embedding: number[]
): Promise<void> {
  
  try {
    const response = await fetch(`${qdrant.url}/collections/content_chunks/points`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        points: [{
          id: chunk.id,
          vector: embedding,
          payload: {
            file_id: chunk.fileId,
            chunk_index: chunk.chunkIndex,
            chunk_type: chunk.chunkType,
            page_number: chunk.pageNumber,
            content_length: chunk.content.length,
            indexed_at: new Date().toISOString()
          }
        }]
      })
    });
    
    if (!response.ok) {
      throw new Error(`Qdrant chunk embedding error: ${response.statusText}`);
    }
    
  } catch (error) {
    console.error('Error storing chunk embedding in Qdrant:', error);
    throw error;
  }
}

async function updateFileProcessingStatus(
  postgres: PostgresResource,
  fileId: string,
  stage: string
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
      UPDATE files 
      SET processing_stage = $2, processed_at = CURRENT_TIMESTAMP
      WHERE id = $1
    `;
    
    await client.query(query, [fileId, stage]);
    
  } finally {
    await client.end();
  }
}