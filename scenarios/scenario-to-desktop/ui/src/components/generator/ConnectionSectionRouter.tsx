/**
 * Connection section router for GeneratorForm.
 * Switches between Bundled, External (Remote), and Embedded server sections.
 */

import type { Ref } from "react";
import type { PipelineConfig, ProbeResponse, ProxyHintsResponse } from "../../lib/api";
import type { ConnectionDecision } from "../../domain/deployment";
import type { DeploymentManagerBundleHelperHandle, BundleResult } from "../runtime/DeploymentManagerBundleHelper";
import { BundledRuntimeSection, ExternalServerSection, EmbeddedServerSection } from "../runtime";

export interface ConnectionSectionRouterProps {
  connectionDecision: ConnectionDecision;
  // Bundled runtime props
  bundleManifestPath: string;
  onBundleManifestChange: (path: string) => void;
  scenarioName: string;
  bundleHelperRef: Ref<DeploymentManagerBundleHelperHandle>;
  onBundleExported: (manifestPath: string, config?: Partial<PipelineConfig>) => void;
  onBundleComplete: (result: BundleResult) => void;
  initialBundleResult: BundleResult | null;
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
  bundleManifestPath,
  onBundleManifestChange,
  scenarioName,
  bundleHelperRef,
  onBundleExported,
  onBundleComplete,
  initialBundleResult,
  proxyUrl,
  onProxyUrlChange,
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
  if (connectionDecision.kind === "bundled-runtime") {
    return (
      <BundledRuntimeSection
        bundleManifestPath={bundleManifestPath}
        onBundleManifestChange={onBundleManifestChange}
        scenarioName={scenarioName}
        bundleHelperRef={bundleHelperRef}
        onBundleExported={onBundleExported}
        onBundleComplete={onBundleComplete}
        initialBundleResult={initialBundleResult}
      />
    );
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
