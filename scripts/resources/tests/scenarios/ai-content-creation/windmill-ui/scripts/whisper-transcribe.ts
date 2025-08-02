/**
 * Whisper Audio Transcription Script
 * Processes audio files through Whisper API and returns transcription
 */

interface TranscriptionInput {
    audio_file?: string;           // Base64 encoded audio file
    audio_url?: string;            // URL to audio file
    response_format?: 'json' | 'text' | 'srt' | 'verbose_json';
    language?: string;             // Optional language hint
    temperature?: number;          // Sampling temperature (0-1)
}

interface TranscriptionOutput {
    success: boolean;
    text?: string;
    error?: string;
    processing_time?: number;
    language?: string;
    confidence?: number;
}

export async function main(input: TranscriptionInput): Promise<TranscriptionOutput> {
    const startTime = Date.now();
    
    try {
        // Validate input
        if (!input.audio_file && !input.audio_url) {
            return {
                success: false,
                error: "Either audio_file or audio_url must be provided"
            };
        }

        // Get Whisper service URL from environment or default
        const whisperUrl = process.env.WHISPER_BASE_URL || 'http://localhost:8090';
        
        // Prepare form data for multipart/form-data request
        const formData = new FormData();
        
        if (input.audio_file) {
            // Handle base64 encoded audio file
            const audioBuffer = Buffer.from(input.audio_file, 'base64');
            const audioBlob = new Blob([audioBuffer], { type: 'audio/wav' });
            formData.append('audio', audioBlob, 'audio.wav');
        } else if (input.audio_url) {
            // Fetch audio from URL
            const audioResponse = await fetch(input.audio_url);
            if (!audioResponse.ok) {
                return {
                    success: false,
                    error: `Failed to fetch audio from URL: ${audioResponse.statusText}`
                };
            }
            const audioBlob = await audioResponse.blob();
            formData.append('audio', audioBlob, 'audio.wav');
        }

        // Add optional parameters
        formData.append('response_format', input.response_format || 'json');
        if (input.language) {
            formData.append('language', input.language);
        }
        if (input.temperature !== undefined) {
            formData.append('temperature', input.temperature.toString());
        }

        // Make request to Whisper API
        const response = await fetch(`${whisperUrl}/transcribe`, {
            method: 'POST',
            body: formData,
            headers: {
                // Don't set Content-Type - let browser set it with boundary for multipart
            }
        });

        if (!response.ok) {
            return {
                success: false,
                error: `Whisper API error: ${response.status} ${response.statusText}`,
                processing_time: Date.now() - startTime
            };
        }

        const result = await response.json();
        
        return {
            success: true,
            text: result.text || result.transcript || '',
            language: result.language || input.language,
            confidence: result.confidence,
            processing_time: Date.now() - startTime
        };

    } catch (error) {
        return {
            success: false,
            error: `Transcription failed: ${error instanceof Error ? error.message : String(error)}`,
            processing_time: Date.now() - startTime
        };
    }
}

// Example usage for testing
export const test_input: TranscriptionInput = {
    audio_file: "", // Base64 encoded audio would go here
    response_format: 'json',
    language: 'en',
    temperature: 0.0
};