export type StoryEnvironment = {
  browser?: boolean;
  wait?: (milliseconds: number) => Promise<void>;
  flush?: () => Promise<void>;
  actInteraction?: (dispatch: () => void) => Promise<void>;
  skipKinds?: ReadonlySet<string>;
  queries?: Record<string, unknown>;
  report?: (passed: boolean, failures: unknown[], skipped: unknown[]) => void;
};

export type StoryFailure = {
  kind?: string;
  message?: string;
  expectation?: unknown;
  selector?: string;
  [key: string]: unknown;
};

export declare const browserEnv: StoryEnvironment;
export declare const jsdomEnv: StoryEnvironment;
export declare function runStory(
  previewStory: Record<string, unknown>,
  modules: { document: Document; window?: Window },
  env?: StoryEnvironment,
): Promise<{ passed: boolean; failures: StoryFailure[]; skipped: unknown[] }>;
