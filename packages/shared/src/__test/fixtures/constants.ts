/* c8 ignore start */
/* eslint-disable no-magic-numbers */
/**
 * Shared constants for test fixtures
 * 
 * Centralized location for common constants used across fixture files
 * to eliminate magic numbers and improve maintainability.
 */

// Time constants
export const MILLISECONDS_PER_SECOND = 1000;
export const SECONDS_PER_MINUTE = 60;
export const MINUTES_PER_HOUR = 60;
export const HOURS_PER_DAY = 24;
export const DAYS_PER_WEEK = 7;
export const DAYS_PER_MONTH = 30;
export const MONTHS_PER_YEAR = 12;

// Common time durations
export const MINUTES_TO_MS = SECONDS_PER_MINUTE * MILLISECONDS_PER_SECOND;
export const HOURS_TO_MS = MINUTES_PER_HOUR * SECONDS_PER_MINUTE * MILLISECONDS_PER_SECOND;
export const MILLISECONDS_PER_HOUR = HOURS_TO_MS;
export const DAYS_TO_MS = HOURS_PER_DAY * HOURS_TO_MS;
export const WEEKS_TO_MS = DAYS_PER_WEEK * DAYS_TO_MS;

// Specific hour durations
export const TWO_HOURS_TO_MS = 2 * HOURS_TO_MS;
export const THREE_HOURS_TO_MS = 3 * HOURS_TO_MS;
export const FOUR_HOURS_TO_MS = 4 * HOURS_TO_MS;
export const SIX_HOURS_TO_MS = 6 * HOURS_TO_MS;

// Specific day durations
export const TWO_DAYS_TO_MS = 2 * DAYS_TO_MS;
export const THREE_DAYS_TO_MS = 3 * DAYS_TO_MS;

// Specific time periods
export const ONE_SECOND_MS = MILLISECONDS_PER_SECOND;
export const FIVE_SECONDS_MS = 5 * ONE_SECOND_MS;
export const TEN_SECONDS_MS = 10 * ONE_SECOND_MS;
export const THIRTY_SECONDS_MS = 30 * ONE_SECOND_MS; // 30 seconds  
export const ONE_MINUTE_MS = MINUTES_TO_MS;
export const TWO_MINUTES_MS = 2 * ONE_MINUTE_MS;
export const FIVE_MINUTES_MS = 5 * ONE_MINUTE_MS;
export const TEN_MINUTES_MS = 10 * ONE_MINUTE_MS;
export const FIFTEEN_MINUTES_MS = 15 * ONE_MINUTE_MS;
export const THIRTY_MINUTES_MS = 30 * ONE_MINUTE_MS;
export const ONE_HOUR_MS = HOURS_TO_MS;
export const TWO_HOURS_MS = 2 * ONE_HOUR_MS;
export const TWELVE_HOURS_MS = 12 * ONE_HOUR_MS;
export const ONE_DAY_MS = DAYS_TO_MS;
export const THREE_DAYS_MS = 3 * ONE_DAY_MS;
export const SEVEN_DAYS_MS = 7 * ONE_DAY_MS;
export const ONE_WEEK_MS = WEEKS_TO_MS;
export const THIRTY_DAYS_MS = 30 * ONE_DAY_MS;
export const NINETY_DAYS_MS = 90 * ONE_DAY_MS;
export const ONE_YEAR_MS = 365 * ONE_DAY_MS;

// Common business values (only meaningful constants)
export const ONE_HUNDRED = 100;
export const FIVE_HUNDRED = 500;
export const ONE_THOUSAND = 1000;
export const TEN_THOUSAND = 10000;

// Common percentages (as decimals)
export const TEN_PERCENT = 0.1;
export const TWENTY_PERCENT = 0.2;
export const FIFTY_PERCENT = 0.5;
export const EIGHTY_PERCENT = 0.8;
export const NINETY_PERCENT = 0.9;

// Common defaults
export const DEFAULT_ERROR_RATE = TEN_PERCENT;
export const DEFAULT_COUNT = 10;
export const DEFAULT_DELAY_MS = FIVE_HUNDRED;
export const DEFAULT_TIMEOUT_MS = THIRTY_SECONDS_MS;

// API and data limits
export const MAX_NAME_LENGTH = ONE_HUNDRED;
export const MAX_DESCRIPTION_LENGTH = FIVE_HUNDRED;
export const MAX_KEYS_PER_USER = 50;
export const API_KEY_LENGTH = 64;
export const SHORT_ID_LENGTH = 8;
export const MIN_PASSWORD_LENGTH = 8;
export const MIN_KEY_LENGTH_FOR_MASKING = 8;

// Credit system constants
export const DEFAULT_NEW_USER_CREDITS = ONE_HUNDRED;
export const DEFAULT_FREE_CREDITS = 10;
export const DEFAULT_DAILY_CREDIT_LIMIT = ONE_THOUSAND;
export const DEFAULT_EXISTING_USER_CREDITS = FIVE_HUNDRED;

// Network constants
export const LOCALHOST_IP = "127.0.0.1";
export const PRIVATE_IP_BASE = "192.168.1.";
export const PRIVATE_IP_FIRST = "192.168.1.1";

// Test device constants
export const TEST_DEVICE_ID = "test-device";
export const TEST_DEVICE_NAME = "Test Browser";
export const LOGIN_DEVICE_ID = "login-device";
export const LOGIN_DEVICE_NAME = "Login Device";
export const NEW_DEVICE_ID = "new-device";
export const NEW_DEVICE_NAME = "Signup Device";

// Common test data counts
export const DEFAULT_RECENT_ACHIEVEMENTS_COUNT = 5;
export const DEFAULT_CHAT_INVITES_COUNT = 5;
export const DEFAULT_BULK_OPERATION_COUNT = 10;

/* c8 ignore stop */
