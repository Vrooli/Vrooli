/**
 * SettingsView - Route wrapper for the settings page.
 * Uses the refactored SettingsView component with extracted sections.
 *
 * Tab state is driven by URL query params (?tab=subscription, etc.)
 * This enables deep-linking to specific tabs and proper browser history.
 */
import { lazy, Suspense, useCallback, useMemo } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { LoadingSpinner } from '@shared/ui';
import { selectors } from '@constants/selectors';

const SettingsViewContent = lazy(() => import('./SettingsView'));

/** Valid settings tab IDs - must match SettingsTab type in SettingsView.tsx */
const VALID_TABS = ['display', 'replay', 'branding', 'workflow', 'apikeys', 'data', 'sessions', 'subscription', 'schedules', 'diagnostics'] as const;
type SettingsTab = (typeof VALID_TABS)[number];

function isValidTab(tab: string | null): tab is SettingsTab {
  return tab !== null && VALID_TABS.includes(tab as SettingsTab);
}

export default function SettingsView() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();

  // Read current tab from URL query parameter (URL is source of truth)
  const activeTab = useMemo((): SettingsTab => {
    const tab = searchParams.get('tab');
    return isValidTab(tab) ? tab : 'display';
  }, [searchParams]);

  // Update URL when tab changes
  const handleTabChange = useCallback((tab: string) => {
    if (isValidTab(tab)) {
      setSearchParams({ tab }, { replace: true });
    }
  }, [setSearchParams]);

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
        <SettingsViewContent
          onBack={handleBack}
          activeTab={activeTab}
          onTabChange={handleTabChange}
        />
      </Suspense>
    </div>
  );
}

// Also export sections for use elsewhere
export * from './sections';
