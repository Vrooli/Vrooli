export interface LifecycleControlConfig {
  running: boolean;
  loading: boolean;
  onStart: () => void;
  onStop?: () => void;
  onRestart?: () => void;
}
