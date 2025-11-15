import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import { Button } from "../ui/button";
import { Download } from "lucide-react";
import { platformIcons, platformNames } from "./utils";

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

interface DownloadButtonsProps {
  scenarioName: string;
  platforms?: string[];
}

export function DownloadButtons({ scenarioName, platforms = [] }: DownloadButtonsProps) {
  const handleDownload = (platform: string) => {
    const downloadUrl = buildUrl(`/desktop/download/${scenarioName}/${platform}`);
    window.open(downloadUrl, '_blank');
  };

  if (platforms.length === 0) {
    return null;
  }

  return (
    <div className="flex gap-2">
      {platforms.map((platform) => (
        <Button
          key={platform}
          variant="outline"
          size="sm"
          className="gap-2"
          onClick={() => handleDownload(platform)}
        >
          <Download className="h-3 w-3" />
          <span>{platformIcons[platform]}</span>
          {platformNames[platform]}
        </Button>
      ))}
    </div>
  );
}
