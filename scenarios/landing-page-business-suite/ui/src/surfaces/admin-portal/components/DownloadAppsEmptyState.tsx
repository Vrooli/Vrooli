import { Download, Monitor, Package, Plus, Smartphone } from 'lucide-react';
import { Button } from '../../../shared/ui/button';
import { EmptyState } from '../../../shared/ui/EmptyState';

interface DownloadAppsEmptyStateProps {
  onAddApp: () => void;
}

export function DownloadAppsEmptyState({ onAddApp }: DownloadAppsEmptyStateProps) {
  return (
    <div data-testid="downloads-empty-state">
      <EmptyState
        title="No download apps configured yet"
        description="Add your first app to display download options on your landing page."
        icon={<Download className="h-12 w-12" />}
        action={(
          <Button onClick={onAddApp} className="gap-2">
          <Plus className="h-4 w-4" />
          Add Your First App
          </Button>
        )}
        className="items-center rounded-3xl border-white/10 bg-white/5 p-8 text-center"
      />
      <div className="mt-8 border-t border-white/10 pt-6">
        <p className="text-xs uppercase tracking-[0.3em] text-slate-500 mb-4">What you'll configure</p>
        <div className="grid gap-4 sm:grid-cols-3 text-sm">
          <DownloadSetupCapability icon={Package} title="App identity" description="Name, tagline, description, and install instructions" className="text-blue-300" />
          <DownloadSetupCapability icon={Monitor} title="Desktop installers" description="Windows, Mac, and Linux artifact URLs with versions" className="text-emerald-300" />
          <DownloadSetupCapability icon={Smartphone} title="Mobile store links" description="App Store and Google Play URLs (optional)" className="text-purple-300" />
        </div>
      </div>
    </div>
  );
}

interface DownloadSetupCapabilityProps {
  icon: typeof Package;
  title: string;
  description: string;
  className: string;
}

function DownloadSetupCapability({ icon: Icon, title, description, className }: DownloadSetupCapabilityProps) {
  return (
    <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4">
      <Icon className={`h-5 w-5 mb-2 ${className}`} />
      <p className="font-semibold text-white">{title}</p>
      <p className="text-xs text-slate-400 mt-1">{description}</p>
    </div>
  );
}
