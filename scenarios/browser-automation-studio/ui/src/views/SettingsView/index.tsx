/**
 * SettingsView - Route wrapper for the settings page.
 * Uses the refactored SettingsView component with extracted sections.
 */
import { lazy, Suspense } from 'react';
import { useNavigate } from 'react-router-dom';
import { LoadingSpinner } from '@shared/ui';
import { selectors } from '@constants/selectors';

const SettingsViewContent = lazy(() => import('./SettingsView'));

export default function SettingsView() {
  const navigate = useNavigate();

  const handleBack = () => {
    navigate('/');
  };

  return (
    <div data-testid={selectors.app.shell.ready}>
      <Suspense
        fallback={
          <div className="h-screen flex items-center justify-center bg-flow-bg">
            <LoadingSpinner variant="default" size={24} message="Loading settings..." />
          </div>
        }
      >
        <SettingsViewContent onBack={handleBack} />
      </Suspense>
    </div>
  );
}

// Also export sections for use elsewhere
export * from './sections';
