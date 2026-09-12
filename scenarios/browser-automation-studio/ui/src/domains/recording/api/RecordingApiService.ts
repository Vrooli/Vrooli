/**
 * RecordingApiService
 *
 * Centralized API service for recording domain operations.
 * Features:
 * - Proper JSON parsing with error logging (no silent failures)
 * - Zod schema validation
 * - Request deduplication for expensive operations
 * - AbortController support for cancellation
 * - Consistent error handling
 */

import { z } from 'zod';
import { ConnectError } from '@connectrpc/connect';
import { safeParse } from '@/shared/api/safeParse';
import { getApiBase } from '@/config';
import { logger } from '@/utils/logger';
import { visionNavigationClient } from '@/api/visionNavigation';
import type { StartNavigationResponse } from '@/api/visionNavigation';
import * as schemas from './schemas';

// ============================================================================
// Types
// ============================================================================

export interface RequestOptions {
  signal?: AbortSignal;
}

/**
 * Discriminated union result type for API calls.
 * Hooks consume this and handle success/failure in their state management.
 */
export type ApiResult<T> =
  | { success: true; data: T }
  | { success: false; error: string };

// ============================================================================
// RecordingApiService
// ============================================================================

export class RecordingApiService {
  private apiUrl: string;

  /** Request deduplication map for expensive operations */
  private pendingRequests = new Map<string, Promise<ApiResult<unknown>>>();

  constructor(apiUrl?: string) {
    this.apiUrl = apiUrl ?? getApiBase();
  }

  // ==========================================================================
  // Private Helpers
  // ==========================================================================

  /**
   * Parse JSON with proper error logging (not silent like the old safeJson).
   */
  private async parseJson(response: Response, context: string): Promise<unknown> {
    const text = await response.text();
    if (!text) return null;

    try {
      return JSON.parse(text);
    } catch (err) {
      logger.error(`JSON parse failed for ${context}`, {
        component: 'RecordingApiService',
        action: 'parseJson',
        responsePreview: text.slice(0, 200),
      }, err);
      throw new Error(`Invalid JSON response from ${context}`);
    }
  }

  /**
   * Validate data with Zod schema, returning ApiResult.
   */
  private validate<T>(
    schema: z.ZodSchema<T>,
    data: unknown,
    context: string
  ): ApiResult<T> {
    const result = safeParse(schema, data, context);
    if (result.success) {
      return { success: true, data: result.data };
    }
    return { success: false, error: result.error };
  }

  /**
   * Deduplicate concurrent requests to the same endpoint.
   * Useful for expensive operations like replay preview or workflow generation.
   */
  private async deduplicatedRequest<T>(
    key: string,
    request: () => Promise<ApiResult<T>>
  ): Promise<ApiResult<T>> {
    const pending = this.pendingRequests.get(key);
    if (pending) {
      logger.debug(`Request deduplicated: ${key}`, { component: 'RecordingApiService' });
      return pending as Promise<ApiResult<T>>;
    }

    const promise = request().finally(() => {
      this.pendingRequests.delete(key);
    });

    this.pendingRequests.set(key, promise as Promise<ApiResult<unknown>>);
    return promise;
  }

  /**
   * Extract error message from API error response.
   */
  private extractErrorMessage(payload: unknown, fallback: string): string {
    const result = schemas.ApiErrorResponseSchema.safeParse(payload);
    if (result.success) {
      return result.data.message ?? result.data.error ?? `API error: ${fallback}`;
    }
    return `API error: ${fallback}`;
  }

  /**
   * Handle common error cases (abort, network errors).
   */
  private handleError(err: unknown, context: string): ApiResult<never> {
    if (err instanceof Error) {
      if (err.name === 'AbortError') {
        return { success: false, error: 'Request cancelled' };
      }
      logger.error(`API request failed: ${context}`, {
        component: 'RecordingApiService',
        action: context,
      }, err);
      return { success: false, error: err.message };
    }
    return { success: false, error: 'Unknown error' };
  }

  // ==========================================================================
  // Recording Lifecycle APIs
  // ==========================================================================

  /**
   * Start recording on a session.
   * Handles 409 Conflict when recording is already in progress.
   */
  async startRecording(
    sessionId: string,
    options?: RequestOptions
  ): Promise<ApiResult<schemas.StartRecordingResponse>> {
    try {
      const response = await fetch(`${this.apiUrl}/recordings/live/start`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: sessionId }),
        signal: options?.signal,
      });

      const payload = await this.parseJson(response, 'startRecording');

      // Handle 409 Conflict: Recording already in progress
      // This happens on page refresh - treat as successful state sync
      if (response.status === 409) {
        // Backend returns error format {code, message} for 409, not StartRecordingResponse
        // Check if it's the expected RECORDING_IN_PROGRESS error
        const errorResult = schemas.ApiErrorResponseSchema.safeParse(payload);
        if (errorResult.success && errorResult.data.code === 'RECORDING_IN_PROGRESS') {
          // Recording is active - return synthetic success for session restoration
          const data: schemas.StartRecordingResponse = {
            recording_id: `restored-${sessionId}-${Date.now()}`,
            session_id: sessionId,
            started_at: new Date().toISOString(),
          };
          logger.info('Recording already in progress, syncing state', {
            component: 'RecordingApiService',
            recordingId: data.recording_id,
          });
          return { success: true, data };
        }

        // Fallback: try parsing as relaxed StartRecordingResponse
        const relaxedSchema = schemas.StartRecordingResponseSchema.extend({
          recording_id: z.string().optional(),
          session_id: z.string().optional(),
          started_at: z.string().optional(),
        });
        const validated = this.validate(relaxedSchema, payload, 'StartRecording (409)');
        if (validated.success) {
          const data: schemas.StartRecordingResponse = {
            recording_id: validated.data.recording_id ?? `restored-${sessionId}-${Date.now()}`,
            session_id: validated.data.session_id ?? sessionId,
            started_at: validated.data.started_at ?? new Date().toISOString(),
          };
          return { success: true, data };
        }

        // If all parsing fails, still treat as success since 409 means recording is active
        logger.warn('409 response format unexpected, treating as active recording', {
          component: 'RecordingApiService',
          payload,
        });
        return {
          success: true,
          data: {
            recording_id: `restored-${sessionId}-${Date.now()}`,
            session_id: sessionId,
            started_at: new Date().toISOString(),
          },
        };
      }

      if (!response.ok) {
        const message = this.extractErrorMessage(payload, response.statusText);
        return { success: false, error: message };
      }

      return this.validate(schemas.StartRecordingResponseSchema, payload, 'StartRecording');
    } catch (err) {
      return this.handleError(err, 'startRecording');
    }
  }

  /**
   * Stop recording on a session.
   */
  async stopRecording(
    sessionId: string,
    options?: RequestOptions
  ): Promise<ApiResult<schemas.StopRecordingResponse>> {
    try {
      const response = await fetch(`${this.apiUrl}/recordings/live/${sessionId}/stop`, {
        method: 'POST',
        signal: options?.signal,
      });

      const payload = await this.parseJson(response, 'stopRecording');

      if (!response.ok) {
        const message = this.extractErrorMessage(payload, response.statusText);
        return { success: false, error: message };
      }

      return this.validate(schemas.StopRecordingResponseSchema, payload, 'StopRecording');
    } catch (err) {
      return this.handleError(err, 'stopRecording');
    }
  }

  /**
   * Generate workflow from recorded actions.
   * Uses request deduplication to prevent concurrent calls.
   */
  async generateWorkflow(
    sessionId: string,
    params: {
      name: string;
      projectId?: string;
      actions: unknown[];
      settings?: unknown;
    },
    options?: RequestOptions
  ): Promise<ApiResult<schemas.GenerateWorkflowResponse>> {
    const key = `generateWorkflow:${sessionId}`;

    return this.deduplicatedRequest(key, async () => {
      try {
        const response = await fetch(
          `${this.apiUrl}/recordings/live/${sessionId}/generate-workflow`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              name: params.name,
              project_id: params.projectId,
              actions: params.actions,
              settings: params.settings,
            }),
            signal: options?.signal,
          }
        );

        const payload = await this.parseJson(response, 'generateWorkflow');

        if (!response.ok) {
          const message = this.extractErrorMessage(payload, response.statusText);
          return { success: false, error: message };
        }

        return this.validate(
          schemas.GenerateWorkflowResponseSchema,
          payload,
          'GenerateWorkflow'
        );
      } catch (err) {
        return this.handleError(err, 'generateWorkflow');
      }
    });
  }

  /**
   * Validate a selector on the current page.
   */
  async validateSelector(
    sessionId: string,
    selector: string,
    options?: RequestOptions
  ): Promise<ApiResult<schemas.SelectorValidation>> {
    try {
      const response = await fetch(
        `${this.apiUrl}/recordings/live/${sessionId}/validate-selector`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ selector }),
          signal: options?.signal,
        }
      );

      const payload = await this.parseJson(response, 'validateSelector');

      if (!response.ok) {
        const message = this.extractErrorMessage(payload, response.statusText);
        return { success: false, error: message };
      }

      return this.validate(schemas.SelectorValidationSchema, payload, 'SelectorValidation');
    } catch (err) {
      return this.handleError(err, 'validateSelector');
    }
  }

  /**
   * Replay actions for preview.
   * Uses request deduplication to prevent concurrent calls.
   */
  async replayPreview(
    sessionId: string,
    params: {
      actions: unknown[];
      limit?: number;
      stopOnFailure?: boolean;
    },
    options?: RequestOptions
  ): Promise<ApiResult<schemas.ReplayPreviewResponse>> {
    const key = `replayPreview:${sessionId}`;

    return this.deduplicatedRequest(key, async () => {
      try {
        const response = await fetch(
          `${this.apiUrl}/recordings/live/${sessionId}/replay-preview`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              actions: params.actions,
              limit: params.limit,
              stop_on_failure: params.stopOnFailure ?? true,
            }),
            signal: options?.signal,
          }
        );

        const payload = await this.parseJson(response, 'replayPreview');

        if (!response.ok) {
          const message = this.extractErrorMessage(payload, response.statusText);
          return { success: false, error: message };
        }

        return this.validate(schemas.ReplayPreviewResponseSchema, payload, 'ReplayPreview');
      } catch (err) {
        return this.handleError(err, 'replayPreview');
      }
    });
  }

  // ==========================================================================
  // Session APIs
  // ==========================================================================

  /**
   * Create a new recording session.
   */
  async createSession(
    params: {
      viewportWidth?: number;
      viewportHeight?: number;
      sessionProfileId?: string;
      streamQuality?: number;
      streamFps?: number;
      streamScale?: 'css' | 'device';
      restoreTabs?: boolean;
    },
    options?: RequestOptions
  ): Promise<ApiResult<schemas.CreateSessionResponse>> {
    try {
      const response = await fetch(`${this.apiUrl}/recordings/live/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          viewport_width: params.viewportWidth ?? 1280,
          viewport_height: params.viewportHeight ?? 720,
          session_profile_id: params.sessionProfileId,
          stream_quality: params.streamQuality,
          stream_fps: params.streamFps,
          stream_scale: params.streamScale,
          restore_tabs: params.restoreTabs,
        }),
        signal: options?.signal,
      });

      const payload = await this.parseJson(response, 'createSession');

      if (!response.ok) {
        const message = this.extractErrorMessage(payload, response.statusText);
        return { success: false, error: message };
      }

      return this.validate(schemas.CreateSessionResponseSchema, payload, 'CreateSession');
    } catch (err) {
      return this.handleError(err, 'createSession');
    }
  }

  // ==========================================================================
  // Timeline APIs
  // ==========================================================================

  /**
   * Fetch timeline entries for a session.
   */
  async getTimeline(
    sessionId: string,
    params?: {
      limit?: number;
      pageId?: string;
    },
    options?: RequestOptions
  ): Promise<ApiResult<schemas.TimelineResponse>> {
    try {
      const searchParams = new URLSearchParams();
      if (params?.limit) {
        searchParams.set('limit', params.limit.toString());
      }
      if (params?.pageId) {
        searchParams.set('pageId', params.pageId);
      }

      const url = `${this.apiUrl}/recordings/live/${sessionId}/timeline?${searchParams}`;
      const response = await fetch(url, { signal: options?.signal });

      const payload = await this.parseJson(response, 'getTimeline');

      if (!response.ok) {
        const message = this.extractErrorMessage(payload, response.statusText);
        return { success: false, error: message };
      }

      return this.validate(schemas.TimelineResponseSchema, payload, 'Timeline');
    } catch (err) {
      return this.handleError(err, 'getTimeline');
    }
  }

  // ==========================================================================
  // AI Navigation APIs
  // ==========================================================================

  /**
   * Start AI-driven navigation via the VisionNavigationService Connect-RPC.
   *
   * Returns the proto StartNavigationResponse directly on success. On error
   * the connect error is serialized as enriched JSON (code/message) so the
   * useAINavigation hook can reconstruct an AINavigationError.
   */
  async startAINavigation(
    params: {
      sessionId: string;
      prompt: string;
      model: string;
      maxSteps?: number;
    },
    headers: HeadersInit,
    options?: RequestOptions
  ): Promise<ApiResult<StartNavigationResponse>> {
    try {
      const resp = await visionNavigationClient.startNavigation(
        {
          sessionId: params.sessionId,
          prompt: params.prompt,
          model: params.model,
          maxSteps: params.maxSteps ?? 20,
          clientSource: this.extractClientSource(headers),
        },
        { signal: options?.signal },
      );
      return { success: true, data: resp };
    } catch (err) {
      return this.handleConnectError(err, 'startAINavigation', true);
    }
  }

  /**
   * Abort AI navigation via VisionNavigationService.
   */
  async abortAINavigation(
    navigationId: string,
    options?: RequestOptions
  ): Promise<ApiResult<void>> {
    try {
      await visionNavigationClient.abortNavigation(
        { navigationId },
        { signal: options?.signal },
      );
      return { success: true, data: undefined };
    } catch (err) {
      return this.handleConnectError(err, 'abortAINavigation', false);
    }
  }

  /**
   * Resume AI navigation after human intervention.
   */
  async resumeAINavigation(
    navigationId: string,
    options?: RequestOptions
  ): Promise<ApiResult<void>> {
    try {
      await visionNavigationClient.resumeNavigation(
        { navigationId },
        { signal: options?.signal },
      );
      return { success: true, data: undefined };
    } catch (err) {
      return this.handleConnectError(err, 'resumeAINavigation', false);
    }
  }

  /**
   * Pull X-Client-Source out of a HeadersInit blob so the StartNavigation
   * proto field can carry it. We accept both Headers and plain objects since
   * callers vary.
   */
  private extractClientSource(headers: HeadersInit | undefined): string {
    if (!headers) return '';
    if (headers instanceof Headers) {
      return headers.get('X-Client-Source') ?? '';
    }
    if (Array.isArray(headers)) {
      const entry = headers.find(([k]) => k.toLowerCase() === 'x-client-source');
      return entry ? entry[1] : '';
    }
    // Record<string, string>
    const rec = headers as Record<string, string>;
    return rec['X-Client-Source'] ?? rec['x-client-source'] ?? '';
  }

  /**
   * Convert a Connect error into our ApiResult shape. When `enriched` is
   * true the error string is a JSON envelope { code, message, details } so
   * useAINavigation can reconstruct an AINavigationError.
   */
  private handleConnectError<T>(
    err: unknown,
    context: string,
    enriched: boolean,
  ): ApiResult<T> {
    if (err instanceof ConnectError) {
      logger.warn(`Connect error in ${context}`, {
        component: 'RecordingApiService',
        code: err.code,
        message: err.rawMessage,
      });
      if (enriched) {
        return {
          success: false,
          error: JSON.stringify({
            code: String(err.code),
            message: err.rawMessage,
          }),
        };
      }
      return { success: false, error: err.rawMessage };
    }
    return this.handleError(err, context);
  }

  // History APIs (updateHistorySettings, navigateToHistoryUrl) moved to
  // RecordingsService over Connect-RPC during the Phase 8 migration. See
  // ui/src/api/recordings.ts and ui/src/domains/recording/hooks/useHistory.ts.
}

// ============================================================================
// Singleton Instance
// ============================================================================

/**
 * Default singleton instance of RecordingApiService.
 * Use this for most cases. Create new instances only for testing or custom configs.
 */
export const recordingApi = new RecordingApiService();
