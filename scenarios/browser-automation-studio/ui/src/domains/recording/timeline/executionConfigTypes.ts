import type { ArtifactProfile } from '@/domains/executions/store';
import type { NavigationWaitUntil } from '@/types/workflow';

export interface ExecutionConfigSettings {
  navigationWaitUntil: NavigationWaitUntil;
  actionTimeoutSeconds: number;
  viewportWidth: number;
  viewportHeight: number;
  continueOnError: boolean;
  artifactProfile?: ArtifactProfile;
}
