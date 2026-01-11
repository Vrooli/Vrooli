declare const __dirname: string;

interface NodeProcess {
  env: Record<string, string | undefined>;
  argv: string[];
}

declare const process: NodeProcess;

declare module "path" {
  export function resolve(...paths: string[]): string;
}
