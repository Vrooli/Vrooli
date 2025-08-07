// Upload Handler - Manages file uploads and transcription pipeline integration
// This script interfaces with the n8n transcription pipeline workflow

interface UploadRequest {
  file: File;
  whisperModel?: string;
  autoAnalyze?: boolean;
  sessionId?: string;
  userIdentifier?: string;
}

interface TranscriptionResponse {
  success: boolean;
  transcriptionId: string;
  filename: string;
  transcriptionText: string;
  confidence: number;
  language: string;
  duration: number;
  processingTime: number;
  urls: {
    view: string;
    download: string;
    audio: string;
  };
}

/**
 * Validates uploaded file before processing
 */
export async function validateFile(file: File): Promise<{ valid: boolean; error?: string }> {
  // Check file size (500MB limit)
  const maxSizeBytes = 500 * 1024 * 1024;
  if (file.size > maxSizeBytes) {
    return { valid: false, error: `File size ${(file.size / 1024 / 1024).toFixed(1)}MB exceeds limit of 500MB` };
  }
  
  // Check file type
  const supportedTypes = [
    'audio/mpeg', 'audio/mp3',
    'audio/wav', 'audio/wave',
    'audio/m4a', 'audio/mp4',
    'audio/ogg', 'audio/vorbis',
    'audio/flac'
  ];
  
  if (!supportedTypes.includes(file.type)) {
    return { valid: false, error: `Unsupported file type: ${file.type}. Supported formats: MP3, WAV, M4A, OGG, FLAC` };
  }
  
  // Check filename
  if (file.name.length > 255) {
    return { valid: false, error: 'Filename too long (max 255 characters)' };
  }
  
  return { valid: true };
}

/**
 * Initiates transcription by calling the n8n workflow
 */
export async function startTranscription(uploadRequest: UploadRequest): Promise<TranscriptionResponse> {
  try {
    // Validate file first
    const validation = await validateFile(uploadRequest.file);
    if (!validation.valid) {
      throw new Error(validation.error);
    }
    
    // Convert file to base64 for n8n processing
    const fileData = await fileToBase64(uploadRequest.file);
    
    // Prepare request payload for n8n transcription pipeline
    const payload = {
      filename: uploadRequest.file.name,
      fileData: fileData,
      contentType: uploadRequest.file.type,
      fileSizeBytes: uploadRequest.file.size,
      whisperModel: uploadRequest.whisperModel || 'base',
      sessionId: uploadRequest.sessionId || generateSessionId(),
      userIdentifier: uploadRequest.userIdentifier || 'windmill-user',
      originalPath: `/uploads/${uploadRequest.file.name}`
    };
    
    // Call n8n transcription pipeline webhook
    const response = await fetch('http://localhost:5678/webhook/transcription-upload', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(payload)
    });
    
    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`Transcription pipeline failed: ${response.status} - ${errorText}`);
    }
    
    const result = await response.json() as TranscriptionResponse;
    
    // If auto-analyze is enabled, trigger analysis
    if (uploadRequest.autoAnalyze && result.success) {
      // Trigger summary analysis asynchronously
      setTimeout(async () => {
        try {
          await triggerAutoAnalysis(result.transcriptionId, uploadRequest.sessionId);
        } catch (error) {
          console.warn('Auto-analysis failed:', error);
        }
      }, 1000);
    }
    
    return result;
    
  } catch (error) {
    console.error('Upload error:', error);
    throw error instanceof Error ? error : new Error('Unknown upload error');
  }
}

/**
 * Converts file to base64 string for n8n processing
 */
async function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      if (typeof reader.result === 'string') {
        // Remove data URL prefix to get just base64
        const base64 = reader.result.split(',')[1];
        resolve(base64);
      } else {
        reject(new Error('Failed to convert file to base64'));
      }
    };
    reader.onerror = () => reject(new Error('Failed to read file'));
    reader.readAsDataURL(file);
  });
}

/**
 * Generates a session ID for tracking
 */
function generateSessionId(): string {
  return `windmill-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

/**
 * Triggers automatic analysis after transcription
 */
async function triggerAutoAnalysis(transcriptionId: string, sessionId?: string): Promise<void> {
  const payload = {
    transcriptionId: transcriptionId,
    analysisType: 'summary',
    sessionId: sessionId || generateSessionId(),
    userIdentifier: 'windmill-auto'
  };
  
  await fetch('http://localhost:5678/webhook/ai-analysis', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(payload)
  });
}

/**
 * Fetches list of all transcriptions for the grid display
 */
export async function fetchTranscriptionsList(): Promise<any[]> {
  try {
    // In a real implementation, this would query the database
    // For now, we'll make a simple HTTP request to a hypothetical endpoint
    const response = await fetch('http://localhost:3000/api/transcriptions', {
      method: 'GET',
      headers: {
        'Accept': 'application/json'
      }
    });
    
    if (!response.ok) {
      throw new Error(`Failed to fetch transcriptions: ${response.status}`);
    }
    
    const transcriptions = await response.json();
    
    // Format data for the UI grid
    return transcriptions.map((t: any) => ({
      id: t.id,
      filename: t.filename,
      duration_formatted: formatDuration(t.duration_seconds),
      confidence_score: Math.round((t.confidence_score || 0) * 100),
      language_detected: t.language_detected || 'Unknown',
      created_at_formatted: formatDate(t.created_at),
      transcription_text: t.transcription_text,
      // Add action buttons data
      actions: {
        view: true,
        download: true,
        delete: true,
        analyze: true
      }
    }));
    
  } catch (error) {
    console.error('Failed to fetch transcriptions:', error);
    return []; // Return empty array on error to prevent UI crashes
  }
}

/**
 * Deletes a transcription and associated files
 */
export async function deleteTranscription(transcriptionId: string): Promise<{ success: boolean; message: string }> {
  try {
    const response = await fetch(`http://localhost:3000/api/transcriptions/${transcriptionId}`, {
      method: 'DELETE',
      headers: {
        'Accept': 'application/json'
      }
    });
    
    if (!response.ok) {
      throw new Error(`Delete failed: ${response.status}`);
    }
    
    const result = await response.json();
    return { success: true, message: 'Transcription deleted successfully' };
    
  } catch (error) {
    console.error('Delete error:', error);
    return { success: false, message: error instanceof Error ? error.message : 'Delete failed' };
  }
}

/**
 * Utility function to format duration in seconds to MM:SS
 */
function formatDuration(seconds: number): string {
  if (!seconds || seconds < 0) return 'Unknown';
  
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
}

/**
 * Utility function to format date for display
 */
function formatDate(dateString: string): string {
  try {
    const date = new Date(dateString);
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  } catch {
    return 'Unknown';
  }
}

/**
 * Export transcription in various formats
 */
export async function exportTranscription(transcriptionId: string, format: 'txt' | 'pdf' | 'json' | 'srt'): Promise<Blob> {
  const response = await fetch(`http://localhost:3000/api/transcriptions/${transcriptionId}/export?format=${format}`, {
    method: 'GET'
  });
  
  if (!response.ok) {
    throw new Error(`Export failed: ${response.status}`);
  }
  
  return await response.blob();
}