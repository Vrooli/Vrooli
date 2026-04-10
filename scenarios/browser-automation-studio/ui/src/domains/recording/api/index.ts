/**
 * Recording API Module
 *
 * Exports the centralized API service and schemas for the recording domain.
 */

// Service
export { RecordingApiService, recordingApi } from './RecordingApiService';
export type { RequestOptions, ApiResult } from './RecordingApiService';

// Schemas and types
export * from './schemas';
