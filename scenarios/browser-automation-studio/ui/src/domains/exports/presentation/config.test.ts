import { describe, expect, it } from 'vitest';
import {
  getExportStatusConfig,
  getExecutionStatusConfig,
  EXPORT_STATUS_CONFIG,
  EXECUTION_STATUS_CONFIG,
} from './statusConfig';
import { getFormatConfig, FORMAT_CONFIG } from './formatConfig';

describe('getExportStatusConfig', () => {
  it('returns correct config for pending', () => {
    const config = getExportStatusConfig('pending');
    expect(config.label).toBe('Pending');
    expect(config.color).toBe('text-gray-400');
  });

  it('returns correct config for processing', () => {
    const config = getExportStatusConfig('processing');
    expect(config.label).toBe('Processing');
    expect(config.color).toBe('text-yellow-400');
  });

  it('returns correct config for completed', () => {
    const config = getExportStatusConfig('completed');
    expect(config.label).toBe('Ready');
    expect(config.color).toBe('text-green-400');
  });

  it('returns correct config for failed', () => {
    const config = getExportStatusConfig('failed');
    expect(config.label).toBe('Failed');
    expect(config.color).toBe('text-red-400');
  });

  it('returns pending config for unknown status', () => {
    const config = getExportStatusConfig('unknown');
    expect(config.label).toBe('Pending');
  });
});

describe('getExecutionStatusConfig', () => {
  it('returns correct config for running', () => {
    const config = getExecutionStatusConfig('running');
    expect(config.label).toBe('Running');
    expect(config.color).toBe('text-green-400');
  });

  it('returns correct config for completed', () => {
    const config = getExecutionStatusConfig('completed');
    expect(config.label).toBe('Completed');
    expect(config.color).toBe('text-emerald-400');
  });

  it('returns correct config for cancelled', () => {
    const config = getExecutionStatusConfig('cancelled');
    expect(config.label).toBe('Cancelled');
    expect(config.color).toBe('text-amber-400');
  });

  it('returns pending config for unknown status', () => {
    const config = getExecutionStatusConfig('unknown');
    expect(config.label).toBe('Pending');
  });
});

describe('getFormatConfig', () => {
  it('returns correct config for mp4', () => {
    const config = getFormatConfig('mp4');
    expect(config.label).toBe('MP4 Video');
    expect(config.color).toBe('text-purple-400');
  });

  it('returns correct config for gif', () => {
    const config = getFormatConfig('gif');
    expect(config.label).toBe('Animated GIF');
    expect(config.color).toBe('text-pink-400');
  });

  it('returns correct config for json', () => {
    const config = getFormatConfig('json');
    expect(config.label).toBe('JSON Package');
    expect(config.color).toBe('text-blue-400');
  });

  it('returns correct config for html', () => {
    const config = getFormatConfig('html');
    expect(config.label).toBe('HTML Bundle');
    expect(config.color).toBe('text-green-400');
  });

  it('returns json config for unknown format', () => {
    const config = getFormatConfig('unknown');
    expect(config.label).toBe('JSON Package');
  });
});

describe('config objects have required properties', () => {
  it('all export status configs have icon, color, bgColor, label', () => {
    for (const [key, config] of Object.entries(EXPORT_STATUS_CONFIG)) {
      expect(config.icon).toBeDefined();
      expect(config.color).toBeDefined();
      expect(config.bgColor).toBeDefined();
      expect(config.label).toBeDefined();
    }
  });

  it('all execution status configs have icon, color, bgColor, label', () => {
    for (const [key, config] of Object.entries(EXECUTION_STATUS_CONFIG)) {
      expect(config.icon).toBeDefined();
      expect(config.color).toBeDefined();
      expect(config.bgColor).toBeDefined();
      expect(config.label).toBeDefined();
    }
  });

  it('all format configs have icon, color, bgColor, label', () => {
    for (const [key, config] of Object.entries(FORMAT_CONFIG)) {
      expect(config.icon).toBeDefined();
      expect(config.color).toBeDefined();
      expect(config.bgColor).toBeDefined();
      expect(config.label).toBeDefined();
    }
  });
});
