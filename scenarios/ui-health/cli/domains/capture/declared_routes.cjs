// Parse declarations only. Never import or execute the scenario being inspected.
const ts = require(process.argv[1]);
const input = JSON.parse(require('node:fs').readFileSync(0, 'utf8'));
const routes = ts.createSourceFile('routes.ts', input.routes, ts.ScriptTarget.Latest, true);
const app = ts.createSourceFile('App.tsx', input.app, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
if (routes.parseDiagnostics.length || app.parseDiagnostics.length) {
  throw new Error('route declarations contain syntax errors');
}
const unwrap = (n) => {
  while (n && (ts.isAsExpression(n) || ts.isSatisfiesExpression(n) || ts.isParenthesizedExpression(n))) n = n.expression;
  return n;
};
const tables = new Map();
for (const statement of routes.statements) {
  if (!ts.isVariableStatement(statement) || !(statement.declarationList.flags & ts.NodeFlags.Const)) continue;
  for (const declaration of statement.declarationList.declarations) {
    const value = unwrap(declaration.initializer);
    if (!ts.isIdentifier(declaration.name) || !value || !ts.isObjectLiteralExpression(value)) continue;
    const table = new Map();
    for (const property of value.properties) {
      if (ts.isPropertyAssignment(property) && ts.isStringLiteral(property.initializer)) {
        table.set(property.name.text, property.initializer.text);
      }
    }
    tables.set(declaration.name.text, table);
  }
}
const aliases = new Map();
const routerNames = new Set();
for (const statement of app.statements) {
  if (!ts.isImportDeclaration(statement) || !ts.isStringLiteral(statement.moduleSpecifier)) continue;
  const named = statement.importClause?.namedBindings;
  if (!named || !ts.isNamedImports(named)) continue;
  for (const spec of named.elements) {
    const name = (spec.propertyName || spec.name).text;
    if (/^\.\/routes(?:\.ts)?$/.test(statement.moduleSpecifier.text) && tables.has(name)) aliases.set(spec.name.text, tables.get(name));
    if (['react-router', 'react-router-dom'].includes(statement.moduleSpecifier.text) && name === 'Route') routerNames.add(spec.name.text);
  }
}
function routeValue(initializer) {
  let value = initializer;
  if (value && ts.isJsxExpression(value)) value = unwrap(value.expression);
  if (value && ts.isStringLiteral(value)) return value.text;
  if (value && ts.isPropertyAccessExpression(value) && ts.isIdentifier(value.expression)) {
    return aliases.get(value.expression.text)?.get(value.name.text);
  }
}
const results = [];
function walk(node, nested = false) {
  if (ts.isJsxElement(node) || ts.isJsxSelfClosingElement(node)) {
    const opening = ts.isJsxElement(node) ? node.openingElement : node;
    if (routerNames.has(opening.tagName.getText(app))) {
      const attributes = opening.attributes.properties;
      const attribute = (name) => attributes.find(p => ts.isJsxAttribute(p) && p.name.text === name)?.initializer;
      const route = routeValue(attribute('path'));
      const element = attribute('element');
      // Nested paths and templates require route context/parameters; abstain.
      if (!nested && route?.startsWith('/') && !/[\:*]/.test(route) && !route.startsWith('//') && element) {
        function subjects(child) {
          if (ts.isJsxOpeningElement(child) || ts.isJsxSelfClosingElement(child)) {
            const subject = child.tagName.getText(app);
            if (/^[A-Z]\w*$/.test(subject)) results.push({subject, route});
          }
          ts.forEachChild(child, subjects);
        }
        subjects(element);
      }
      if (ts.isJsxElement(node)) {
        for (const child of node.children) walk(child, nested || !!attribute('path'));
      }
      return;
    }
  }
  ts.forEachChild(node, child => walk(child, nested));
}
walk(app);
process.stdout.write(JSON.stringify(results));
