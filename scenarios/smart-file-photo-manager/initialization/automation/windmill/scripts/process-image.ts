/**
 * Smart File Photo Manager - Image Processing Script
 * Analyzes images using LLaVA vision model for content understanding
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

interface ImageAnalysisResult {
  description: string;
  detectedObjects: string[];
  sceneType: string;
  hasText: boolean;
  extractedText?: string;
  confidence: number;
  suggestedTags: string[];
  embedding: number[];
}

interface ImageProcessingInput {
  fileId: string;
  imagePath: string;
  filename: string;
  mimeType: string;
  width?: number;
  height?: number;
}

export async function main(
  input: ImageProcessingInput,
  ollama: Resource<"ollama"> = "f/file_manager/ollama",
  qdrant: Resource<"qdrant"> = "f/file_manager/qdrant", 
  postgres: Resource<"postgres"> = "f/file_manager/postgres"
): Promise<ImageAnalysisResult> {
  
  console.log(`Processing image: ${input.filename} (${input.fileId})`);
  
  try {
    // Step 1: Analyze image with LLaVA
    const visionAnalysis = await analyzeImageWithVision(
      ollama as OllamaResource,
      input.imagePath,
      input.filename
    );
    
    // Step 2: Generate embedding for visual content
    const embedding = await generateImageEmbedding(
      ollama as OllamaResource,
      visionAnalysis.description,
      visionAnalysis.detectedObjects
    );
    
    // Step 3: Store analysis results in database
    await storeImageAnalysis(
      postgres as PostgresResource,
      input.fileId,
      visionAnalysis,
      embedding
    );
    
    // Step 4: Store embedding in vector database
    await storeImageEmbedding(
      qdrant as QdrantResource,
      input.fileId,
      embedding,
      {
        filename: input.filename,
        detectedObjects: visionAnalysis.detectedObjects,
        sceneType: visionAnalysis.sceneType,
        width: input.width,
        height: input.height,
        hasText: visionAnalysis.hasText
      }
    );
    
    const result: ImageAnalysisResult = {
      description: visionAnalysis.description,
      detectedObjects: visionAnalysis.detectedObjects,
      sceneType: visionAnalysis.sceneType,
      hasText: visionAnalysis.hasText,
      extractedText: visionAnalysis.extractedText,
      confidence: visionAnalysis.confidence,
      suggestedTags: generateTagsFromAnalysis(visionAnalysis),
      embedding: embedding
    };
    
    console.log(`Image processing completed for ${input.filename}`);
    return result;
    
  } catch (error) {
    console.error(`Error processing image ${input.filename}:`, error);
    throw error;
  }
}

async function analyzeImageWithVision(
  ollama: OllamaResource,
  imagePath: string,
  filename: string
): Promise<{
  description: string;
  detectedObjects: string[];
  sceneType: string;
  hasText: boolean;
  extractedText?: string;
  confidence: number;
}> {
  
  const prompts = {
    description: `Analyze this image and provide a detailed description of what you see. Include:
- Main subjects and objects
- Setting and environment
- Actions taking place
- Visual characteristics and composition
Be specific and descriptive.`,
    
    objects: `List all distinct objects, items, and elements you can identify in this image. 
Format as a comma-separated list. Focus on concrete, identifiable objects.`,
    
    scene: `What type of scene or setting is this? Choose from categories like:
indoor, outdoor, office, home, nature, city, beach, mountain, restaurant, vehicle, etc.
Provide just the most appropriate category.`,
    
    text: `Is there any readable text visible in this image? If yes, transcribe it exactly. 
If no text is visible, respond with "NO_TEXT".`
  };
  
  try {
    // Analyze description
    const descriptionResponse = await fetch(`${ollama.base_url}/api/generate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        model: 'llava:13b',
        prompt: prompts.description,
        images: [await encodeImageToBase64(imagePath)],
        stream: false,
        options: {
          temperature: 0.1,
          top_p: 0.9
        }
      })
    });
    
    if (!descriptionResponse.ok) {
      throw new Error(`Vision API error: ${descriptionResponse.statusText}`);
    }
    
    const descriptionResult = await descriptionResponse.json();
    const description = descriptionResult.response.trim();
    
    // Analyze objects
    const objectsResponse = await fetch(`${ollama.base_url}/api/generate`, {
      method: 'POST', 
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        model: 'llava:13b',
        prompt: prompts.objects,
        images: [await encodeImageToBase64(imagePath)],
        stream: false,
        options: {
          temperature: 0.1,
          top_p: 0.9
        }
      })
    });
    
    const objectsResult = await objectsResponse.json();
    const detectedObjects = parseObjectsList(objectsResult.response);
    
    // Analyze scene
    const sceneResponse = await fetch(`${ollama.base_url}/api/generate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        model: 'llava:13b',
        prompt: prompts.scene,
        images: [await encodeImageToBase64(imagePath)],
        stream: false,
        options: {
          temperature: 0.1,
          top_p: 0.9
        }
      })
    });
    
    const sceneResult = await sceneResponse.json();
    const sceneType = sceneResult.response.trim().toLowerCase();
    
    // Analyze text
    const textResponse = await fetch(`${ollama.base_url}/api/generate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        model: 'llava:13b',
        prompt: prompts.text,
        images: [await encodeImageToBase64(imagePath)],
        stream: false,
        options: {
          temperature: 0.1,
          top_p: 0.9
        }
      })
    });
    
    const textResult = await textResponse.json();
    const extractedText = textResult.response.trim();
    const hasText = extractedText !== "NO_TEXT" && extractedText.length > 0;
    
    // Calculate confidence based on response length and detail
    const confidence = calculateVisionConfidence(description, detectedObjects);
    
    return {
      description,
      detectedObjects,
      sceneType,
      hasText,
      extractedText: hasText ? extractedText : undefined,
      confidence
    };
    
  } catch (error) {
    console.error('Vision analysis error:', error);
    throw error;
  }
}

async function generateImageEmbedding(
  ollama: OllamaResource,
  description: string,
  objects: string[]
): Promise<number[]> {
  
  // Combine description and objects for comprehensive embedding
  const content = `${description} Objects: ${objects.join(', ')}`;
  
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
    console.error('Embedding generation error:', error);
    throw error;
  }
}

async function storeImageAnalysis(
  postgres: PostgresResource,
  fileId: string,
  analysis: any,
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
      SET 
        description = $2,
        detected_objects = $3,
        visual_embedding = $4,
        processing_stage = 'analyzed',
        processed_at = CURRENT_TIMESTAMP,
        custom_metadata = jsonb_build_object(
          'scene_type', $5,
          'has_text', $6,
          'extracted_text', $7,
          'vision_confidence', $8
        )
      WHERE id = $1
    `;
    
    await client.query(query, [
      fileId,
      analysis.description,
      JSON.stringify(analysis.detectedObjects),
      `[${embedding.join(',')}]`,
      analysis.sceneType,
      analysis.hasText,
      analysis.extractedText || null,
      analysis.confidence
    ]);
    
    console.log(`Stored image analysis for file ${fileId}`);
    
  } finally {
    await client.end();
  }
}

async function storeImageEmbedding(
  qdrant: QdrantResource,
  fileId: string,
  embedding: number[],
  metadata: any
): Promise<void> {
  
  try {
    const response = await fetch(`${qdrant.url}/collections/image_embeddings/points`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        points: [{
          id: fileId,
          vector: embedding,
          payload: {
            file_id: fileId,
            ...metadata,
            indexed_at: new Date().toISOString()
          }
        }]
      })
    });
    
    if (!response.ok) {
      throw new Error(`Qdrant error: ${response.statusText}`);
    }
    
    console.log(`Stored image embedding for file ${fileId}`);
    
  } catch (error) {
    console.error('Error storing image embedding:', error);
    throw error;
  }
}

// Helper functions
async function encodeImageToBase64(imagePath: string): Promise<string> {
  // This would need to be implemented based on the actual file system access
  // For now, return a placeholder
  return "base64_encoded_image_data";
}

function parseObjectsList(response: string): string[] {
  return response
    .split(',')
    .map(obj => obj.trim().toLowerCase())
    .filter(obj => obj.length > 0 && obj !== 'none' && obj !== 'n/a')
    .slice(0, 10); // Limit to 10 objects
}

function calculateVisionConfidence(description: string, objects: string[]): number {
  let confidence = 0.5; // Base confidence
  
  // Increase confidence based on description length and detail
  if (description.length > 100) confidence += 0.2;
  if (description.length > 200) confidence += 0.1;
  
  // Increase confidence based on number of detected objects
  if (objects.length > 2) confidence += 0.1;
  if (objects.length > 5) confidence += 0.1;
  
  // Cap confidence at 0.95
  return Math.min(confidence, 0.95);
}

function generateTagsFromAnalysis(analysis: any): string[] {
  const tags = new Set<string>();
  
  // Add scene type as tag
  tags.add(analysis.sceneType);
  
  // Add top detected objects as tags
  analysis.detectedObjects.slice(0, 5).forEach((obj: string) => tags.add(obj));
  
  // Add contextual tags based on description
  const description = analysis.description.toLowerCase();
  
  if (description.includes('person') || description.includes('people')) tags.add('people');
  if (description.includes('work') || description.includes('office')) tags.add('work');
  if (description.includes('family') || description.includes('home')) tags.add('personal');
  if (description.includes('vacation') || description.includes('travel')) tags.add('travel');
  if (description.includes('meeting') || description.includes('presentation')) tags.add('business');
  if (description.includes('nature') || description.includes('landscape')) tags.add('nature');
  if (description.includes('food') || description.includes('restaurant')) tags.add('food');
  if (description.includes('celebration') || description.includes('party')) tags.add('celebration');
  
  return Array.from(tags).slice(0, 8); // Limit to 8 tags
}