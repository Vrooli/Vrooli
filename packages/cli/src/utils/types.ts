// Re-export types that might have import issues
export type { AxiosError, AxiosRequestConfig } from "axios";

export interface CliOptions {
    debug?: boolean;
    json?: boolean;
    profile?: string;
}