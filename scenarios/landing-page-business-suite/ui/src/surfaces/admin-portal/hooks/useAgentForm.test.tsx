import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useAgentForm } from './useAgentForm';
import * as agentService from '../services/agent.service';

// Mock the service module
vi.mock('../services/agent.service', async () => {
  const actual = await vi.importActual('../services/agent.service');
  return {
    ...actual,
    submitAgentCustomization: vi.fn(),
  };
});

const mockSubmitAgentCustomization = vi.mocked(agentService.submitAgentCustomization);

describe('useAgentForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('initial state', () => {
    it('has default form values', () => {
      const { result } = renderHook(() => useAgentForm());
      expect(result.current.form).toEqual({
        brief: '',
        assets: '',
        preview: true,
      });
    });

    it('has no result initially', () => {
      const { result } = renderHook(() => useAgentForm());
      expect(result.current.result).toBeNull();
    });

    it('is not submitting initially', () => {
      const { result } = renderHook(() => useAgentForm());
      expect(result.current.submitting).toBe(false);
    });

    it('has no error initially', () => {
      const { result } = renderHook(() => useAgentForm());
      expect(result.current.error).toBeNull();
    });

    it('has no validation error initially', () => {
      const { result } = renderHook(() => useAgentForm());
      expect(result.current.validationError).toBeNull();
    });
  });

  describe('form updates', () => {
    it('updates brief', () => {
      const { result } = renderHook(() => useAgentForm());

      act(() => {
        result.current.setBrief('Make the hero section more compelling');
      });

      expect(result.current.form.brief).toBe('Make the hero section more compelling');
    });

    it('clears validation error when setting brief', () => {
      const { result } = renderHook(() => useAgentForm());

      // Trigger validation error by submitting empty form
      act(() => {
        result.current.handleSubmit();
      });

      expect(result.current.validationError).not.toBeNull();

      // Setting brief should clear validation error
      act(() => {
        result.current.setBrief('New brief');
      });

      expect(result.current.validationError).toBeNull();
    });

    it('updates assets', () => {
      const { result } = renderHook(() => useAgentForm());

      act(() => {
        result.current.setAssets('https://example.com/logo.png\nhttps://example.com/hero.jpg');
      });

      expect(result.current.form.assets).toBe(
        'https://example.com/logo.png\nhttps://example.com/hero.jpg'
      );
    });

    it('updates preview', () => {
      const { result } = renderHook(() => useAgentForm());

      expect(result.current.form.preview).toBe(true);

      act(() => {
        result.current.setPreview(false);
      });

      expect(result.current.form.preview).toBe(false);
    });
  });

  describe('handleSubmit', () => {
    it('returns validation error for empty brief', async () => {
      const { result } = renderHook(() => useAgentForm());

      let submitResult: { success: boolean; message?: string };
      await act(async () => {
        submitResult = await result.current.handleSubmit();
      });

      expect(submitResult!.success).toBe(false);
      expect(submitResult!.message).toBe('Please provide a brief for the agent');
      expect(result.current.validationError).toEqual({
        message: 'Please provide a brief for the agent',
        title: 'Missing Input',
      });
    });

    it('submits successfully with valid brief', async () => {
      const mockResult = {
        job_id: 'job_123',
        status: 'pending',
        agent_id: 'agent_abc',
      };
      mockSubmitAgentCustomization.mockResolvedValue(mockResult);

      const { result } = renderHook(() => useAgentForm());

      act(() => {
        result.current.setBrief('Make the hero section more compelling');
      });

      let submitResult: { success: boolean; message?: string };
      await act(async () => {
        submitResult = await result.current.handleSubmit();
      });

      expect(submitResult!.success).toBe(true);
      expect(result.current.result).toEqual(mockResult);
      expect(mockSubmitAgentCustomization).toHaveBeenCalledWith(
        'landing-page',
        'Make the hero section more compelling',
        [],
        true
      );
    });

    it('submits with assets', async () => {
      const mockResult = {
        job_id: 'job_123',
        status: 'pending',
        agent_id: 'agent_abc',
      };
      mockSubmitAgentCustomization.mockResolvedValue(mockResult);

      const { result } = renderHook(() => useAgentForm());

      act(() => {
        result.current.setBrief('Add logo');
        result.current.setAssets('https://example.com/logo.png\nhttps://example.com/hero.jpg');
      });

      await act(async () => {
        await result.current.handleSubmit();
      });

      expect(mockSubmitAgentCustomization).toHaveBeenCalledWith(
        'landing-page',
        'Add logo',
        ['https://example.com/logo.png', 'https://example.com/hero.jpg'],
        true
      );
    });

    it('submits with preview disabled', async () => {
      const mockResult = {
        job_id: 'job_123',
        status: 'pending',
        agent_id: 'agent_abc',
      };
      mockSubmitAgentCustomization.mockResolvedValue(mockResult);

      const { result } = renderHook(() => useAgentForm());

      act(() => {
        result.current.setBrief('Test brief');
        result.current.setPreview(false);
      });

      await act(async () => {
        await result.current.handleSubmit();
      });

      expect(mockSubmitAgentCustomization).toHaveBeenCalledWith(
        'landing-page',
        'Test brief',
        [],
        false
      );
    });

    it('handles API error', async () => {
      mockSubmitAgentCustomization.mockRejectedValue(new Error('Network error'));

      const { result } = renderHook(() => useAgentForm());

      act(() => {
        result.current.setBrief('Test brief');
      });

      let submitResult: { success: boolean; message?: string };
      await act(async () => {
        submitResult = await result.current.handleSubmit();
      });

      expect(submitResult!.success).toBe(false);
      expect(submitResult!.message).toBe('Network error');
      expect(result.current.error).toBe('Network error');
      expect(result.current.result).toBeNull();
    });

    it('sets submitting state during submit', async () => {
      let resolvePromise: () => void;
      mockSubmitAgentCustomization.mockReturnValue(
        new Promise((resolve) => {
          resolvePromise = () =>
            resolve({
              job_id: 'job_123',
              status: 'pending',
              agent_id: 'agent_abc',
            });
        })
      );

      const { result } = renderHook(() => useAgentForm());

      act(() => {
        result.current.setBrief('Test brief');
      });

      let submitPromise: Promise<{ success: boolean; message?: string }>;
      act(() => {
        submitPromise = result.current.handleSubmit();
      });

      expect(result.current.submitting).toBe(true);

      await act(async () => {
        resolvePromise!();
        await submitPromise;
      });

      expect(result.current.submitting).toBe(false);
    });

    it('trims brief before submitting', async () => {
      const mockResult = {
        job_id: 'job_123',
        status: 'pending',
        agent_id: 'agent_abc',
      };
      mockSubmitAgentCustomization.mockResolvedValue(mockResult);

      const { result } = renderHook(() => useAgentForm());

      act(() => {
        result.current.setBrief('  Test brief with spaces  ');
      });

      await act(async () => {
        await result.current.handleSubmit();
      });

      expect(mockSubmitAgentCustomization).toHaveBeenCalledWith(
        'landing-page',
        'Test brief with spaces',
        [],
        true
      );
    });
  });

  describe('resetForm', () => {
    it('resets form to default values', () => {
      const { result } = renderHook(() => useAgentForm());

      act(() => {
        result.current.setBrief('Test brief');
        result.current.setAssets('https://example.com/logo.png');
        result.current.setPreview(false);
      });

      act(() => {
        result.current.resetForm();
      });

      expect(result.current.form).toEqual({
        brief: '',
        assets: '',
        preview: true,
      });
    });

    it('clears validation error', () => {
      const { result } = renderHook(() => useAgentForm());

      // Trigger validation error
      act(() => {
        result.current.handleSubmit();
      });

      expect(result.current.validationError).not.toBeNull();

      act(() => {
        result.current.resetForm();
      });

      expect(result.current.validationError).toBeNull();
    });

    it('clears error', async () => {
      mockSubmitAgentCustomization.mockRejectedValue(new Error('Test error'));

      const { result } = renderHook(() => useAgentForm());

      act(() => {
        result.current.setBrief('Test');
      });

      await act(async () => {
        await result.current.handleSubmit();
      });

      expect(result.current.error).toBe('Test error');

      act(() => {
        result.current.resetForm();
      });

      expect(result.current.error).toBeNull();
    });
  });

  describe('clearResult', () => {
    it('clears result and resets form', async () => {
      const mockResult = {
        job_id: 'job_123',
        status: 'pending',
        agent_id: 'agent_abc',
      };
      mockSubmitAgentCustomization.mockResolvedValue(mockResult);

      const { result } = renderHook(() => useAgentForm());

      act(() => {
        result.current.setBrief('Test brief');
      });

      await act(async () => {
        await result.current.handleSubmit();
      });

      expect(result.current.result).not.toBeNull();

      act(() => {
        result.current.clearResult();
      });

      expect(result.current.result).toBeNull();
      expect(result.current.form.brief).toBe('');
    });
  });

  describe('clearError', () => {
    it('clears error state', async () => {
      mockSubmitAgentCustomization.mockRejectedValue(new Error('Test error'));

      const { result } = renderHook(() => useAgentForm());

      act(() => {
        result.current.setBrief('Test');
      });

      await act(async () => {
        await result.current.handleSubmit();
      });

      expect(result.current.error).toBe('Test error');

      act(() => {
        result.current.clearError();
      });

      expect(result.current.error).toBeNull();
    });
  });

  describe('clearValidationError', () => {
    it('clears validation error state', () => {
      const { result } = renderHook(() => useAgentForm());

      // Trigger validation error
      act(() => {
        result.current.handleSubmit();
      });

      expect(result.current.validationError).not.toBeNull();

      act(() => {
        result.current.clearValidationError();
      });

      expect(result.current.validationError).toBeNull();
    });
  });
});
