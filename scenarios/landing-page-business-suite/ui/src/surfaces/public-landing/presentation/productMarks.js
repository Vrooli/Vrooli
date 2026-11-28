/** Shared finite SVG grammar for the native renderer and Node-rendered head. */
export const productMarkPaths = Object.freeze({
  'letter-a': 'm6 24 10-17 10 17M11 19h10',
  landscape: 'M5 23V9h22v14H5Zm0-6 7-6 7 10 4-5 4 7M21 8v8',
  suite: 'm5 9 7 15 4-8 4 8 7-15',
  play: 'm11 8 12 8-12 8V8Z',
  pulse: 'M4 17h5l3-7 5 13 3-9h8',
});

export const productMarkDrawing = Object.freeze({
  viewBox: '0 0 32 32', fill: 'none', strokeWidth: 2.3,
  strokeLinecap: 'round', strokeLinejoin: 'round',
});

/** Own-property lookup: unknown configured marks never select a default brand. */
export function getProductMarkPath(kind) {
  return typeof kind === 'string' && Object.hasOwn(productMarkPaths, kind) ? productMarkPaths[kind] : undefined;
}
