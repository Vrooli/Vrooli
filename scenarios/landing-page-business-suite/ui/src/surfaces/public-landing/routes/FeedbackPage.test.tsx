import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { Code, ConnectError } from '@connectrpc/connect';
import { renderWithProviders } from "@vrooli/api-base/testing";
import { createFeedback } from '../../../shared/api/feedback';
import { FeedbackPage } from './FeedbackPage';

vi.mock('../../../shared/api/feedback', () => ({ createFeedback: vi.fn() }));

function renderFeedbackPage() {
  return renderWithProviders(<FeedbackPage />, { route: '/feedback' });
}

function completeForm() {
  fireEvent.change(screen.getByPlaceholderText('you@example.com'), { target: { value: 'buyer@example.com' } });
  fireEvent.change(screen.getByPlaceholderText('How can we help?'), { target: { value: 'Subscription question' } });
  fireEvent.change(screen.getByPlaceholderText('Share your thoughts, questions, or suggestions.'), { target: { value: 'Please help with my subscription.' } });
}

describe('FeedbackPage', () => {
  beforeEach(() => {
    vi.mocked(createFeedback).mockReset();
  });

  it('submits a complete public feedback request and confirms durable receipt', async () => {
    vi.mocked(createFeedback).mockResolvedValue({ success: true, id: 7 });
    renderFeedbackPage();
    completeForm();

    fireEvent.click(screen.getByRole('button', { name: /send general/i }));

    await waitFor(() => {
      expect(createFeedback).toHaveBeenCalledTimes(1);
    });
    expect(createFeedback).toHaveBeenCalledWith(expect.objectContaining({ type: 'general', email: 'buyer@example.com' }), expect.any(AbortSignal));
    expect(await screen.findByText('Thank you for your feedback!')).toBeInTheDocument();
  });

  it('shows refund-specific fields and preserves the selected feedback type in the request', async () => {
    vi.mocked(createFeedback).mockResolvedValue({ success: true, id: 8 });
    renderFeedbackPage();
    fireEvent.click(screen.getByRole('button', { name: /request a refund/i }));
    expect(screen.getByText('30-Day Money-Back Guarantee')).toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText('you@example.com'), { target: { value: 'buyer@example.com' } });
    fireEvent.change(screen.getByPlaceholderText('Refund request for my subscription'), { target: { value: 'Refund please' } });
    fireEvent.change(screen.getByPlaceholderText("Tell us why you'd like a refund (optional but helpful for us to improve)."), { target: { value: 'Not a fit' } });
    fireEvent.change(screen.getByPlaceholderText('sub_xxxxx or cs_xxxxx'), { target: { value: 'sub_123' } });
    fireEvent.click(screen.getByRole('button', { name: /send request/i }));

    await waitFor(() => {
      expect(createFeedback).toHaveBeenCalledTimes(1);
    });
    expect(createFeedback).toHaveBeenCalledWith(expect.objectContaining({ type: 'refund', orderId: 'sub_123' }), expect.any(AbortSignal));
  });

  it('surfaces a server validation error without offering an unsafe retry', async () => {
    vi.mocked(createFeedback).mockRejectedValue(new ConnectError('Subject is required', Code.InvalidArgument));
    renderFeedbackPage();
    completeForm();
    fireEvent.click(screen.getByRole('button', { name: /send general/i }));

    expect(await screen.findByText('Subject is required')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /try again/i })).not.toBeInTheDocument();
  });

  it('offers retry after a recoverable network failure', async () => {
    vi.mocked(createFeedback).mockRejectedValue(new TypeError('offline'));
    renderFeedbackPage();
    completeForm();
    fireEvent.click(screen.getByRole('button', { name: /send general/i }));

    expect(await screen.findByText(/unable to reach the server/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
  });

  it('replays the preserved feedback request after retrying a transient failure', async () => {
    vi.mocked(createFeedback)
      .mockRejectedValueOnce(new TypeError('offline'))
      .mockResolvedValueOnce({ success: true, id: 9 });
    renderFeedbackPage();
    completeForm();
    fireEvent.click(screen.getByRole('button', { name: /send general/i }));

    await screen.findByText(/unable to reach the server/i);
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));

    await waitFor(() => {
      expect(createFeedback).toHaveBeenCalledTimes(2);
    });
    expect(await screen.findByText('Thank you for your feedback!')).toBeInTheDocument();
  });

  it.each([
    [new ConnectError('The feedback service is unavailable', Code.Unavailable), 'Server Error', true],
    [new Error('A local processing error occurred'), 'Submission Failed', true],
  ])('presents recoverable non-validation feedback failures safely', async (error, expectedTitle, retryable) => {
    vi.mocked(createFeedback).mockRejectedValueOnce(error);
    renderFeedbackPage();
    completeForm();
    fireEvent.click(screen.getByRole('button', { name: /send general/i }));

    expect(await screen.findByText(expectedTitle)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
    expect(retryable).toBe(true);
  });

  it('adapts feedback prompts and delivery language for bug and feature reports', () => {
    renderFeedbackPage();
    fireEvent.click(screen.getByRole('button', { name: /report a bug/i }));
    expect(screen.getByPlaceholderText('Bug: Describe the issue briefly')).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/describe the bug/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /feature request/i }));
    expect(screen.getByPlaceholderText('Feature idea: Your suggestion')).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/describe your feature idea/i)).toBeInTheDocument();
  });
});
