/**
 * Utility functions for scenario inventory
 */

export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
}

export const platformIcons: Record<string, string> = {
  win: '🪟',
  mac: '🍎',
  linux: '🐧'
};

export const platformNames: Record<string, string> = {
  win: 'Windows',
  mac: 'macOS',
  linux: 'Linux'
};
