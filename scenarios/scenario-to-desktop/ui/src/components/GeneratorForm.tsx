import { useState, type FormEvent } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import { generateDesktop, type DesktopConfig } from "../lib/api";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { Select } from "./ui/select";
import { Checkbox } from "./ui/checkbox";
import { Button } from "./ui/button";
import { Rocket, Search } from "lucide-react";
import type { ScenariosResponse } from "./scenario-inventory/types";

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

interface GeneratorFormProps {
  selectedTemplate: string;
  onTemplateChange: (template: string) => void;
  onBuildStart: (buildId: string) => void;
}

export function GeneratorForm({ selectedTemplate, onTemplateChange, onBuildStart }: GeneratorFormProps) {
  const [scenarioName, setScenarioName] = useState("");
  const [useDropdown, setUseDropdown] = useState(true);
  const [framework, setFramework] = useState("electron");
  const [platforms, setPlatforms] = useState({
    win: true,
    mac: true,
    linux: true
  });
  const [outputPath, setOutputPath] = useState("./desktop-app");

  // Fetch available scenarios
  const { data: scenariosData, isLoading: loadingScenarios } = useQuery<ScenariosResponse>({
    queryKey: ['scenarios-desktop-status'],
    queryFn: async () => {
      const res = await fetch(buildUrl('/scenarios/desktop-status'));
      if (!res.ok) throw new Error('Failed to fetch scenarios');
      return res.json();
    },
  });

  const generateMutation = useMutation({
    mutationFn: generateDesktop,
    onSuccess: (data) => {
      onBuildStart(data.build_id);
    }
  });

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();

    const selectedPlatforms = Object.entries(platforms)
      .filter(([_, enabled]) => enabled)
      .map(([platform]) => platform);

    if (selectedPlatforms.length === 0) {
      alert("Please select at least one target platform");
      return;
    }

    const config: DesktopConfig = {
      app_name: scenarioName,
      app_display_name: `${scenarioName} Desktop`,
      app_description: `Desktop application for ${scenarioName} scenario`,
      version: "1.0.0",
      author: "Vrooli Platform",
      license: "MIT",
      app_id: `com.vrooli.${scenarioName.replace(/-/g, ".")}`,
      server_type: "node",
      server_port: 3000,
      server_path: "ui/server.js",
      api_endpoint: "http://localhost:3000",
      framework,
      template_type: selectedTemplate,
      platforms: selectedPlatforms,
      output_path: outputPath,
      features: {
        splash: true,
        autoUpdater: true,
        devTools: true
      },
      window: {
        width: 1200,
        height: 800,
        background: "#f5f5f5"
      }
    };

    generateMutation.mutate(config);
  };

  const handlePlatformChange = (platform: string, checked: boolean) => {
    setPlatforms((prev) => ({ ...prev, [platform]: checked }));
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Rocket className="h-5 w-5" />
          Generate Desktop App
        </CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <Label htmlFor="scenarioName">Scenario Name</Label>
              <button
                type="button"
                onClick={() => setUseDropdown(!useDropdown)}
                className="text-xs text-blue-400 hover:text-blue-300"
              >
                {useDropdown ? "Enter manually" : "Select from list"}
              </button>
            </div>

            {useDropdown ? (
              <Select
                id="scenarioName"
                value={scenarioName}
                onChange={(e) => setScenarioName(e.target.value)}
                required
                className="mt-1.5"
                disabled={loadingScenarios}
              >
                <option value="">
                  {loadingScenarios ? "Loading scenarios..." : "Select a scenario..."}
                </option>
                {scenariosData?.scenarios
                  .filter(s => !s.has_desktop) // Only show scenarios without desktop
                  .sort((a, b) => a.name.localeCompare(b.name))
                  .map(scenario => (
                    <option key={scenario.name} value={scenario.name}>
                      {scenario.name} {scenario.display_name ? `(${scenario.display_name})` : ""}
                    </option>
                  ))}
              </Select>
            ) : (
              <Input
                id="scenarioName"
                value={scenarioName}
                onChange={(e) => setScenarioName(e.target.value)}
                placeholder="e.g., picker-wheel"
                required
                className="mt-1.5"
              />
            )}
            <p className="mt-1.5 text-xs text-slate-400">
              {useDropdown
                ? "Select from available scenarios that don't have desktop versions yet"
                : "Enter scenario name manually (must exist in scenarios directory)"}
            </p>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <Label htmlFor="framework">Framework</Label>
              <Select
                id="framework"
                value={framework}
                onChange={(e) => setFramework(e.target.value)}
                className="mt-1.5"
              >
                <option value="electron">Electron</option>
                <option value="tauri">Tauri</option>
                <option value="neutralino">Neutralino</option>
              </Select>
            </div>

            <div>
              <Label htmlFor="template">Template</Label>
              <Select
                id="template"
                value={selectedTemplate}
                onChange={(e) => onTemplateChange(e.target.value)}
                className="mt-1.5"
              >
                <option value="basic">Basic</option>
                <option value="advanced">Advanced</option>
                <option value="multi_window">Multi-Window</option>
                <option value="kiosk">Kiosk Mode</option>
              </Select>
            </div>
          </div>

          <div>
            <Label>Target Platforms</Label>
            <div className="mt-2 flex flex-wrap gap-4">
              <Checkbox
                id="platformWin"
                checked={platforms.win}
                onChange={(e) => handlePlatformChange("win", e.target.checked)}
                label="Windows"
              />
              <Checkbox
                id="platformMac"
                checked={platforms.mac}
                onChange={(e) => handlePlatformChange("mac", e.target.checked)}
                label="macOS"
              />
              <Checkbox
                id="platformLinux"
                checked={platforms.linux}
                onChange={(e) => handlePlatformChange("linux", e.target.checked)}
                label="Linux"
              />
            </div>
          </div>

          <div>
            <Label htmlFor="outputPath">Output Directory</Label>
            <Input
              id="outputPath"
              value={outputPath}
              onChange={(e) => setOutputPath(e.target.value)}
              placeholder="./desktop-app"
              className="mt-1.5"
            />
          </div>

          <Button
            type="submit"
            className="w-full"
            disabled={generateMutation.isPending}
          >
            {generateMutation.isPending ? "Generating..." : "Generate Desktop Application"}
          </Button>

          {generateMutation.isError && (
            <div className="rounded-lg bg-red-900/20 p-3 text-sm text-red-300">
              <strong>Error:</strong> {(generateMutation.error as Error).message}
            </div>
          )}
        </form>
      </CardContent>
    </Card>
  );
}
