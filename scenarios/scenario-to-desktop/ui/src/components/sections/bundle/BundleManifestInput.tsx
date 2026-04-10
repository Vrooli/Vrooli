/**
 * Manifest path input component for bundle configuration.
 * Allows users to specify or view the bundle manifest path.
 */

import { Input } from "../../ui/input";
import { Label } from "../../ui/label";

interface BundleManifestInputProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}

export function BundleManifestInput({ value, onChange, disabled }: BundleManifestInputProps) {
  return (
    <div className="space-y-2">
      <div>
        <Label htmlFor="bundleManifest">Bundle manifest path</Label>
        <p className="text-[11px] text-slate-400 mt-0.5">
          Path to the bundle.json file. Generated automatically when you run the bundle stage.
        </p>
      </div>
      <Input
        id="bundleManifest"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="/home/you/Vrooli/scenarios/my-scenario/platforms/electron/bundle/bundle.json"
        disabled={disabled}
      />
      {value.trim() && (
        <p className="text-[11px] text-slate-400">
          Expect this file to live alongside staged binaries/assets in the bundle directory.
        </p>
      )}
    </div>
  );
}
