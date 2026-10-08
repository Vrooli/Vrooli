import { render, screen, waitFor } from '@/test-utils';
import NavigatePreviewPanel from './NavigatePreviewPanel';

vi.mock('@/config', () => ({ getConfig: async () => ({ API_URL: 'http://api.test' }) }));
vi.mock('@stores/scenarioStore', () => ({ useScenarioStore: (selector: (state: unknown) => unknown) => selector({ scenarios: [{ name: 'demo-app', description: 'Demo application', status: 'running' }], fetchScenarios: vi.fn().mockResolvedValue(undefined) }) }));
vi.mock('@/api/scenarios', () => ({ scenariosClient: { getPort: vi.fn().mockResolvedValue({ url: '', port: 8123 }) } }));

describe('NavigatePreviewPanel', () => {
  it('captures and visibly renders a preview for a URL destination', async () => {
    const screenshot = vi.fn();
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ screenshot: 'data:image/png;base64,nav', consoleLogs: [{ level: 'log', message: 'ready' }] }) });
    vi.stubGlobal('fetch', fetchMock);
    render(<NavigatePreviewPanel destinationType="NAVIGATE_DESTINATION_TYPE_URL" url="https://example.test" scenario="" scenarioPath="" onChange={vi.fn()} onScreenshot={screenshot} />);

    expect(await screen.findByAltText('Navigation target preview')).toHaveAttribute('src', 'data:image/png;base64,nav');
    await waitFor(() => expect(screenshot).toHaveBeenCalledWith(expect.objectContaining({ previewScreenshot: 'data:image/png;base64,nav', previewScreenshotSourceUrl: 'https://example.test' })));
    expect(fetchMock).toHaveBeenCalledWith('http://api.test/preview-screenshot', expect.objectContaining({ method: 'POST' }));
  });

  it('resolves a selected scenario and path before capturing its navigation preview', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ screenshot: 'data:image/png;base64,scenario' }) });
    vi.stubGlobal('fetch', fetchMock);
    render(<NavigatePreviewPanel destinationType="NAVIGATE_DESTINATION_TYPE_SCENARIO" url="" scenario="demo-app" scenarioPath="/account" onChange={vi.fn()} onScreenshot={vi.fn()} />);

    expect(await screen.findByAltText('Navigation target preview')).toHaveAttribute('src', 'data:image/png;base64,scenario');
    expect(fetchMock).toHaveBeenCalledWith('http://api.test/preview-screenshot', expect.objectContaining({ body: JSON.stringify({ url: 'http://localhost:8123/account', viewport: undefined }) }));
  });
});
