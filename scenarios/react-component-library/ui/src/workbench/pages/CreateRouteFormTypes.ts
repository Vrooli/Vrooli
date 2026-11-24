export type TextSetter = (value: string) => void;
export interface MutationStatus {
  isError: boolean;
  error: unknown;
  isPending: boolean;
  mutate: () => void;
}
export interface AdoptionPreflight {
  blocking: boolean;
  styleFit: string;
  dependencyVerdict: string;
  warnings: string[];
  unsatisfiedTokens: string[];
}
export interface AdoptionAssetOption {
  id: string;
  displayName: string;
  latestVersion?: string;
  version?: string;
}
export interface ScenarioOption {
  scenario: string;
}
