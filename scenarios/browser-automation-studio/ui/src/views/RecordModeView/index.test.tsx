import { fireEvent, screen, waitFor, act } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import toast from 'react-hot-toast';
import { renderWithProviders } from '@/test-utils/renderWithProviders';
import RecordModeView from './index';

vi.mock('@/config', () => ({ getConfig: async () => ({ API_URL: '/api/v1' }) }));
vi.mock('@/lib/profiler', () => ({ onProfilerRender: vi.fn() }));
vi.mock('@utils/logger', () => ({ logger: { warn: vi.fn() } }));
vi.mock('react-hot-toast', () => ({ default: { error: vi.fn(), success: vi.fn() } }));
vi.mock('@/domains/recording/RecordingSession', () => ({
  RecordModePage: ({ onClose, onWorkflowGenerated }: {
    onClose: () => Promise<void>;
    onWorkflowGenerated: (workflow: string, project: string) => Promise<void>;
  }) => <>
    <button onClick={onClose}>Close browser</button>
    <button onClick={() => onWorkflowGenerated('workflow', 'project')}>Finish workflow</button>
  </>,
}));

describe('profile recovery on close [REQ:BAS-RH-J06]', () => {
  const fetchMock = vi.fn<typeof fetch>();

  beforeEach(() => {
    vi.clearAllMocks();
    fetchMock.mockReset();
    vi.stubGlobal('fetch', fetchMock);
    sessionStorage.clear();
  });
  afterEach(() => vi.unstubAllGlobals());

  async function openBrowser() {
    renderWithProviders(<MemoryRouter initialEntries={['/record/session']}><Routes>
      <Route path="/record/:sessionId" element={<RecordModeView />} />
      <Route path="/" element={<div>Dashboard</div>} />
      <Route path="/projects/:projectId" element={<div>Saved workflow</div>} />
    </Routes></MemoryRouter>, { withoutRouter: true });
    await screen.findByRole('button', { name: 'Close browser' });
  }

  it.each(['Close browser', 'Finish workflow'])('keeps the browser reachable when %s is rejected', async (button) => {
    fetchMock.mockResolvedValue(new Response('save failed', { status: 500 }));
    await openBrowser();
    fireEvent.click(screen.getByRole('button', { name: button }));
    await waitFor(() => expect(toast.error).toHaveBeenCalled());
    expect(screen.getByRole('button', { name: 'Close browser' })).toBeInTheDocument();
    expect(screen.queryByText('Dashboard')).not.toBeInTheDocument();
    expect(screen.queryByText('Saved workflow')).not.toBeInTheDocument();
  });

  it('keeps the browser reachable after a transport failure', async () => {
    fetchMock.mockRejectedValue(new TypeError('connection lost'));
    await openBrowser();
    fireEvent.click(screen.getByRole('button', { name: 'Close browser' }));
    await waitFor(() => expect(toast.error).toHaveBeenCalled());
    expect(screen.getByRole('button', { name: 'Close browser' })).toBeInTheDocument();
  });

  it('retries the same session after a failed save and then returns to the dashboard', async () => {
    fetchMock.mockResolvedValueOnce(new Response(null, { status: 500 }))
      .mockResolvedValueOnce(new Response(null, { status: 200 }));
    await openBrowser();
    fireEvent.click(screen.getByRole('button', { name: 'Close browser' }));
    await waitFor(() => expect(toast.error).toHaveBeenCalled());
    fireEvent.click(screen.getByRole('button', { name: 'Close browser' }));
    await screen.findByText('Dashboard');
    expect(fetchMock).toHaveBeenCalledTimes(2);
    for (const call of fetchMock.mock.calls) {
      expect(call).toEqual(['/api/v1/recordings/live/session/session/close', { method: 'POST' }]);
    }
  });

  it('admits one close while persistence is pending', async () => {
    let finish!: (response: Response) => void;
    fetchMock.mockReturnValue(new Promise((resolve) => { finish = resolve; }));
    await openBrowser();
    fireEvent.click(screen.getByRole('button', { name: 'Close browser' }));
    fireEvent.click(screen.getByRole('button', { name: 'Close browser' }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    await act(async () => { finish(new Response(null, { status: 200 })); });
    await screen.findByText('Dashboard');
  });

  it('opens the saved workflow after the session closes successfully', async () => {
    fetchMock.mockResolvedValue(new Response(null, { status: 200 }));
    await openBrowser();
    fireEvent.click(screen.getByRole('button', { name: 'Finish workflow' }));
    await screen.findByText('Saved workflow');
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(toast.success).toHaveBeenCalled();
  });
});
