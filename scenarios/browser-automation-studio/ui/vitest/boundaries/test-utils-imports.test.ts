import { describe, expect, it } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';

const srcRoot = path.resolve(process.cwd(), 'src');
const testUtilsRoot = path.join(srcRoot, 'test-utils');
const importPattern =
  /\b(?:import|export)\s+(?:type\s+)?(?:[^'"]*?\s+from\s+)?['"]([^'"]+)['"]|import\(\s*['"]([^'"]+)['"]\s*\)/g;

describe('test utility boundaries', () => {
  it('keeps src/test-utils out of production modules', () => {
    const violations: string[] = [];

    for (const filePath of walkSourceFiles(srcRoot)) {
      if (!isProductionSource(filePath)) {
        continue;
      }

      const source = fs.readFileSync(filePath, 'utf8');
      for (const importPath of importSpecifiers(source)) {
        if (importsTestUtils(filePath, importPath)) {
          violations.push(`${path.relative(srcRoot, filePath)} imports ${importPath}`);
        }
      }
    }

    expect(violations).toEqual([]);
  });
});

function* walkSourceFiles(root: string): Generator<string> {
  for (const entry of fs.readdirSync(root, { withFileTypes: true })) {
    const fullPath = path.join(root, entry.name);
    if (entry.isDirectory()) {
      if (entry.name === 'test-utils') {
        continue;
      }
      yield* walkSourceFiles(fullPath);
      continue;
    }

    if (/\.(ts|tsx)$/.test(entry.name)) {
      yield fullPath;
    }
  }
}

function isProductionSource(filePath: string): boolean {
  const normalized = path.relative(srcRoot, filePath).replaceAll(path.sep, '/');
  return !/(\.test|\.spec)\.(ts|tsx)$/.test(normalized) && !normalized.endsWith('.d.ts');
}

function importSpecifiers(source: string): string[] {
  const imports: string[] = [];

  for (const match of source.matchAll(importPattern)) {
    const importPath = match[1] ?? match[2];
    if (importPath) {
      imports.push(importPath);
    }
  }

  return imports;
}

function importsTestUtils(fromFile: string, importPath: string): boolean {
  if (importPath === '@/test-utils' || importPath.startsWith('@/test-utils/')) {
    return true;
  }

  if (!importPath.startsWith('.')) {
    return false;
  }

  const resolved = path.resolve(path.dirname(fromFile), importPath);
  return resolved === testUtilsRoot || resolved.startsWith(testUtilsRoot + path.sep);
}
