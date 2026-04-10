/**
 * Utility to parse export settings from an existing Export record.
 * Used when editing an export to pre-populate the export dialog.
 */

import type { Export } from '../store';
import type {
  ExportFormat,
  ExportDimensionPreset,
  ExportRenderSource,
  ExportStylization,
} from '@/domains/executions/export/config/types';

/**
 * Parsed export settings that can be used to pre-populate the export dialog.
 */
export interface ParsedExportSettings {
  format: ExportFormat;
  fileStem: string;
  dimensionPreset?: ExportDimensionPreset;
  customWidth?: number;
  customHeight?: number;
  renderSource?: ExportRenderSource;
  stylization?: ExportStylization;
  chromeTheme?: string;
  background?: string;
  cursorTheme?: string;
  cursorScale?: number;
  browserScale?: number;
}

/**
 * Extract settings from an existing export for pre-populating the export dialog.
 */
export function parseExportSettings(export_: Export): ParsedExportSettings {
  const settings = export_.settings ?? {};

  return {
    format: export_.format,
    fileStem: export_.name,
    dimensionPreset: settings.dimensionPreset as ExportDimensionPreset | undefined,
    customWidth: typeof settings.customWidth === 'number' ? settings.customWidth : undefined,
    customHeight: typeof settings.customHeight === 'number' ? settings.customHeight : undefined,
    renderSource: settings.renderSource as ExportRenderSource | undefined,
    stylization: settings.stylization as ExportStylization | undefined,
    chromeTheme: typeof settings.chromeTheme === 'string' ? settings.chromeTheme : undefined,
    background: typeof settings.background === 'string' ? settings.background : undefined,
    cursorTheme: typeof settings.cursorTheme === 'string' ? settings.cursorTheme : undefined,
    cursorScale: typeof settings.cursorScale === 'number' ? settings.cursorScale : undefined,
    browserScale: typeof settings.browserScale === 'number' ? settings.browserScale : undefined,
  };
}
