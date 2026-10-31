import fs from 'node:fs';
import path from 'node:path';

const srcRoot = path.resolve(__dirname, '../../../src');
const helpersRoot = path.resolve(__dirname, '../../helpers');
const importPattern =
  /\b(?:import|export)\s+(?:type\s+)?(?:[^'"]*?\s+from\s+)?['"]([^'"]+)['"]|import\(\s*['"]([^'"]+)['"]\s*\)/g;

describe('test utility boundaries', () => {
  it('keeps tests/helpers out of production modules', () => {
    const violations: string[] = [];

    for (const filePath of walkSourceFiles(srcRoot)) {
      if (!isProductionSource(filePath)) {
        continue;
      }

      const source = fs.readFileSync(filePath, 'utf8');
      for (const importPath of importSpecifiers(source)) {
        if (importsTestHelpers(filePath, importPath)) {
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
      yield* walkSourceFiles(fullPath);
      continue;
    }

    if (entry.name.endsWith('.ts')) {
      yield fullPath;
    }
  }
}

function isProductionSource(filePath: string): boolean {
  const normalized = path.relative(srcRoot, filePath).replaceAll(path.sep, '/');
  return !normalized.endsWith('.test.ts') && !normalized.endsWith('.d.ts');
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

function importsTestHelpers(fromFile: string, importPath: string): boolean {
  if (importPath.startsWith('tests/helpers') || importPath.includes('/tests/helpers/')) {
    return true;
  }

  if (!importPath.startsWith('.')) {
    return false;
  }

  const resolved = path.resolve(path.dirname(fromFile), importPath);
  return resolved === helpersRoot || resolved.startsWith(helpersRoot + path.sep);
}
