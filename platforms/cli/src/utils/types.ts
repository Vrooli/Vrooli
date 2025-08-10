// Re-export types that might have import issues
export type { AxiosError, AxiosRequestConfig } from "axios";

// AI_CHECK: TEST_COVERAGE=1 | LAST: 2025-07-14

export interface CliOptions {
    debug?: boolean;
    json?: boolean;
    profile?: string;
}
