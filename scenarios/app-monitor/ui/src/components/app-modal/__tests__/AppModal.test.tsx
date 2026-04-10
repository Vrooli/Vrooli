import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import AppModal from '@/components/AppModal';
import { createMockApp, createMockDiagnostics, createMockDiagnosticsWithWarnings } from './fixtures';

// Stub buildPreviewUrl so it doesn't depend on app internals
vi.mock('@/utils/appPreview', () => ({
  buildPreviewUrl: () => null,
}));

// Stub heavy tab internals that depend on API services
vi.mock('@/components/tabs/LighthouseTab', () => ({
  default: () => <div data-testid="lighthouse-tab">LighthouseTab</div>,
}));

const defaultProps = () => ({
  app: createMockApp(),
  isOpen: true,
  onClose: vi.fn(),
  onAction: vi.fn(async () => {}),
  onViewLogs: vi.fn(),
});

describe('AppModal', () => {
  it('renders the app name in the header', () => {
    render(<AppModal {...defaultProps()} />);
    expect(screen.getByText('Test App')).toBeInTheDocument();
  });

  it('calls onClose when Escape is pressed', async () => {
    const props = defaultProps();
    render(<AppModal {...props} />);
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' });
    await waitFor(() => {
      expect(props.onClose).toHaveBeenCalled();
    });
  });

  it('renders nothing when isOpen is false', () => {
    const { container } = render(<AppModal {...defaultProps()} isOpen={false} />);
    expect(container.innerHTML).toBe('');
  });

  it('shows all tab buttons', () => {
    render(<AppModal {...defaultProps()} />);
    expect(screen.getByRole('tab', { name: /overview/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /diagnostics/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /tech stack/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /docs/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /performance/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /completeness/i })).toBeInTheDocument();
  });

  it('switches tabs when clicked', async () => {
    const user = userEvent.setup();
    render(<AppModal {...defaultProps()} />);

    const diagnosticsTab = screen.getByRole('tab', { name: /diagnostics/i });
    await user.click(diagnosticsTab);

    expect(diagnosticsTab).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByRole('tab', { name: /overview/i })).toHaveAttribute('aria-selected', 'false');
  });

  it('shows diagnostics badge with warning count', () => {
    render(
      <AppModal
        {...defaultProps()}
        preloadedDiagnostics={createMockDiagnosticsWithWarnings(5)}
      />,
    );
    const badge = screen.getByText('5');
    expect(badge).toHaveClass('modal-tab-badge--warn');
  });

  it('shows docs badge with info variant', () => {
    render(
      <AppModal
        {...defaultProps()}
        preloadedDiagnostics={createMockDiagnostics({
          documents: { total: 12, root_docs: [], docs_docs: [] },
        })}
      />,
    );
    const badge = screen.getByText('12');
    expect(badge).toHaveClass('modal-tab-badge--info');
  });

  it('calls onAction with start when Start is clicked', async () => {
    const props = defaultProps();
    props.app = createMockApp({ status: 'stopped' });
    const user = userEvent.setup();

    render(<AppModal {...props} />);
    await user.click(screen.getByText('Start'));

    expect(props.onAction).toHaveBeenCalledWith('test-app', 'start');
  });

  it('calls onViewLogs when Logs is clicked', async () => {
    const props = defaultProps();
    const user = userEvent.setup();

    render(<AppModal {...props} />);
    await user.click(screen.getByText('Logs'));

    expect(props.onViewLogs).toHaveBeenCalledWith('test-app');
  });

  it('copies preview URL to clipboard', async () => {
    const writeText = vi.fn(() => Promise.resolve());
    const originalClipboard = navigator.clipboard;
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText },
      writable: true,
      configurable: true,
    });

    try {
      render(
        <AppModal
          {...defaultProps()}
          previewUrl="http://localhost:3000"
        />,
      );

      // Use fireEvent instead of userEvent — userEvent.setup() replaces
      // navigator.clipboard with its own mock, bypassing our spy.
      fireEvent.click(screen.getByLabelText('Copy preview URL'));
      await waitFor(() => {
        expect(writeText).toHaveBeenCalledWith('http://localhost:3000');
      });
    } finally {
      Object.defineProperty(navigator, 'clipboard', {
        value: originalClipboard,
        writable: true,
        configurable: true,
      });
    }
  });

  it('renders tab panels with proper ARIA attributes', () => {
    render(<AppModal {...defaultProps()} />);

    const overviewPanel = screen.getByRole('tabpanel', { name: /overview/i });
    expect(overviewPanel).toHaveAttribute('id', 'tabpanel-overview');
    expect(overviewPanel).toHaveAttribute('aria-labelledby', 'tab-overview');
  });
});
