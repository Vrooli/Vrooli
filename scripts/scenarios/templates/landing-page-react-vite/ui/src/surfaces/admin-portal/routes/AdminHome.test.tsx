import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { AdminHome } from './AdminHome';
import { AdminAuthProvider } from '../../../app/providers/AdminAuthProvider';

const mockNavigate = vi.fn();

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

vi.mock('../components/RuntimeSignalStrip', () => ({
  RuntimeSignalStrip: () => <div data-testid="runtime-signal-mock" />,
}));

const renderWithRouter = (component: React.ReactElement) => {
  return render(
    <BrowserRouter>
      <AdminAuthProvider>
        {component}
      </AdminAuthProvider>
    </BrowserRouter>
  );
};

describe('AdminHome [REQ:ADMIN-MODES]', () => {
  const originalFetch = global.fetch;
  const originalLocation = window.location;

  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn().mockResolvedValue({ ok: false } as Response);
    delete (window as { location?: Location }).location;
    window.location = { ...originalLocation, pathname: '/admin' };
    window.localStorage.clear();
  });

  afterEach(() => {
    global.fetch = originalFetch;
    window.location = originalLocation;
    window.localStorage.clear();
  });

  it('[REQ:ADMIN-MODES] should display exactly two modes: Analytics and Customization', () => {
    renderWithRouter(<AdminHome />);

    expect(screen.getByText('Analytics / Metrics')).toBeInTheDocument();
    // Use role query to avoid ambiguity - there's a heading and button text with "Customization"
    expect(screen.getByRole('heading', { name: 'Customization' })).toBeInTheDocument();

    const modeButtons = screen.getAllByRole('button').filter(btn =>
      btn.getAttribute('data-testid')?.startsWith('admin-mode-')
    );
    expect(modeButtons).toHaveLength(2);
  });

  it('[REQ:ADMIN-NAV] should navigate to analytics when Analytics mode is clicked', async () => {
    const user = userEvent.setup();
    renderWithRouter(<AdminHome />);

    const analyticsButton = screen.getByTestId('admin-mode-analytics');
    await user.click(analyticsButton);

    expect(mockNavigate).toHaveBeenCalledWith('/admin/analytics');
  });

  it('[REQ:ADMIN-NAV] should navigate to customization when Customization mode is clicked', async () => {
    const user = userEvent.setup();
    renderWithRouter(<AdminHome />);

    const customizationButton = screen.getByTestId('admin-mode-customization');
    await user.click(customizationButton);

    expect(mockNavigate).toHaveBeenCalledWith('/admin/customization');
  });

  it('should display mode descriptions', () => {
    renderWithRouter(<AdminHome />);

    expect(screen.getByText(/View conversion rates, A\/B test results/)).toBeInTheDocument();
    expect(screen.getByText(/Customize landing page content, trigger agent-based/)).toBeInTheDocument();
  });

  it('should surface quick resume panel when recents exist', async () => {
    window.localStorage.setItem(
      'landing_admin_experience',
      JSON.stringify({
        version: 1,
        lastVariant: {
          slug: 'alpha',
          name: 'Variant Alpha',
          surface: 'variant',
          lastVisitedAt: new Date().toISOString(),
        },
        lastAnalytics: {
          variantSlug: 'beta',
          variantName: 'Variant Beta',
          timeRangeDays: 30,
          savedAt: new Date().toISOString(),
        },
      })
    );

    renderWithRouter(<AdminHome />);

    expect(await screen.findByTestId('admin-resume-panel')).toBeInTheDocument();
    expect(screen.getByTestId('admin-resume-customization')).toBeInTheDocument();
    expect(screen.getByTestId('admin-resume-analytics')).toBeInTheDocument();
  });
});
