import { Card, CardContent } from "../ui/card";
import { Badge } from "../ui/badge";
import { Monitor, Package, CheckCircle, XCircle, Clock } from "lucide-react";
import { formatBytes } from "./utils";
import { GenerateDesktopButton } from "./GenerateDesktopButton";
import { BuildDesktopButton } from "./BuildDesktopButton";
import { RegenerateButton } from "./RegenerateButton";
import { DeleteButton } from "./DeleteButton";
import type { ScenarioDesktopStatus } from "./types";

interface ScenarioCardProps {
  scenario: ScenarioDesktopStatus;
}

export function ScenarioCard({ scenario }: ScenarioCardProps) {
  return (
    <Card
      className={`border ${
        scenario.has_desktop
          ? 'border-green-700 bg-green-950/10'
          : 'border-slate-700 bg-slate-800/50'
      } transition-all hover:border-blue-600`}
    >
      <CardContent className="p-4">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-slate-700">
                {scenario.has_desktop ? (
                  <Monitor className="h-5 w-5 text-green-400" />
                ) : (
                  <Package className="h-5 w-5 text-slate-400" />
                )}
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <h4 className="font-semibold">{scenario.name}</h4>

                  {/* Status Badge - Clear and Prominent */}
                  {!scenario.has_desktop && (
                    <Badge variant="secondary" className="text-xs bg-slate-700 text-slate-300">
                      <XCircle className="mr-1 h-3 w-3" />
                      Not Generated
                    </Badge>
                  )}

                  {scenario.has_desktop && !scenario.built && (
                    <Badge variant="warning" className="text-xs bg-yellow-900/30 text-yellow-400 border-yellow-700">
                      <Clock className="mr-1 h-3 w-3" />
                      Generated
                    </Badge>
                  )}

                  {scenario.has_desktop && scenario.built && (
                    <Badge variant="success" className="text-xs">
                      <CheckCircle className="mr-1 h-3 w-3" />
                      Built: {scenario.platforms?.join(' ') || 'Ready'}
                    </Badge>
                  )}
                </div>
                {scenario.display_name && (
                  <p className="text-sm text-slate-400">{scenario.display_name}</p>
                )}
              </div>
            </div>

            {scenario.has_desktop && (
              <div className="mt-3 ml-13 grid grid-cols-2 gap-4 text-sm md:grid-cols-4">
                {scenario.version && (
                  <div>
                    <span className="text-slate-500">Version:</span>{" "}
                    <span className="text-slate-300">{scenario.version}</span>
                  </div>
                )}
                {scenario.platforms && scenario.platforms.length > 0 && (
                  <div>
                    <span className="text-slate-500">Platforms:</span>{" "}
                    <span className="text-slate-300">{scenario.platforms.join(", ")}</span>
                  </div>
                )}
                {scenario.package_size && scenario.package_size > 0 && (
                  <div>
                    <span className="text-slate-500">Size:</span>{" "}
                    <span className="text-slate-300">{formatBytes(scenario.package_size)}</span>
                  </div>
                )}
                {scenario.last_modified && (
                  <div>
                    <span className="text-slate-500">Modified:</span>{" "}
                    <span className="text-slate-300">{scenario.last_modified}</span>
                  </div>
                )}
              </div>
            )}

            {scenario.desktop_path && (
              <div className="mt-2 ml-13 text-xs text-slate-500">
                {scenario.desktop_path}
              </div>
            )}

            {/* Action buttons based on state */}
            <div className="mt-3 ml-13 flex flex-col gap-3">
              <GenerateDesktopButton scenario={scenario} />
              {scenario.has_desktop && (
                <>
                  <BuildDesktopButton scenarioName={scenario.name} />
                  <div className="flex flex-wrap gap-2">
                    {scenario.built && <RegenerateButton scenarioName={scenario.name} connectionConfig={scenario.connection_config} />}
                    <DeleteButton scenarioName={scenario.name} />
                  </div>
                </>
              )}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
