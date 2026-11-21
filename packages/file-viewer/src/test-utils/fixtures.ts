import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

// Repository root: packages/file-viewer/src/test-utils -> four levels up.
const repositoryRoot = join(dirname(fileURLToPath(import.meta.url)), "..", "..", "..", "..");

/**
 * Real files the readers are proven against. The demo set under
 * `scenarios/web-console/fixtures/file-preview` is shared with Web Console's
 * README and API tests, so it stays there; this reads it by repository path.
 */
export function readRepositoryFile(relativePath: string): Buffer {
  return readFileSync(join(repositoryRoot, relativePath));
}

export function readPreviewFixture(name: string): Buffer {
  return readRepositoryFile(`scenarios/web-console/fixtures/file-preview/${name}`);
}

/** The fixture as a standalone ArrayBuffer (a Buffer's backing store is shared and pooled). */
export function previewFixtureBuffer(name: string): ArrayBuffer {
  return Uint8Array.from(readPreviewFixture(name)).buffer;
}
