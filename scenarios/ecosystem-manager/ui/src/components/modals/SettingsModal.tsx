/**
 * SettingsModal Component
 * Main settings modal with tabs for processor, agent, display, and other settings
 */

import { useState, useEffect } from 'react';
import { Settings as SettingsIcon, Save, RotateCcw } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Button } from '@/components/ui/button';
import { useSettings, useSaveSettings, useResetSettings } from '@/hooks/useSettings';
import { ProcessorTab } from './settings/ProcessorTab';
import { AgentTab } from './settings/AgentTab';
import { DisplayTab } from './settings/DisplayTab';
import { PromptTesterTab } from './settings/PromptTesterTab';
import { RateLimitsTab } from './settings/RateLimitsTab';
import { RecyclerTab } from './settings/RecyclerTab';
import { AutoSteerTab } from './settings/AutoSteerTab';
import type { Settings } from '@/types/api';
import { useAppState } from '@/contexts/AppStateContext';

interface SettingsModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function SettingsModal({ open, onOpenChange }: SettingsModalProps) {
  const { data: settings, isLoading } = useSettings();
  const saveSettings = useSaveSettings();
  const resetSettings = useResetSettings();
  const { setCachedSettings } = useAppState();

  const [localSettings, setLocalSettings] = useState<Settings | null>(null);
  const [activeTab, setActiveTab] = useState('processor');

  // Initialize local settings when modal opens or settings load
  useEffect(() => {
    if (open && settings) {
      setLocalSettings(JSON.parse(JSON.stringify(settings))); // Deep clone
    }
  }, [open, settings]);

  const handleSave = () => {
    if (localSettings) {
      saveSettings.mutate(localSettings, {
        onSuccess: (updated) => {
          setCachedSettings(updated);
          onOpenChange(false);
        },
      });
    }
  };

  const handleReset = () => {
    resetSettings.mutate(undefined, {
      onSuccess: (resetData) => {
        setLocalSettings(JSON.parse(JSON.stringify(resetData)));
        setCachedSettings(resetData);
      },
    });
  };

  const handleCancel = () => {
    onOpenChange(false);
  };

  const updateLocalSettings = <K extends keyof Settings>(
    section: K,
    updates: Partial<Settings[K]>
  ) => {
    if (!localSettings) return;
    setLocalSettings({
      ...localSettings,
      [section]: {
        ...localSettings[section],
        ...updates,
      },
    });
  };

  if (isLoading || !localSettings) {
    return (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-4xl max-h-[90vh]">
          <div className="flex items-center justify-center h-64">
            <div className="text-slate-400">Loading settings...</div>
          </div>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="w-[calc(100vw-2rem)] sm:max-w-5xl md:max-w-6xl max-h-[90vh]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <SettingsIcon className="h-5 w-5" />
            Settings
          </DialogTitle>
          <DialogDescription>
            Configure processor, agent, display, and recycler settings
          </DialogDescription>
        </DialogHeader>

        <Tabs value={activeTab} onValueChange={setActiveTab} className="flex-1">
          <TabsList className="grid w-full grid-cols-7">
            <TabsTrigger value="processor">Processor</TabsTrigger>
            <TabsTrigger value="agent">Agent</TabsTrigger>
            <TabsTrigger value="display">Display</TabsTrigger>
            <TabsTrigger value="prompt-tester">Prompt</TabsTrigger>
            <TabsTrigger value="rate-limits">Limits</TabsTrigger>
            <TabsTrigger value="recycler">Recycler</TabsTrigger>
            <TabsTrigger value="auto-steer">Auto Steer</TabsTrigger>
          </TabsList>

          <div className="mt-4 overflow-y-auto max-h-[calc(90vh-280px)]">
            <TabsContent value="processor" className="space-y-4">
              <ProcessorTab
                settings={localSettings.processor}
                onChange={(updates) => updateLocalSettings('processor', updates)}
              />
            </TabsContent>

            <TabsContent value="agent" className="space-y-4">
              <AgentTab
                settings={localSettings.agent}
                onChange={(updates) => updateLocalSettings('agent', updates)}
              />
            </TabsContent>

            <TabsContent value="display" className="space-y-4">
              <DisplayTab
                settings={localSettings.display}
                onChange={(updates) => updateLocalSettings('display', updates)}
              />
            </TabsContent>

            <TabsContent value="prompt-tester" className="space-y-4">
              <PromptTesterTab />
            </TabsContent>

            <TabsContent value="rate-limits" className="space-y-4">
              <RateLimitsTab
                settings={localSettings.processor}
                onChange={(updates) => updateLocalSettings('processor', updates)}
              />
            </TabsContent>

            <TabsContent value="recycler" className="space-y-4">
              <RecyclerTab
                settings={localSettings.recycler}
                onChange={(updates) => updateLocalSettings('recycler', updates)}
              />
            </TabsContent>

            <TabsContent value="auto-steer" className="space-y-4">
              <AutoSteerTab />
            </TabsContent>
          </div>
        </Tabs>

        <DialogFooter className="flex items-center justify-between">
          <Button
            variant="outline"
            size="sm"
            onClick={handleReset}
            disabled={resetSettings.isPending}
          >
            <RotateCcw className="h-4 w-4 mr-2" />
            Reset to Defaults
          </Button>
          <div className="flex gap-2">
            <Button variant="outline" onClick={handleCancel}>
              Cancel
            </Button>
            <Button onClick={handleSave} disabled={saveSettings.isPending}>
              {saveSettings.isPending ? (
                'Saving...'
              ) : (
                <>
                  <Save className="h-4 w-4 mr-2" />
                  Save Changes
                </>
              )}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
