import { Download, RotateCcw, Upload } from "lucide-react";
import { Button } from "../../ui/primitives";
import { Notice, NoticeTitle } from "../../ui/composites";
import { CodePreview } from "../CodePreview";
import type { Config } from "../../../lib/api";

interface ImportExportSettingsProps {
  onExport: () => void;
  onImport: () => void;
  onReset: () => void;
  config: Config | null;
}

export function ImportExportSettings({ onExport, onImport, onReset, config }: ImportExportSettingsProps) {
  return (
    <div className="space-y-6">
      <div className="rounded-lg border border-border-default/70 bg-surface-overlay/40 p-4">
        <h3 className="mb-2 text-lg font-medium">Export Configuration</h3>
        <p className="mb-4 text-sm text-text-muted">
          Download your current configuration as a JSON file. You can use this to backup your settings or
          transfer them to another installation.
        </p>
        <Button onClick={onExport}>
          <Download className="mr-2" size={16} />
          Export Configuration
        </Button>
      </div>

      <div className="rounded-lg border border-border-default/70 bg-surface-overlay/40 p-4">
        <h3 className="mb-2 text-lg font-medium">Import Configuration</h3>
        <p className="mb-4 text-sm text-text-muted">
          Load a previously exported configuration file. This will replace your current settings.
        </p>
        <Button variant="outline" onClick={onImport}>
          <Upload className="mr-2" size={16} />
          Import Configuration
        </Button>
      </div>

      <Notice tone="warning" className="p-4">
        <NoticeTitle tone="warning" className="mb-2 text-lg">
          Reset to Defaults
        </NoticeTitle>
        <p className="mb-4 text-sm text-text-muted">
          Reset all settings to their default values. This will clear all your customizations.
        </p>
        <Button
          variant="outline"
          onClick={onReset}
          className="border-accent-warning/30 text-accent-warning hover:bg-accent-warning/20"
        >
          <RotateCcw className="mr-2" size={16} />
          Reset to Defaults
        </Button>
      </Notice>

      {config && (
        <div className="rounded-lg border border-border-default/70 bg-surface-overlay/40 p-4">
          <h3 className="mb-2 text-lg font-medium">Current Configuration</h3>
          <CodePreview code={config} language="json" maxHeight="12rem" />
        </div>
      )}
    </div>
  );
}
