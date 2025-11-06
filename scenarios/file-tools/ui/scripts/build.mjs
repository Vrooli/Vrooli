import { copyFile, mkdir, readdir, rm, stat } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const rootDir = path.resolve(__dirname, '..');
const distDir = path.join(rootDir, 'dist');
const srcDir = path.join(rootDir, 'src');

async function copyDirectory(source, destination) {
  const entries = await readdir(source, { withFileTypes: true });
  await mkdir(destination, { recursive: true });

  for (const entry of entries) {
    const sourcePath = path.join(source, entry.name);
    const destinationPath = path.join(destination, entry.name);

    if (entry.isDirectory()) {
      await copyDirectory(sourcePath, destinationPath);
      continue;
    }

    await copyFile(sourcePath, destinationPath);
  }
}

async function ensureSourceExists() {
  try {
    const info = await stat(srcDir);
    if (!info.isDirectory()) {
      throw new Error(`Expected ${srcDir} to be a directory`);
    }
  } catch (error) {
    throw new Error(`Static asset directory missing: ${srcDir}. ${error.message}`);
  }
}

async function build() {
  await ensureSourceExists();
  await rm(distDir, { recursive: true, force: true });
  await mkdir(distDir, { recursive: true });
  await copyDirectory(srcDir, distDir);
  console.log(`[file-tools-ui] build complete -> ${distDir}`);
}

build().catch((error) => {
  console.error('[file-tools-ui] build failed:', error);
  process.exitCode = 1;
});
