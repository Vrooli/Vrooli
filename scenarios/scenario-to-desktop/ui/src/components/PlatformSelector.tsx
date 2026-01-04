import { Checkbox } from "./ui/checkbox";
import { Label } from "./ui/label";

type PlatformSelection = {
  win: boolean;
  mac: boolean;
  linux: boolean;
};

interface PlatformSelectorProps {
  platforms: PlatformSelection;
  onPlatformChange: (platform: string, checked: boolean) => void;
}

export function PlatformSelector({ platforms, onPlatformChange }: PlatformSelectorProps) {
  return (
    <div>
      <Label>Target Platforms</Label>
      <div className="mt-2 flex flex-wrap gap-4">
        <Checkbox
          id="platformWin"
          checked={platforms.win}
          onChange={(e) => onPlatformChange("win", e.target.checked)}
          label="Windows"
        />
        <Checkbox
          id="platformMac"
          checked={platforms.mac}
          onChange={(e) => onPlatformChange("mac", e.target.checked)}
          label="macOS"
        />
        <Checkbox
          id="platformLinux"
          checked={platforms.linux}
          onChange={(e) => onPlatformChange("linux", e.target.checked)}
          label="Linux"
        />
      </div>
    </div>
  );
}
