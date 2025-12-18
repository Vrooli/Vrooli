/**
 * SettingsView - Route wrapper for the settings page.
 * Uses the refactored SettingsView component with extracted sections.
 */
import { lazy, Suspense, useCallback, useMemo } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { LoadingSpinner } from '@shared/ui';
import { selectors } from '@constants/selectors';

const SettingsViewContent = lazy(() => import('./SettingsView'));

export default function SettingsView() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  // Read initial tab from URL query parameter
  const initialTab = useMemo(() => {
    const tab = searchParams.get('tab');
    // Validate it's a known tab, otherwise return undefined
    const validTabs = ['display', 'replay', 'branding', 'workflow', 'apikeys', 'data', 'sessions', 'subscription', 'schedules'];
    return tab && validTabs.includes(tab) ? tab : undefined;
  }, [searchParams]);

  const handleBack = useCallback(() => {
    // Go back in history if possible, otherwise navigate home
    if (window.history.length > 2) {
      navigate(-1);
    } else {
      navigate('/');
    }
  }, [navigate]);

  return (
    <div data-testid={selectors.app.shell.ready}>
      <Suspense
        fallback={
          <div className="h-screen flex items-center justify-center bg-flow-bg">
            <LoadingSpinner variant="default" size={24} message="Loading settings..." />
          </div>
        }
      >
        <SettingsViewContent onBack={handleBack} initialTab={initialTab} />
      </Suspense>
    </div>
  );
}

// Also export sections for use elsewhere
export * from './sections';
