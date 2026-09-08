// JSX ownership analysis. Called by the ordinary ui-health rule and its tests.
// Parsing prevents comments, string literals and unused imports from becoming
// evidence of rendered chrome or a library mount.
const path = require('node:path').posix;

function analyze(input, ts) {
  const files = new Map(input.files.map(file => [file.path, file.content]));
  const modules = new Map();
  const findings = [];
  const shell = input.shell;
  const add = (file, node, element, reason) => findings.push({
    file, line: node ? modules.get(file).ast.getLineAndCharacterOfPosition(node.getStart()).line + 1 : 1,
    element, reason,
  });
  function literal(node) {
    if (!node) return '';
    if (ts.isStringLiteralLike(node)) return node.text;
    if (ts.isJsxExpression(node)) return literal(node.expression);
    return '';
  }
  function attrs(node) {
    return Object.fromEntries(node.attributes.properties.filter(ts.isJsxAttribute)
      .map(attr => [attr.name.getText(), literal(attr.initializer)]));
  }
  function resolve(file, specifier) {
    if (!specifier.startsWith('.')) return null;
    const base = path.normalize(path.join(path.dirname(file), specifier));
    return [base, ...['.tsx', '.ts', '.jsx', '.js', '/index.tsx', '/index.ts', '/index.jsx', '/index.js'].map(ext => base + ext)]
      .find(candidate => files.has(candidate)) || null;
  }
  for (const [file, content] of files) {
    if (!/\.[jt]sx?$/.test(file)) continue;
    const ast = ts.createSourceFile(file, content, ts.ScriptTarget.Latest, true,
      file.endsWith('.tsx') ? ts.ScriptKind.TSX : file.endsWith('.ts') ? ts.ScriptKind.TS : file.endsWith('.jsx') ? ts.ScriptKind.JSX : ts.ScriptKind.JS);
    const imports = new Map(), exports = new Map(), locals = new Map();
    for (const statement of ast.statements) {
      if (ts.isImportDeclaration(statement) && ts.isStringLiteral(statement.moduleSpecifier)) {
        const source = statement.moduleSpecifier.text, clause = statement.importClause;
        if (clause?.isTypeOnly) continue;
        if (clause?.name) imports.set(clause.name.text, { source, name:'default' });
        if (clause?.namedBindings && ts.isNamedImports(clause.namedBindings)) {
          for (const item of clause.namedBindings.elements) if (!item.isTypeOnly)
            imports.set(item.name.text, { source, name:item.propertyName?.text || item.name.text });
        } else if (clause?.namedBindings) imports.set(clause.namedBindings.name.text, { source, name:'*' });
      }
      const exported = statement.modifiers?.some(m => m.kind === ts.SyntaxKind.ExportKeyword);
      const isDefault = statement.modifiers?.some(m => m.kind === ts.SyntaxKind.DefaultKeyword);
      if (ts.isFunctionDeclaration(statement)) {
        const name = statement.name?.text || 'default';
        locals.set(name, statement);
        if (exported) exports.set(isDefault ? 'default' : name, { node:statement });
      }
      if (ts.isVariableStatement(statement)) for (const declaration of statement.declarationList.declarations) {
        if (!ts.isIdentifier(declaration.name)) continue;
        locals.set(declaration.name.text, declaration.initializer);
        if (exported) exports.set(declaration.name.text, { node:declaration.initializer });
      }
      if (ts.isExportAssignment(statement)) exports.set('default', { node:statement.expression });
      if (ts.isExportDeclaration(statement) && !statement.isTypeOnly && statement.exportClause && ts.isNamedExports(statement.exportClause)) {
        for (const item of statement.exportClause.elements) if (!item.isTypeOnly) exports.set(item.name.text, {
          local:item.propertyName?.text || item.name.text, source:literal(statement.moduleSpecifier),
        });
      }
    }
    modules.set(file, { ast, imports, exports, locals });
    if (ast.parseDiagnostics.length) add(file, null, 'source', 'JSX could not be parsed; repair syntax before ownership can be verified');
  }

  const seen = new Set();
  const ownedNodes = new Set();
  let mounted = false;
  function isLibraryShell(source, name) {
    const match = source.match(/^@vrooli\/react-component-library\/([^/]+)(?:\/([0-9]+)(?:\.[0-9]+\.[0-9]+)?)?$/);
    return Boolean(match && match[1] === shell.asset && (name === shell.asset || name === 'default'));
  }
  function visitExport(file, name) {
    const key = `${file}:${name}`;
    if (seen.has(key)) return;
    seen.add(key);
    const mod = modules.get(file), exported = mod?.exports.get(name);
    if (!exported) return;
    if (exported.source) {
      if (isLibraryShell(exported.source, exported.local)) mounted = true;
      const target = resolve(file, exported.source);
      if (target) visitExport(target, exported.local);
    } else {
      const node = exported.node || mod.locals.get(exported.local);
      if (node && ts.isIdentifier(node)) visitComponent(file, node.text);
      else visitNode(file, node);
    }
  }
  function visitComponent(file, name) {
    const mod = modules.get(file);
    const [base, member] = name.split('.');
    const imp = mod.imports.get(base);
    if (imp) {
      const exported = member || imp.name;
      if (isLibraryShell(imp.source, exported)) mounted = true;
      const target = resolve(file, imp.source);
      if (target) visitExport(target, exported);
    } else if (mod.locals.has(name) && !seen.has(`${file}:local:${name}`)) {
      seen.add(`${file}:local:${name}`); visitNode(file, mod.locals.get(name));
    }
  }
  function visitNode(file, node, root = true) {
    if (!node) return;
    if (root) ownedNodes.add(node);
    if (!root && (ts.isFunctionDeclaration(node) || ts.isVariableDeclaration(node))) return;
    if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) visitComponent(file, node.tagName.getText());
    ts.forEachChild(node, child => visitNode(file, child, false));
  }
  if (!modules.has(shell.entry)) add(shell.entry, null, 'module', 'declared shell entry does not exist');
  else if (!modules.get(shell.entry).exports.has(shell.export)) add(shell.entry, null, shell.export, 'declared shell export does not exist');
  else visitExport(shell.entry, shell.export);

  // CSS-backed board frames matter as much as utility-class frames. Only
  // single-element selectors can establish a frame here; descendant selectors
  // must not label unrelated children as viewport owners.
  const frameClasses = new Set(), frameAttributes = new Set();
  for (const [file, content] of files) if (file.endsWith('.css')) {
    for (const match of content.matchAll(/([^{}]+)\{([^{}]+)\}/g)) {
      if (!/(?:position\s*:\s*(?:fixed|absolute)[^}]*inset\s*:\s*0|(?:height|block-size)\s*:\s*100d?vh)/.test(match[2])) continue;
      for (const selector of match[1].split(',').map(s => s.trim())) {
        if (/^\.[\w-]+$/.test(selector)) frameClasses.add(selector.slice(1));
        const attr = selector.match(/^\[([\w-]+)\]$/); if (attr) frameAttributes.add(attr[1]);
      }
    }
  }
  for (const [file, mod] of modules) {
    function inspect(node, contentScope = false, overlay = false, applicationOwner = false) {
      applicationOwner ||= ownedNodes.has(node);
      if (ts.isJsxElement(node) || ts.isJsxSelfClosingElement(node)) {
        const opening = ts.isJsxElement(node) ? node.openingElement : node;
        const tag = opening.tagName.getText(), properties = attrs(opening);
        overlay ||= properties.role === 'dialog' || properties.role === 'alertdialog' || properties['aria-modal'] === 'true';
        // BottomNav is viewport-fixed application chrome, even when a page
        // nests it inside a section. AppShell already owns these tabs.
        const [importBase, importMember] = tag.split('.');
        const navigationImport = mod.imports.get(importBase);
        if (!overlay && shell.archetype === 'navigated-console' && navigationImport &&
            /^@vrooli\/react-component-library\/BottomNav(?:\/[0-9]+(?:\.[0-9]+\.[0-9]+)?)?$/.test(navigationImport.source) &&
            ['BottomNav', 'default'].includes(importMember || navigationImport.name)) {
          add(file, opening, `<${tag}>`, 'standalone application navigation competes with the declared shell; use its navigation items and keep page section navigation in the page flow');
        }
        const classes = (properties.className || '').split(/\s+/);
        const frame = classes.some(c => frameClasses.has(c) || /^(?:min-)?h-(?:screen|dvh)$/.test(c)) ||
          Object.keys(properties).some(a => frameAttributes.has(a) || /^data-rcl-(?:app|ambient|sidebar)-shell$/.test(a)) ||
          (/\bfixed\b/.test(properties.className || '') && /\binset-0\b/.test(properties.className || ''));
        let appLinks = false, brandLink = false, containsDialog = false;
        function links(child) {
          if (ts.isJsxOpeningElement(child) || ts.isJsxSelfClosingElement(child)) {
            const childProps = attrs(child), childTag = child.tagName.getText();
            if (["dialog", "alertdialog"].includes(childProps.role) || childProps["aria-modal"] === "true") containsDialog = true;
            const imp = mod.imports.get(childTag);
            if (childTag === 'a' || (imp && /react-router/.test(imp.source) && ['Link','NavLink'].includes(imp.name))) {
              const href = childProps.to || childProps.href || '';
              if (href === '/') brandLink = true;
              if (href.startsWith('/') || (imp && ('to' in childProps))) appLinks = true;
            }
          }
          ts.forEachChild(child, links);
        }
        links(node);
        const fallbackOrDecoration = !appLinks && (properties.role === 'alert' || properties['aria-hidden'] === 'true');
        const reason = !overlay && !contentScope && /^[a-z]/.test(tag) && (
          frame && !fallbackOrDecoration && !containsDialog ? 'locally rendered viewport or board frame' :
          properties.role === 'banner' || (tag === 'header' && (brandLink || applicationOwner)) ? 'locally rendered application header' :
          (['nav','aside'].includes(tag) && (appLinks || properties.role === 'navigation')) ? 'locally rendered application navigation' : ''
        );
        if (reason) add(file, opening, `<${tag}>`, reason);
        contentScope ||= fallbackOrDecoration || (['main','article','section'].includes(tag) && !frame);
        // JSX child scope is lexical: a page section may own a local header,
        // while a sibling navigation column remains application chrome.
        for (const attr of opening.attributes.properties) if (ts.isJsxAttribute(attr) && attr.initializer) inspect(attr.initializer, contentScope, overlay, applicationOwner);
        if (ts.isJsxElement(node)) for (const child of node.children) inspect(child, contentScope, overlay, applicationOwner);
        return;
      }
      ts.forEachChild(node, child => inspect(child, contentScope, overlay, applicationOwner));
    }
    inspect(mod.ast);
  }
  return { mounted, findings };
}

module.exports = { analyze };
if (!module.parent) {
  const input = JSON.parse(require('node:fs').readFileSync(0, 'utf8'));
  const ts = require(input.typescript);
  process.stdout.write(JSON.stringify(analyze(input, ts)));
}
