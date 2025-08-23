/**
 * Smart File Photo Manager - Text Extraction Script
 * Extracts and processes text from documents using Unstructured.io
 */

import { Resource } from "windmill-client";

interface UnstructuredResource {
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

interface DocumentElement {
  type: string;
  text: string;
  metadata: {
    page_number?: number;
    filename?: string;
    file_directory?: string;
    coordinates?: any;
  };
}

interface TextExtractionResult {
  fullText: string;
  chunks: ContentChunk[];
  metadata: DocumentMetadata;
  language: string;
  entities: ExtractedEntity[];
  documentType: string;
  summary: string;
}

interface ContentChunk {
  index: number;
  content: string;
  type: string;
  pageNumber?: number;
  startChar: number;
  endChar: number;
}

interface DocumentMetadata {
  pageCount: number;
  wordCount: number;
  characterCount: number;
  hasImages: boolean;
  hasTables: boolean;
  language: string;
  author?: string;
  title?: string;
  createdDate?: string;
}

interface ExtractedEntity {
  text: string;
  type: string;
  confidence: number;
  startChar: number;
  endChar: number;
}

interface TextExtractionInput {
  fileId: string;
  filePath: string;
  filename: string;
  mimeType: string;
  fileSize: number;
}

export async function main(
  input: TextExtractionInput,
  unstructured: Resource<"unstructured"> = "f/file_manager/unstructured",
  postgres: Resource<"postgres"> = "f/file_manager/postgres"
): Promise<TextExtractionResult> {
  
  console.log(`Extracting text from: ${input.filename} (${input.fileId})`);
  
  try {
    // Step 1: Extract document elements using Unstructured.io
    const elements = await extractDocumentElements(
      unstructured as UnstructuredResource,
      input.filePath,
      input.filename,
      input.mimeType
    );
    
    // Step 2: Process and structure the extracted content
    const fullText = elements.map(el => el.text).join('\n\n');
    const chunks = createContentChunks(elements);
    const metadata = extractDocumentMetadata(elements, fullText);
    
    // Step 3: Detect language and classify document
    const language = detectLanguage(fullText);
    const documentType = classifyDocument(fullText, input.filename);
    
    // Step 4: Extract entities
    const entities = extractEntities(fullText);
    
    // Step 5: Generate summary
    const summary = generateSummary(fullText);
    
    // Step 6: Store results in database
    await storeTextExtractionResults(
      postgres as PostgresResource,
      input.fileId,
      {
        fullText,
        chunks,
        metadata,
        language,
        entities,
        documentType,
        summary
      }
    );
    
    const result: TextExtractionResult = {
      fullText,
      chunks,
      metadata,
      language,
      entities,
      documentType,
      summary
    };
    
    console.log(`Text extraction completed for ${input.filename}: ${chunks.length} chunks, ${fullText.length} characters`);
    return result;
    
  } catch (error) {
    console.error(`Error extracting text from ${input.filename}:`, error);
    throw error;
  }
}

async function extractDocumentElements(
  unstructured: UnstructuredResource,
  filePath: string,
  filename: string,
  mimeType: string
): Promise<DocumentElement[]> {
  
  try {
    // Prepare form data for Unstructured.io API
    const formData = new FormData();
    
    // Note: In a real implementation, you'd read the file from storage
    // For now, we'll simulate this
    const fileBlob = new Blob(['file content'], { type: mimeType });
    formData.append('files', fileBlob, filename);
    
    // Set processing strategy based on file type
    const strategy = mimeType === 'application/pdf' ? 'hi_res' : 'fast';
    formData.append('strategy', strategy);
    formData.append('pdf_infer_table_structure', 'true');
    formData.append('extract_images_in_pdf', 'true');
    formData.append('languages', 'eng');
    
    const response = await fetch(`${unstructured.base_url}/general/v0/general`, {
      method: 'POST',
      body: formData
    });
    
    if (!response.ok) {
      throw new Error(`Unstructured.io API error: ${response.statusText}`);
    }
    
    const elements = await response.json();
    
    return elements.map((el: any) => ({
      type: el.type || 'text',
      text: el.text || '',
      metadata: el.metadata || {}
    }));
    
  } catch (error) {
    console.error('Document extraction error:', error);
    throw error;
  }
}

function createContentChunks(elements: DocumentElement[]): ContentChunk[] {
  const chunks: ContentChunk[] = [];
  let currentChunk = '';
  let chunkIndex = 0;
  let startChar = 0;
  
  const maxChunkSize = 1000;
  const overlapSize = 100;
  
  for (const element of elements) {
    const elementText = element.text.trim();
    if (!elementText) continue;
    
    // If adding this element would exceed chunk size, finalize current chunk
    if (currentChunk.length + elementText.length > maxChunkSize && currentChunk.length > 0) {
      const endChar = startChar + currentChunk.length;
      
      chunks.push({
        index: chunkIndex,
        content: currentChunk.trim(),
        type: 'paragraph',
        pageNumber: element.metadata.page_number,
        startChar,
        endChar
      });
      
      // Start new chunk with overlap
      const overlapText = currentChunk.slice(-overlapSize);
      currentChunk = overlapText + ' ' + elementText;
      startChar = endChar - overlapSize;
      chunkIndex++;
    } else {
      // Add element to current chunk
      currentChunk += (currentChunk ? '\n\n' : '') + elementText;
    }
  }
  
  // Add final chunk
  if (currentChunk.trim()) {
    chunks.push({
      index: chunkIndex,
      content: currentChunk.trim(),
      type: 'paragraph',
      startChar,
      endChar: startChar + currentChunk.length
    });
  }
  
  return chunks;
}

function extractDocumentMetadata(elements: DocumentElement[], fullText: string): DocumentMetadata {
  const pages = new Set<number>();
  let hasImages = false;
  let hasTables = false;
  
  for (const element of elements) {
    if (element.metadata.page_number) {
      pages.add(element.metadata.page_number);
    }
    if (element.type === 'Image') hasImages = true;
    if (element.type === 'Table') hasTables = true;
  }
  
  const words = fullText.split(/\s+/).filter(word => word.length > 0);
  
  return {
    pageCount: pages.size || 1,
    wordCount: words.length,
    characterCount: fullText.length,
    hasImages,
    hasTables,
    language: detectLanguage(fullText)
  };
}

function detectLanguage(text: string): string {
  // Simple language detection based on common words
  // In a real implementation, you might use a proper language detection library
  
  const englishWords = ['the', 'and', 'or', 'but', 'in', 'on', 'at', 'to', 'for', 'of', 'with', 'by'];
  const spanishWords = ['el', 'la', 'y', 'o', 'pero', 'en', 'de', 'con', 'por', 'para', 'que'];
  const frenchWords = ['le', 'la', 'et', 'ou', 'mais', 'dans', 'sur', 'à', 'pour', 'de', 'avec'];
  const germanWords = ['der', 'die', 'das', 'und', 'oder', 'aber', 'in', 'auf', 'zu', 'für', 'von', 'mit'];
  
  const lowerText = text.toLowerCase();
  
  const englishScore = englishWords.reduce((score, word) => 
    score + (lowerText.includes(` ${word} `) ? 1 : 0), 0);
  const spanishScore = spanishWords.reduce((score, word) => 
    score + (lowerText.includes(` ${word} `) ? 1 : 0), 0);
  const frenchScore = frenchWords.reduce((score, word) => 
    score + (lowerText.includes(` ${word} `) ? 1 : 0), 0);
  const germanScore = germanWords.reduce((score, word) => 
    score + (lowerText.includes(` ${word} `) ? 1 : 0), 0);
  
  const maxScore = Math.max(englishScore, spanishScore, frenchScore, germanScore);
  
  if (maxScore === 0) return 'unknown';
  if (maxScore === englishScore) return 'en';
  if (maxScore === spanishScore) return 'es';
  if (maxScore === frenchScore) return 'fr';
  if (maxScore === germanScore) return 'de';
  
  return 'en'; // Default to English
}

function classifyDocument(text: string, filename: string): string {
  const lowerText = text.toLowerCase();
  const lowerFilename = filename.toLowerCase();
  
  // Check filename patterns
  if (lowerFilename.includes('invoice')) return 'invoice';
  if (lowerFilename.includes('contract')) return 'contract';
  if (lowerFilename.includes('report')) return 'report';
  if (lowerFilename.includes('presentation')) return 'presentation';
  if (lowerFilename.includes('resume')) return 'resume';
  if (lowerFilename.includes('manual')) return 'manual';
  
  // Check content patterns
  if (lowerText.includes('invoice') && lowerText.includes('amount due')) return 'invoice';
  if (lowerText.includes('agreement') && lowerText.includes('parties')) return 'contract';
  if (lowerText.includes('quarterly') || lowerText.includes('annual')) return 'report';
  if (lowerText.includes('slide') || lowerText.includes('presentation')) return 'presentation';
  if (lowerText.includes('experience') && lowerText.includes('education')) return 'resume';
  if (lowerText.includes('meeting') && lowerText.includes('agenda')) return 'meeting_notes';
  if (lowerText.includes('procedure') || lowerText.includes('instructions')) return 'manual';
  if (lowerText.includes('dear') && lowerText.includes('sincerely')) return 'letter';
  
  return 'document';
}

function extractEntities(text: string): ExtractedEntity[] {
  const entities: ExtractedEntity[] = [];
  
  // Simple entity extraction patterns
  const patterns = [
    { pattern: /\b\d{4}-\d{2}-\d{2}\b/g, type: 'date' },
    { pattern: /\$[\d,]+\.?\d*/g, type: 'money' },
    { pattern: /\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b/g, type: 'email' },
    { pattern: /\b\d{3}-\d{3}-\d{4}\b/g, type: 'phone' },
    { pattern: /\b[A-Z][a-z]+ [A-Z][a-z]+\b/g, type: 'person' }
  ];
  
  for (const { pattern, type } of patterns) {
    let match;
    while ((match = pattern.exec(text)) !== null) {
      entities.push({
        text: match[0],
        type,
        confidence: 0.8,
        startChar: match.index,
        endChar: match.index + match[0].length
      });
    }
  }
  
  return entities.slice(0, 20); // Limit to 20 entities
}

function generateSummary(text: string): string {
  // Simple extractive summary - take first 200 characters of meaningful content
  const sentences = text.split(/[.!?]+/).filter(s => s.trim().length > 20);
  
  if (sentences.length === 0) return text.slice(0, 200) + '...';
  
  let summary = '';
  for (const sentence of sentences) {
    if (summary.length + sentence.length > 300) break;
    summary += sentence.trim() + '. ';
  }
  
  return summary.trim() || text.slice(0, 200) + '...';
}

async function storeTextExtractionResults(
  postgres: PostgresResource,
  fileId: string,
  results: TextExtractionResult
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
    
    // Update main file record
    const updateFileQuery = `
      UPDATE files 
      SET 
        ocr_text = $2,
        page_count = $3,
        processing_stage = 'extracted',
        processed_at = CURRENT_TIMESTAMP,
        custom_metadata = jsonb_build_object(
          'document_type', $4,
          'language', $5,
          'word_count', $6,
          'character_count', $7,
          'has_images', $8,
          'has_tables', $9,
          'summary', $10,
          'entities', $11
        )
      WHERE id = $1
    `;
    
    await client.query(updateFileQuery, [
      fileId,
      results.fullText,
      results.metadata.pageCount,
      results.documentType,
      results.language,
      results.metadata.wordCount,
      results.metadata.characterCount,
      results.metadata.hasImages,
      results.metadata.hasTables,
      results.summary,
      JSON.stringify(results.entities)
    ]);
    
    // Insert content chunks
    for (const chunk of results.chunks) {
      const insertChunkQuery = `
        INSERT INTO content_chunks (
          file_id, chunk_index, content, chunk_type, 
          start_char, end_char, page_number, metadata
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
      `;
      
      await client.query(insertChunkQuery, [
        fileId,
        chunk.index,
        chunk.content,
        chunk.type,
        chunk.startChar,
        chunk.endChar,
        chunk.pageNumber || null,
        JSON.stringify({ language: results.language })
      ]);
    }
    
    await client.query('COMMIT');
    console.log(`Stored text extraction results for file ${fileId}: ${results.chunks.length} chunks`);
    
  } catch (error) {
    await client.query('ROLLBACK');
    throw error;
  } finally {
    await client.end();
  }
}