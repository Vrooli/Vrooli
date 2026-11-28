import path from 'node:path'

export function resolveUIDistDir(configuredDist, packageDir) {
  const root = path.resolve(packageDir)
  const candidate = configuredDist?.trim()
    ? path.resolve(root, configuredDist)
    : path.join(root, 'dist')
  const relative = path.relative(root, candidate)
  if (relative === '..' || relative.startsWith(`..${path.sep}`) || path.isAbsolute(relative)) {
    throw new Error('BAS_UI_DIST_DIR must stay within the BAS UI package')
  }
  return candidate
}
