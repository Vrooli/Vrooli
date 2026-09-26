const prefix = "@vrooli/react-component-library/";
const grammar = /^@vrooli\/react-component-library\/([A-Za-z][A-Za-z0-9-]*)(?:\/(\d+|\d+\.\d+\.\d+))?$/;
const release = /^\d+\.\d+\.\d+$/;

export function parseLibrarySpecifier(value) {
  const match = grammar.exec(String(value).trim());
  return match ? { name: match[1], selector: match[2] ?? "" } : null;
}

export function parseLibrarySpecifiers(source) {
  const result = new Map();
  for (const token of String(source).split(/["'`\s,;(){}]+/).filter(Boolean)) {
    const parsed = parseLibrarySpecifier(token);
    if (parsed) result.set(`${parsed.name}/${parsed.selector}`, parsed);
  }
  return [...result.values()].sort((a, b) => `${a.name}/${a.selector}`.localeCompare(`${b.name}/${b.selector}`));
}

export const isReleaseSpecifier = (value) => release.test(value);
export const librarySpecifierPrefix = prefix;
