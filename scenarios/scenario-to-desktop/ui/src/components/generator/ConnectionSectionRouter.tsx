/**
 * Connection section router for GeneratorForm.
 * Switches between External (Remote) and Embedded server sections.
 * Note: BundleSection renders the bundled workflow directly.
 */

import type { ProbeResponse, ProxyHintsResponse } from "../../lib/api";
import type { ConnectionDecision } from "../../domain/deployment";
import { ExternalServerSection, EmbeddedServerSection } from "../runtime";

export interface ConnectionSectionRouterProps {
  connectionDecision: ConnectionDecision;
  scenarioName: string;
  // External server props
  proxyUrl: string;
  onProxyUrlChange: (url: string) => void;
  proxyHints: ProxyHintsResponse | null | undefined;
  connectionTester: {
    isPending: boolean;
    mutate: () => void;
  };
  connectionResult: ProbeResponse | null;
  connectionError: string | null;
  autoManageTier1: boolean;
  onAutoManageTier1Change: (value: boolean) => void;
  vrooliBinaryPath: string;
  onVrooliBinaryPathChange: (path: string) => void;
  // Embedded server props
  serverPort: number;
  onServerPortChange: (port: number) => void;
  localServerPath: string;
  onLocalServerPathChange: (path: string) => void;
  localApiEndpoint: string;
  onLocalApiEndpointChange: (endpoint: string) => void;
}

export function ConnectionSectionRouter({
  connectionDecision,
  proxyUrl,
  onProxyUrlChange,
  scenarioName,
  proxyHints,
  connectionTester,
  connectionResult,
  connectionError,
  autoManageTier1,
  onAutoManageTier1Change,
  vrooliBinaryPath,
  onVrooliBinaryPathChange,
  serverPort,
  onServerPortChange,
  localServerPath,
  onLocalServerPathChange,
  localApiEndpoint,
  onLocalApiEndpointChange,
}: ConnectionSectionRouterProps) {
  // BundleSection owns bundled mode, so this router has no bundled branch.
  if (connectionDecision.kind === "bundled-runtime") {
    return null;
  }

  if (connectionDecision.kind === "remote-server") {
    return (
      <ExternalServerSection
        proxyUrl={proxyUrl}
        onProxyUrlChange={onProxyUrlChange}
        scenarioName={scenarioName}
        proxyHints={proxyHints}
        connectionTester={connectionTester}
        connectionResult={connectionResult}
        connectionError={connectionError}
        autoManageTier1={autoManageTier1}
        onAutoManageTier1Change={onAutoManageTier1Change}
        vrooliBinaryPath={vrooliBinaryPath}
        onVrooliBinaryPathChange={onVrooliBinaryPathChange}
      />
    );
  }

  return (
    <EmbeddedServerSection
      serverPort={serverPort}
      onServerPortChange={onServerPortChange}
      localServerPath={localServerPath}
      onLocalServerPathChange={onLocalServerPathChange}
      localApiEndpoint={localApiEndpoint}
      onLocalApiEndpointChange={onLocalApiEndpointChange}
    />
  );
}
