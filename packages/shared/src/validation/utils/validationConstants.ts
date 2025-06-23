/* eslint-disable no-magic-numbers */
/**
 * Validation constants used throughout the application.
 * These constants replace magic numbers in validation schemas to improve readability
 * and maintainability while ensuring consistency across validation rules.
 */

// Schedule and Time Constants
export const DAY_OF_WEEK_MIN = 1; // Sunday = 1
export const DAY_OF_WEEK_MAX = 7; // Saturday = 7
export const DAY_OF_MONTH_MIN = 1;
export const DAY_OF_MONTH_MAX = 31;

// Authentication and Security Constants
export const PASSWORD_VERIFICATION_CODE_MAX_LENGTH = 128;
export const TIMEZONE_MAX_LENGTH = 128;
export const API_KEY_EXTERNAL_MAX_LENGTH = 255;
export const API_KEY_SERVICE_MAX_LENGTH = 128;

// Data Size Constants - Small (< 1KB)
export const LABEL_MAX_LENGTH = 128;
export const NODE_NAME_MAX_LENGTH = 128;
export const NODE_INPUT_NAME_MAX_LENGTH = 128;
export const NODE_ID_MAX_LENGTH = 128;
export const CODE_LANGUAGE_MAX_LENGTH = 128;

// Data Size Constants - Medium (1KB - 10KB)
export const API_KEY_PERMISSIONS_MAX_LENGTH = 4096;
export const RUN_IO_DATA_MAX_LENGTH = 8192;

// Data Size Constants - Large (10KB+)
export const RUN_DATA_MAX_LENGTH = 16384;
export const PULL_REQUEST_TEXT_MAX_LENGTH = 32768;
export const COMMENT_TEXT_MAX_LENGTH = 32768;

// API Limits
export const API_KEY_ABSOLUTE_MAX_LIMIT = 1000000;

// Common Field Length Constants
export const NAME_MIN_LENGTH = 3;
export const NAME_MAX_LENGTH = 50;
export const TAG_MIN_LENGTH = 2;
export const TAG_MAX_LENGTH = 64;
export const BIO_MAX_LENGTH = 2048;
export const DESCRIPTION_MAX_LENGTH = 2048;
export const HELP_TEXT_MAX_LENGTH = 2048;
export const REFERENCING_MAX_LENGTH = 2048;
export const LANGUAGE_CODE_MIN_LENGTH = 2;
export const LANGUAGE_CODE_MAX_LENGTH = 3;
export const VERSION_LABEL_MAX_LENGTH = 16;
export const VERSION_NOTES_MAX_LENGTH = 4092;
export const EMAIL_MAX_LENGTH = 256;
export const HANDLE_MIN_LENGTH = 3;
export const HANDLE_MAX_LENGTH = 16;
export const HEX_COLOR_MIN_LENGTH = 4;
export const HEX_COLOR_MAX_LENGTH = 7;
export const IMAGE_FILE_MAX_LENGTH = 256;
export const PUSH_NOTIFICATION_KEY_MAX_LENGTH = 256;
export const URL_MAX_LENGTH = 1024;
export const TIMEZONE_FIELD_MAX_LENGTH = 64;

// Test-specific Constants
export const TEST_FIELD_TOO_SHORT_LENGTH = 2; // For testing minimum field length violations
export const TEST_FIELD_TOO_LONG_MULTIPLIER = 1; // Add 1 character beyond max for testing
export const TEST_SMALL_LIMIT = 50;
export const TEST_MEDIUM_LIMIT = 150;
export const TEST_LARGE_LIMIT = 300;

// HTTP Status Code Constants
export const HTTP_STATUS_OK = 200;
export const HTTP_STATUS_CREATED = 201;
export const HTTP_STATUS_BAD_REQUEST = 400;
