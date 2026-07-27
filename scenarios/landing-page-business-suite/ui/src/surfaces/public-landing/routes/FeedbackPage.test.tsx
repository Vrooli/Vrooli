import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders } from '../../../test-utils/renderWithProviders';
import { FeedbackPage } from './FeedbackPage';

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
    vi.unstubAllGlobals();
    vi.stubGlobal('fetch', vi.fn());
  });

  it('submits a complete public feedback request and confirms durable receipt', async () => {
    const fetchMock = vi.mocked(fetch).mockResolvedValue(new Response(JSON.stringify({ success: true, id: 7 }), { status: 201 }));
    renderFeedbackPage();
    completeForm();

    fireEvent.click(screen.getByRole('button', { name: /send general/i }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(1);
    });
    expect(fetchMock).toHaveBeenCalledWith('/api/feedback', expect.objectContaining({ method: 'POST' }));
    expect(await screen.findByText('Thank you for your feedback!')).toBeInTheDocument();
  });

  it('shows refund-specific fields and preserves the selected feedback type in the request', async () => {
    const fetchMock = vi.mocked(fetch).mockResolvedValue(new Response(JSON.stringify({ success: true, id: 8 }), { status: 201 }));
    renderFeedbackPage();
    fireEvent.click(screen.getByRole('button', { name: /request a refund/i }));
    expect(screen.getByText('30-Day Money-Back Guarantee')).toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText('you@example.com'), { target: { value: 'buyer@example.com' } });
    fireEvent.change(screen.getByPlaceholderText('Refund request for my subscription'), { target: { value: 'Refund please' } });
    fireEvent.change(screen.getByPlaceholderText("Tell us why you'd like a refund (optional but helpful for us to improve)."), { target: { value: 'Not a fit' } });
    fireEvent.change(screen.getByPlaceholderText('sub_xxxxx or cs_xxxxx'), { target: { value: 'sub_123' } });
    fireEvent.click(screen.getByRole('button', { name: /send request/i }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(1);
    });
    const requestInit = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(JSON.parse(requestInit.body as string)).toMatchObject({ type: 'refund', orderId: 'sub_123' });
  });

  it('surfaces a server validation error without offering an unsafe retry', async () => {
    vi.mocked(fetch).mockResolvedValue(new Response(JSON.stringify({ error: 'Subject is required' }), { status: 400 }));
    renderFeedbackPage();
    completeForm();
    fireEvent.click(screen.getByRole('button', { name: /send general/i }));

    expect(await screen.findByText('Subject is required')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /try again/i })).not.toBeInTheDocument();
  });

  it('offers retry after a recoverable network failure', async () => {
    vi.mocked(fetch).mockRejectedValue(new TypeError('offline'));
    renderFeedbackPage();
    completeForm();
    fireEvent.click(screen.getByRole('button', { name: /send general/i }));

    expect(await screen.findByText(/unable to reach the server/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
  });
});
