// @vitest-environment node
import { readFileSync, readdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';
import ts from 'typescript';

const productCopy = /\b(?:Aquila|Web Console|Browser Automation Studio|Backdrop Studio)\b|All your agents|The tools\.\s*The possibilities/i;
const reviewImport = /(?:^|\/)(?:fixtures|preview|testFixtures|commerceFixtures|videoTestFixtures)(?:\/|\.|$)/;

/** A narrow provenance guard, not a proof that arbitrary future copy is correct. */
function violations(source: string, filename: string): string[] {
  const file = ts.createSourceFile(filename, source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  const found: string[] = [];
  function visit(node: ts.Node) {
    if ((ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) && node.moduleSpecifier && ts.isStringLiteral(node.moduleSpecifier)) {
      if (reviewImport.test(node.moduleSpecifier.text)) found.push('runtime review-fixture dependency');
    }
    if ((ts.isStringLiteralLike(node) || ts.isJsxText(node) || ts.isTemplateHead(node) || ts.isTemplateMiddle(node) || ts.isTemplateTail(node)) && productCopy.test(node.text)) {
      found.push('product-specific renderer copy');
    }
    ts.forEachChild(node, visit);
  }
  visit(file);
  return found;
}

describe('public presentation content provenance [REQ:LP-PRES-004] [REQ:LP-PRES-002]', () => {
  it('keeps named product copy and review fixtures out of production renderer modules', () => {
    const root = new URL('./', import.meta.url);
    const modules = readdirSync(root).filter(name => /\.tsx?$/.test(name)
      && !/\.test\./.test(name)
      && !/^(?:preview|testFixtures|commerceFixtures|videoTestFixtures)\./.test(name));
    expect(modules.length).toBeGreaterThan(10);
    for (const name of [...modules, '../routes/PublicLanding.tsx']) {
      const path = new URL(name, root);
      expect(violations(readFileSync(path, 'utf8'), fileURLToPath(path)), name).toEqual([]);
    }
  });

  it('rejects hardcoded text, attribute labels, templates and imported fixture defaults', () => {
    for (const source of [
      '<h1>Aquila</h1>', '<button aria-label="Explore Browser Automation Studio" />',
      'const title = `Web Console ${version}`;',
      'import content from "./fixtures/signal.json";',
      'export { fixture } from "./testFixtures";',
    ]) expect(violations(source, 'example.tsx').length).toBeGreaterThan(0);
  });

  it('allows canonical data access, technical identities and explanatory comments', () => {
    expect(violations('// Aquila is configuration, not a default.\nconst key = "web-console"; const page = <h1>{content.title}</h1>;', 'example.tsx')).toEqual([]);
  });
});
