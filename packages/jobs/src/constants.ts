/**
 * Constants used across job schedules
 */

// Batch processing sizes
export const BATCH_SIZE_SMALL = 100;
export const BATCH_SIZE_LARGE = 1000;

// Sitemap generation limits
export const MAX_ENTRIES_PER_SITEMAP = 50000;
export const BYTES_PER_KILOBYTE = 1024;
export const SITEMAP_SIZE_LIMIT_MB = 50;
export const MAX_SITEMAP_FILE_SIZE_BYTES = SITEMAP_SIZE_LIMIT_MB * BYTES_PER_KILOBYTE * BYTES_PER_KILOBYTE;
export const ESTIMATED_ENTRY_OVERHEAD_BYTES = 100;

// API limits  
export const API_BATCH_SIZE = 100;
