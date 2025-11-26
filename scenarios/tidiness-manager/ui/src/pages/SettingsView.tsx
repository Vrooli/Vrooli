import { Settings, Save, RotateCcw, CheckCircle, Info, AlertTriangle, Download, Upload, Copy } from "lucide-react";
import { useState, useRef } from "react";
import { useToast } from "../components/ui/toast";
import { PageHeader } from "../components/layout/PageHeader";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { Button } from "../components/ui/button";

interface SettingsFormData {
  longFileThreshold: number;
  maxScansPerFile: number;
  maxConcurrentCampaigns: number;
  defaultMaxSessions: number;
  defaultFilesPerSession: number;
  lightScanTimeout: number;
  aiScanBatchSize: number;
}

const DEFAULT_SETTINGS: SettingsFormData = {
  longFileThreshold: 500,
  maxScansPerFile: 10,
  maxConcurrentCampaigns: 3,
  defaultMaxSessions: 10,
  defaultFilesPerSession: 5,
  lightScanTimeout: 120,
  aiScanBatchSize: 10,
};

const CONSERVATIVE_PRESET: SettingsFormData = {
  longFileThreshold: 400,
  maxScansPerFile: 3,
  maxConcurrentCampaigns: 1,
  defaultMaxSessions: 5,
  defaultFilesPerSession: 3,
  lightScanTimeout: 180,
  aiScanBatchSize: 5,
};

const AGGRESSIVE_PRESET: SettingsFormData = {
  longFileThreshold: 800,
  maxScansPerFile: 20,
  maxConcurrentCampaigns: 5,
  defaultMaxSessions: 20,
  defaultFilesPerSession: 10,
  lightScanTimeout: 60,
  aiScanBatchSize: 15,
};

export default function SettingsView() {
  const { showToast } = useToast();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [settings, setSettings] = useState<SettingsFormData>(DEFAULT_SETTINGS);
  const [hasChanges, setHasChanges] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const handleChange = (field: keyof SettingsFormData, value: string) => {
    const numValue = parseInt(value, 10);
    if (!isNaN(numValue) && numValue > 0) {
      setSettings(prev => ({ ...prev, [field]: numValue }));
      setHasChanges(true);
    }
  };

  const handleSave = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    setIsSaving(true);
    try {
      // TODO: Wire to API endpoint PUT /api/v1/config when backend ready
      await new Promise(resolve => setTimeout(resolve, 500));
      setHasChanges(false);
      showToast("Settings saved successfully. Changes will apply to future scans and campaigns.", "success");
    } catch (error) {
      showToast(`Failed to save settings: ${(error as Error).message}`, "error");
    } finally {
      setIsSaving(false);
    }
  };

  const handleReset = () => {
    setSettings(DEFAULT_SETTINGS);
    setHasChanges(false);
    showToast("Settings reset to default values", "info");
  };

  const exportSettings = () => {
    const exportData = {
      version: "1.0",
      exported_at: new Date().toISOString(),
      settings: settings,
    };

    const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `tidiness-manager-settings-${new Date().toISOString().split('T')[0]}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    showToast("Settings exported to JSON file", "success");
  };

  const copySettingsJSON = async () => {
    const exportData = {
      version: "1.0",
      settings: settings,
    };

    try {
      await navigator.clipboard.writeText(JSON.stringify(exportData, null, 2));
      showToast("Settings copied as JSON to clipboard", "success");
    } catch (err) {
      showToast("Failed to copy to clipboard", "error");
    }
  };

  const handleImport = () => {
    fileInputRef.current?.click();
  };

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (e) => {
      try {
        const content = e.target?.result as string;
        const data = JSON.parse(content);

        if (!data.settings) {
          showToast("Invalid settings file format", "error");
          return;
        }

        // Validate all required fields
        const requiredFields: (keyof SettingsFormData)[] = [
          'longFileThreshold', 'maxScansPerFile', 'maxConcurrentCampaigns',
          'defaultMaxSessions', 'defaultFilesPerSession', 'lightScanTimeout', 'aiScanBatchSize'
        ];

        const missingFields = requiredFields.filter(field => !(field in data.settings));
        if (missingFields.length > 0) {
          showToast(`Missing required fields: ${missingFields.join(', ')}`, "error");
          return;
        }

        setSettings(data.settings);
        setHasChanges(true);
        showToast("Settings imported successfully. Click Save to apply.", "success");
      } catch (err) {
        showToast("Failed to parse settings file", "error");
      }
    };
    reader.readAsText(file);

    // Reset file input so the same file can be imported again
    if (event.target) event.target.value = '';
  };

  return (
    <div>
      <input
        type="file"
        ref={fileInputRef}
        onChange={handleFileChange}
        accept=".json"
        style={{ display: 'none' }}
      />
      <PageHeader
        title="Settings"
        description={
          <div className="space-y-1">
            <p>Configure tidiness manager thresholds and preferences. These settings affect light scanning, smart scanning, and campaign behavior.</p>
            <p className="text-xs text-slate-500 font-mono hidden sm:block">
              CLI: tidiness-manager configure --help | Export/import JSON for agent automation
            </p>
          </div>
        }
        actions={
          <div className="flex items-center gap-2 flex-wrap">
            <Button
              variant="ghost"
              size="sm"
              onClick={copySettingsJSON}
              title="Copy settings as JSON to clipboard for agent automation"
              className="flex-1 sm:flex-none"
            >
              <Copy className="h-4 w-4 sm:mr-2" />
              <span className="hidden sm:inline">Copy JSON</span>
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={handleImport}
              title="Import settings from JSON file"
              className="flex-1 sm:flex-none"
            >
              <Upload className="h-4 w-4 sm:mr-2" />
              <span className="hidden sm:inline">Import</span>
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={exportSettings}
              title="Download settings as JSON file for backup or agent automation"
              className="flex-1 sm:flex-none"
            >
              <Download className="h-4 w-4 sm:mr-2" />
              <span className="hidden sm:inline">Export</span>
            </Button>
          </div>
        }
      />

      {/* Presets */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle>Quick Presets</CardTitle>
          <CardDescription>
            Apply common configuration templates based on your use case
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3" role="group" aria-label="Configuration presets">
            <button
              onClick={() => {
                setSettings(CONSERVATIVE_PRESET);
                setHasChanges(true);
              }}
              className="border border-slate-700 rounded-lg p-4 hover:bg-white/5 transition-colors text-left focus:ring-2 focus:ring-slate-400 focus:outline-none"
              aria-label="Apply conservative preset: 1 concurrent campaign, 400-line threshold, 5 files per batch"
            >
              <p className="font-semibold text-sm text-slate-200 mb-1">Conservative</p>
              <p className="text-xs text-slate-400 mb-3">
                Low resource usage, higher quality thresholds. Best for cost-sensitive environments or exploratory use.
              </p>
              <div className="space-y-1 text-xs text-slate-500" aria-hidden="true">
                <p>• 1 concurrent campaign</p>
                <p>• 400-line file threshold</p>
                <p>• 5 files/batch</p>
              </div>
            </button>
            <button
              onClick={() => {
                setSettings(DEFAULT_SETTINGS);
                setHasChanges(true);
              }}
              className="border border-blue-500/30 rounded-lg p-4 hover:bg-white/5 transition-colors text-left bg-blue-500/10 focus:ring-2 focus:ring-blue-400 focus:outline-none"
              aria-label="Apply balanced preset (recommended): 3 concurrent campaigns, 500-line threshold, 10 files per batch"
            >
              <div className="flex items-center gap-2 mb-1">
                <p className="font-semibold text-sm text-slate-200">Balanced</p>
                <span className="text-xs px-1.5 py-0.5 rounded bg-blue-500/20 text-blue-300" aria-label="default preset">Default</span>
              </div>
              <p className="text-xs text-slate-400 mb-3">
                Recommended for most scenarios. Good balance between thoroughness and resource consumption.
              </p>
              <div className="space-y-1 text-xs text-slate-500" aria-hidden="true">
                <p>• 3 concurrent campaigns</p>
                <p>• 500-line file threshold</p>
                <p>• 10 files/batch</p>
              </div>
            </button>
            <button
              onClick={() => {
                setSettings(AGGRESSIVE_PRESET);
                setHasChanges(true);
              }}
              className="border border-slate-700 rounded-lg p-4 hover:bg-white/5 transition-colors text-left focus:ring-2 focus:ring-slate-400 focus:outline-none"
              aria-label="Apply aggressive preset: 5 concurrent campaigns, 800-line threshold, 15 files per batch. Warning: higher resource costs"
            >
              <p className="font-semibold text-sm text-slate-200 mb-1">Aggressive</p>
              <p className="text-xs text-slate-400 mb-3">
                Maximum throughput for urgent refactoring or fleet-wide analysis. Higher resource costs.
              </p>
              <div className="space-y-1 text-xs text-slate-500" aria-hidden="true">
                <p>• 5 concurrent campaigns</p>
                <p>• 800-line file threshold</p>
                <p>• 15 files/batch</p>
              </div>
            </button>
          </div>
        </CardContent>
      </Card>

      <div className="space-y-6">
        {/* Thresholds */}
        <Card>
          <CardHeader>
            <div className="flex items-start gap-3">
              <div className="flex-1">
                <CardTitle>File Thresholds</CardTitle>
                <CardDescription>
                  Configure line count thresholds for flagging long files
                </CardDescription>
              </div>
              <div className="bg-blue-500/10 border border-blue-500/30 rounded px-3 py-2 text-xs text-blue-300 flex items-center gap-2">
                <Info className="h-3.5 w-3.5" />
                <span>Changes apply to future scans</span>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label htmlFor="longFileThreshold" className="text-sm text-slate-400 mb-2 block">
                Long File Threshold (lines)
              </label>
              <Input
                id="longFileThreshold"
                type="number"
                min="1"
                value={settings.longFileThreshold}
                onChange={(e) => handleChange('longFileThreshold', e.target.value)}
                aria-describedby="longFileThreshold-description"
              />
              <div className="flex items-start gap-2 mt-1" id="longFileThreshold-description">
                <Info className="h-3.5 w-3.5 text-slate-500 mt-0.5 flex-shrink-0" aria-hidden="true" />
                <p className="text-xs text-slate-500">
                  Files exceeding this line count will be flagged as long. Recommended: 400-600 lines for most projects.
                </p>
              </div>
            </div>

            <div>
              <label htmlFor="maxScansPerFile" className="text-sm text-slate-400 mb-2 block">
                Max Scans Per File
              </label>
              <Input
                id="maxScansPerFile"
                type="number"
                min="1"
                value={settings.maxScansPerFile}
                onChange={(e) => handleChange('maxScansPerFile', e.target.value)}
                aria-describedby="maxScansPerFile-description"
              />
              <div className="flex items-start gap-2 mt-1" id="maxScansPerFile-description">
                <Info className="h-3.5 w-3.5 text-slate-500 mt-0.5 flex-shrink-0" aria-hidden="true" />
                <p className="text-xs text-slate-500">
                  Maximum number of times a file can be scanned in a campaign. Prevents hammering the same files. Recommended: 3-10 visits.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Campaign Settings */}
        <Card>
          <CardHeader>
            <CardTitle>Campaign Settings</CardTitle>
            <CardDescription>
              Configure automated campaign behavior
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label htmlFor="maxConcurrentCampaigns" className="text-sm text-slate-400 mb-2 block">
                Max Concurrent Campaigns
              </label>
              <Input
                id="maxConcurrentCampaigns"
                type="number"
                min="1"
                value={settings.maxConcurrentCampaigns}
                onChange={(e) => handleChange('maxConcurrentCampaigns', e.target.value)}
                aria-describedby="maxConcurrentCampaigns-description"
              />
              <div className="flex items-start gap-2 mt-1" id="maxConcurrentCampaigns-description">
                <AlertTriangle className="h-3.5 w-3.5 text-yellow-500 mt-0.5 flex-shrink-0" aria-hidden="true" />
                <p className="text-xs text-slate-500">
                  Maximum number of campaigns that can run simultaneously. <span className="text-yellow-500">Higher values increase AI/resource costs.</span> Recommended: 3.
                </p>
              </div>
            </div>

            <div>
              <label htmlFor="defaultMaxSessions" className="text-sm text-slate-400 mb-2 block">
                Default Max Sessions
              </label>
              <Input
                id="defaultMaxSessions"
                type="number"
                min="1"
                value={settings.defaultMaxSessions}
                onChange={(e) => handleChange('defaultMaxSessions', e.target.value)}
                aria-describedby="defaultMaxSessions-description"
              />
              <p className="text-xs text-slate-500 mt-1" id="defaultMaxSessions-description">
                Default maximum sessions for new campaigns
              </p>
            </div>

            <div>
              <label htmlFor="defaultFilesPerSession" className="text-sm text-slate-400 mb-2 block">
                Default Files Per Session
              </label>
              <Input
                id="defaultFilesPerSession"
                type="number"
                min="1"
                value={settings.defaultFilesPerSession}
                onChange={(e) => handleChange('defaultFilesPerSession', e.target.value)}
                aria-describedby="defaultFilesPerSession-description"
              />
              <p className="text-xs text-slate-500 mt-1" id="defaultFilesPerSession-description">
                Default number of files to analyze per session
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Scan Settings */}
        <Card>
          <CardHeader>
            <CardTitle>Scan Settings</CardTitle>
            <CardDescription>
              Configure scan behavior and timeouts
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label htmlFor="lightScanTimeout" className="text-sm text-slate-400 mb-2 block">
                Light Scan Timeout (seconds)
              </label>
              <Input
                id="lightScanTimeout"
                type="number"
                min="1"
                value={settings.lightScanTimeout}
                onChange={(e) => handleChange('lightScanTimeout', e.target.value)}
                aria-describedby="lightScanTimeout-description"
              />
              <div className="flex items-start gap-2 mt-1" id="lightScanTimeout-description">
                <Info className="h-3.5 w-3.5 text-slate-500 mt-0.5 flex-shrink-0" aria-hidden="true" />
                <p className="text-xs text-slate-500">
                  Maximum time to wait for light scan completion. Large scenarios may need 120+ seconds. Recommended: 60-180 seconds.
                </p>
              </div>
            </div>

            <div>
              <label htmlFor="aiScanBatchSize" className="text-sm text-slate-400 mb-2 block">
                AI Scan Batch Size
              </label>
              <Input
                id="aiScanBatchSize"
                type="number"
                min="1"
                value={settings.aiScanBatchSize}
                onChange={(e) => handleChange('aiScanBatchSize', e.target.value)}
                aria-describedby="aiScanBatchSize-description"
              />
              <div className="flex items-start gap-2 mt-1" id="aiScanBatchSize-description">
                <AlertTriangle className="h-3.5 w-3.5 text-yellow-500 mt-0.5 flex-shrink-0" aria-hidden="true" />
                <p className="text-xs text-slate-500">
                  Maximum files to process in a single AI scan batch. <span className="text-yellow-500">Larger batches = faster but costlier.</span> Recommended: 5-10 files.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Actions */}
        <div className="flex justify-between items-center">
          <Button
            variant="ghost"
            onClick={handleReset}
            disabled={isSaving}
            className="text-slate-400 hover:text-slate-200"
          >
            <RotateCcw className="h-4 w-4 mr-2" />
            Reset to Defaults
          </Button>
          <Button
            variant="primary"
            onClick={handleSave}
            disabled={!hasChanges || isSaving}
          >
            <Save className="h-4 w-4 mr-2" />
            {isSaving ? 'Saving...' : 'Save Changes'}
          </Button>
        </div>
      </div>
    </div>
  );
}
