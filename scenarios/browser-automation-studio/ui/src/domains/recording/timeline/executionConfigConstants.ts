import type { NavigationWaitUntil } from '@/types/workflow';
import type { ExecutionConfigSettings } from './ExecutionConfigPanel';

export const DEFAULT_EXECUTION_SETTINGS: ExecutionConfigSettings = {
  navigationWaitUntil: 'domcontentloaded',
  actionTimeoutSeconds: 30,
  viewportWidth: 1920,
  viewportHeight: 1080,
  continueOnError: false,
  artifactProfile: 'standard',
};

export const NAVIGATION_WAIT_OPTIONS: {
  value: NavigationWaitUntil;
  label: string;
  description: string;
}[] = [
  { value: 'domcontentloaded', label: 'DOM Ready', description: 'Wait for HTML to parse (fast, recommended)' },
  { value: 'load', label: 'Page Load', description: 'Wait for all resources to load' },
  { value: 'networkidle', label: 'Network Idle', description: 'Wait for no network activity (slow on heavy sites)' },
];
