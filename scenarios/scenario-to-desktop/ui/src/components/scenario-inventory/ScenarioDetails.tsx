import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { GenerateDesktopButton } from "./GenerateDesktopButton";
import { BuildDesktopButton } from "./BuildDesktopButton";
import { DownloadButtons } from "./DownloadButtons";
import { TelemetryUploadCard } from "./TelemetryUploadCard";
import { StepCard } from "./StepCard";
import { DeleteButton } from "./DeleteButton";
import type { ScenarioDesktopStatus } from "./types";
import { Monitor, X, BookOpenCheck } from "lucide-react";

interface ScenarioDetailsProps {
  scenario: ScenarioDesktopStatus;
  onClose: () => void;
}

export function ScenarioDetails({ scenario, onClose }: ScenarioDetailsProps) {
  const artifacts = scenario.build_artifacts || [];
  const hasArtifacts = artifacts.length > 0;

  return (
    <Card className="border-blue-700/40 bg-slate-900/60 shadow-xl">
      <CardHeader className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div className="flex items-center gap-3">
          <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-blue-950/60">
            <Monitor className="h-6 w-6 text-blue-300" />
          </div>
          <div>
            <CardTitle className="text-2xl">
              {scenario.display_name || scenario.name}
            </CardTitle>
            <p className="text-sm text-slate-400">
              Follow the three guided steps to turn this scenario into a shareable desktop installer.
            </p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <Badge variant={scenario.has_desktop ? 'success' : 'secondary'}>
            {scenario.has_desktop ? 'Desktop wrapper ready' : 'Needs desktop wrapper'}
          </Badge>
          <Button variant="ghost" size="sm" onClick={onClose} className="gap-2">
            <X className="h-4 w-4" />
            Back to list
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="rounded-lg border border-slate-800 bg-slate-900/40 p-4 text-sm text-slate-300">
          <p className="font-semibold text-slate-100">How it works</p>
          <p className="mt-1">
            Step 1 links your Electron wrapper to the scenario running on Tier&nbsp;1. Step 2 builds native installers for
            Windows, macOS, and Linux without leaving the browser. Step 3 lets you grab the installers and (optionally)
            upload telemetry so deployment-manager learns what failed. No CLI knowledge required.
          </p>
          <div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-blue-200">
            <BookOpenCheck className="h-3.5 w-3.5" />
            Need more background? Read the Desktop tier guide from the Deployment Hub.
            <a
              href="https://github.com/vrooli/vrooli/blob/main/docs/deployment/tiers/tier-2-desktop.md"
              target="_blank"
              rel="noreferrer"
              className="text-blue-300 underline"
            >
              Open documentation
            </a>
          </div>
        </div>

        <div className="space-y-5">
          <StepCard
            step={1}
            title="Connect to your Tier 1 scenario"
            description="Tell the desktop wrapper which running scenario to open. Paste the Cloudflare or LAN URL—it is saved for everyone testing this app."
            complete={scenario.has_desktop}
          >
            <GenerateDesktopButton scenario={scenario} />
          </StepCard>

          <StepCard
            step={2}
            title="Build installers"
            description="Choose the operating systems you want and let scenario-to-desktop run npm install/build/dist for you."
            complete={hasArtifacts}
            disabled={!scenario.has_desktop}
          >
            {scenario.has_desktop ? (
              <BuildDesktopButton scenarioName={scenario.name} />
            ) : (
              <p className="text-sm text-slate-400">
                Finish step 1 so we know which scenario to build from.
              </p>
            )}
          </StepCard>

          <StepCard
            step={3}
            title="Download + share telemetry"
            description="Hand installers to teammates, then upload the optional telemetry log so we can track failing dependencies."
            complete={hasArtifacts}
            disabled={!hasArtifacts}
          >
            {hasArtifacts ? (
              <div className="space-y-4">
                <DownloadButtons scenarioName={scenario.name} artifacts={artifacts} />
                <TelemetryUploadCard scenarioName={scenario.name} appDisplayName={scenario.display_name || scenario.name} />
              </div>
            ) : (
              <p className="text-sm text-slate-400">Build at least one installer to unlock downloads and telemetry uploads.</p>
            )}
          </StepCard>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-3 text-xs text-slate-400">
          <div>
            <p>Wrapper path: {scenario.desktop_path || 'Not generated yet'}</p>
            {scenario.version && <p>Current version: {scenario.version}</p>}
          </div>
          <DeleteButton scenarioName={scenario.name} />
        </div>
      </CardContent>
    </Card>
  );
}

