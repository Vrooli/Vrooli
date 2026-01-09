/**
 * Utility functions for scenario inventory
 *
 * @deprecated This module exists only for backward compatibility.
 * New code should import directly from the domain layer:
 *   import { formatBytes, getPlatformIcon, getPlatformName } from "../../domain/download";
 *
 * These exports will be removed in a future version.
 */

export {
  formatBytes,
  PLATFORM_DISPLAY_INFO,
  getPlatformIcon,
  getPlatformName
} from "../../domain/download";
