// One-shot offline entrypoint for source-derived boundary facts. This is an
// owner command over the same bundled ts-morph implementation used by the
// sidecar; it does not contact the TypeScript Code Graph API service.
import * as fs from "node:fs";
import * as path from "node:path";

import { Project } from "ts-morph";

import { BOUNDARY_FACTS_VERSION, extractBoundaryFacts } from "./boundary-facts.js";

export interface BoundaryFactsCLIRequest {
  protocol_version: number;
  root_dir: string;
  files: string[];
}

export interface BoundaryFactsCLIResponse {
  protocol_version: typeof BOUNDARY_FACTS_VERSION;
  facts: ReturnType<typeof extractBoundaryFacts>;
}

export function extractBoundaryFactsRequest(request: BoundaryFactsCLIRequest): BoundaryFactsCLIResponse {
  if (request.protocol_version !== BOUNDARY_FACTS_VERSION) {
    throw new Error(`unsupported boundary facts protocol ${request.protocol_version}; expected ${BOUNDARY_FACTS_VERSION}`);
  }
  if (!path.isAbsolute(request.root_dir)) {
    throw new Error("root_dir must be an absolute repository path");
  }

  const root = path.resolve(request.root_dir);
  const files = [...new Set(request.files.map((file) => {
    const resolved = path.resolve(root, file);
    const relative = path.relative(root, resolved);
    if (relative === ".." || relative.startsWith(`..${path.sep}`) || path.isAbsolute(relative)) {
      throw new Error(`source file escapes root_dir: ${file}`);
    }
    if (!fs.statSync(resolved).isFile()) {
      throw new Error(`source file is not a regular file: ${file}`);
    }
    return resolved;
  }))].sort();

  const project = new Project({
    skipAddingFilesFromTsConfig: true,
    compilerOptions: { allowJs: true, checkJs: false },
  });
  for (const file of files) project.addSourceFileAtPath(file);

  return {
    protocol_version: BOUNDARY_FACTS_VERSION,
    facts: extractBoundaryFacts({ project, rootDir: root, files }),
  };
}

export function runBoundaryFactsCLI(): void {
  try {
    const input = fs.readFileSync(0, "utf8");
    const request = JSON.parse(input) as BoundaryFactsCLIRequest;
    const response = extractBoundaryFactsRequest(request);
    process.stdout.write(`${JSON.stringify(response)}\n`);
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    process.stderr.write(`boundary-facts: ${message}\n`);
    process.exitCode = 1;
  }
}
