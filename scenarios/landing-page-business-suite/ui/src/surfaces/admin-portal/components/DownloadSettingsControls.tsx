import { Download, Package } from 'lucide-react';
import { Button } from '../../../shared/ui/button';
import { Callout } from './Callout';

interface DownloadSettingsControlsProps {
  activeTab: 'apps' | 'hosting';
  onTabChange: (tab: 'apps' | 'hosting') => void;
  dirtyCount: number;
  error: string | null;
}

export function DownloadSettingsControls({
  activeTab,
  onTabChange,
  dirtyCount,
  error,
}: DownloadSettingsControlsProps) {
  return (
    <>
      <div className="flex flex-wrap gap-2" data-testid="downloads-tabs">
        <Button
          size="sm"
          variant={activeTab === 'apps' ? 'default' : 'outline'}
          onClick={() => { onTabChange('apps'); }}
          className="gap-2"
        >
          <Download className="h-4 w-4" />
          Apps
        </Button>
        <Button
          size="sm"
          variant={activeTab === 'hosting' ? 'default' : 'outline'}
          onClick={() => { onTabChange('hosting'); }}
          className="gap-2"
        >
          <Package className="h-4 w-4" />
          Hosting
        </Button>
      </div>
      {activeTab === 'apps' && dirtyCount > 0 && (
        <Callout
          type="warning"
          message={`${String(dirtyCount)} app${dirtyCount === 1 ? '' : 's'} have unsaved changes. Save each card to update the runtime payload.`}
        />
      )}
      {error && <Callout type="error" message={error} />}
    </>
  );
}
