import {
  DeploymentMode,
  Platform,
  StageName,
  TemplateType,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

/** Converts user-selected platform labels into the generated contract enum. */
export function platformFromFormValue(value: string): Platform {
  switch (value) {
    case "win":
    case "windows":
      return Platform.WIN;
    case "mac":
    case "macos":
    case "darwin":
      return Platform.MAC;
    case "linux":
      return Platform.LINUX;
    default:
      return Platform.UNSPECIFIED;
  }
}

export function deploymentModeFromFormValue(value: string): DeploymentMode {
  switch (value) {
    case "proxy":
    case "external-server":
      return DeploymentMode.PROXY;
    default:
      return DeploymentMode.BUNDLED;
  }
}

export function templateTypeFromFormValue(value: string): TemplateType {
  switch (value) {
    case "advanced":
      return TemplateType.ADVANCED;
    case "multi-window":
      return TemplateType.MULTI_WINDOW;
    case "kiosk":
      return TemplateType.KIOSK;
    default:
      return TemplateType.BASIC;
  }
}

export const PIPELINE_STAGE_BY_FORM_ID = {
  bundle: StageName.BUNDLE,
  preflight: StageName.PREFLIGHT,
  generate: StageName.GENERATE,
  build: StageName.BUILD,
  smoketest: StageName.SMOKE_TEST,
  deploy: StageName.DEPLOY,
} as const;
