import { useState, useEffect } from 'react';
import { Circle, Compass, ListOrdered, Zap } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { NonePanel } from './panels/NonePanel';
import { ManualPanel } from './panels/ManualPanel';
import { ProfilePanel } from './panels/ProfilePanel';
import { QueuePanel } from './panels/QueuePanel';
import type { SteeringConfig, SteeringStrategy, AutoSteerProfile, PhaseInfo } from '@/types/api';

interface SteeringConfigDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  value: SteeringConfig;
  onChange: (config: SteeringConfig) => void;
  profiles: AutoSteerProfile[];
  phaseNames: PhaseInfo[];
  isLoadingProfiles?: boolean;
  isLoadingPhases?: boolean;
}

const STRATEGY_TABS: { value: SteeringStrategy; label: string; icon: React.ElementType }[] = [
  { value: 'none', label: 'None', icon: Circle },
  { value: 'manual', label: 'Manual', icon: Compass },
  { value: 'queue', label: 'Queue', icon: ListOrdered },
  { value: 'profile', label: 'Profile', icon: Zap },
];

function isValidConfig(config: SteeringConfig): boolean {
  switch (config.strategy) {
    case 'none':
      return true;
    case 'manual':
      return !!config.manualMode;
    case 'queue':
      return Array.isArray(config.queue) && config.queue.length > 0;
    case 'profile':
      return !!config.profileId;
    default:
      return false;
  }
}

export function SteeringConfigDialog({
  open,
  onOpenChange,
  value,
  onChange,
  profiles,
  phaseNames,
  isLoadingProfiles,
  isLoadingPhases,
}: SteeringConfigDialogProps) {
  // Local state for editing
  const [localConfig, setLocalConfig] = useState<SteeringConfig>(value);

  // Reset local state when dialog opens
  useEffect(() => {
    if (open) {
      setLocalConfig(value);
    }
  }, [open, value]);

  const handleStrategyChange = (strategy: string) => {
    setLocalConfig((prev) => ({
      ...prev,
      strategy: strategy as SteeringStrategy,
    }));
  };

  const handleApply = () => {
    onChange(localConfig);
    onOpenChange(false);
  };

  const handleCancel = () => {
    setLocalConfig(value);
    onOpenChange(false);
  };

  const canApply = isValidConfig(localConfig);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Configure Steering</DialogTitle>
          <DialogDescription>Choose how task execution should be guided.</DialogDescription>
        </DialogHeader>

        <Tabs value={localConfig.strategy} onValueChange={handleStrategyChange} className="mt-2">
          <TabsList className="grid w-full grid-cols-4">
            {STRATEGY_TABS.map(({ value: tabValue, label, icon: Icon }) => (
              <TabsTrigger
                key={tabValue}
                value={tabValue}
                className="flex items-center gap-1.5 text-xs"
              >
                <Icon className="h-3.5 w-3.5" />
                {label}
              </TabsTrigger>
            ))}
          </TabsList>

          <div className="mt-4 min-h-[280px]">
            <TabsContent value="none" className="m-0">
              <NonePanel />
            </TabsContent>

            <TabsContent value="manual" className="m-0">
              <ManualPanel
                value={localConfig.manualMode}
                onChange={(mode) => setLocalConfig((prev) => ({ ...prev, manualMode: mode }))}
                phaseNames={phaseNames}
                isLoading={isLoadingPhases}
              />
            </TabsContent>

            <TabsContent value="queue" className="m-0">
              <QueuePanel
                value={localConfig.queue || []}
                onChange={(queue) => setLocalConfig((prev) => ({ ...prev, queue }))}
                phaseNames={phaseNames}
                isLoading={isLoadingPhases}
              />
            </TabsContent>

            <TabsContent value="profile" className="m-0">
              <ProfilePanel
                value={localConfig.profileId}
                onChange={(profileId) => setLocalConfig((prev) => ({ ...prev, profileId }))}
                profiles={profiles}
                isLoading={isLoadingProfiles}
              />
            </TabsContent>
          </div>
        </Tabs>

        <DialogFooter className="mt-4">
          <Button type="button" variant="outline" onClick={handleCancel}>
            Cancel
          </Button>
          <Button type="button" onClick={handleApply} disabled={!canApply}>
            Apply
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
