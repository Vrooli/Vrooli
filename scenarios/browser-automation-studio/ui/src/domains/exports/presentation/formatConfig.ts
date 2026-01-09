/**
 * Export Format Configuration
 *
 * UI configuration for export format display.
 * Uses canonical types from executions/export/config/types.ts.
 */

import { Film, Image, FileJson, Globe } from 'lucide-react';
import type {
  ExportFormat,
  ExportFormatDisplayConfig,
} from '@/domains/executions/export/config/types';

/**
 * Export format display configuration.
 */
export const FORMAT_CONFIG: Record<ExportFormat, ExportFormatDisplayConfig> = {
  mp4: {
    icon: Film,
    color: 'text-purple-400',
    bgColor: 'bg-purple-500/10',
    label: 'MP4 Video',
  },
  gif: {
    icon: Image,
    color: 'text-pink-400',
    bgColor: 'bg-pink-500/10',
    label: 'Animated GIF',
  },
  json: {
    icon: FileJson,
    color: 'text-blue-400',
    bgColor: 'bg-blue-500/10',
    label: 'JSON Package',
  },
  html: {
    icon: Globe,
    color: 'text-green-400',
    bgColor: 'bg-green-500/10',
    label: 'HTML Bundle',
  },
};

/**
 * Gets format config with a fallback to json if format is unknown.
 */
export const getFormatConfig = (format: string): ExportFormatDisplayConfig => {
  return FORMAT_CONFIG[format as ExportFormat] ?? FORMAT_CONFIG.json;
};
