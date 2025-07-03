import { type ApiVersion, type ApiVersionShape } from "@vrooli/shared";
import { type CrudPropsDialog, type CrudPropsPage, type CrudPropsPartial, type FormProps, type ObjectViewProps } from "../../../types.js";

// Schema-related types for better type safety
export interface ApiSchema {
    openapi?: string;
    swagger?: string;
    asyncapi?: string;
    jsonSchema?: string;
    paths?: Record<string, Record<string, ApiMethodDefinition>>;
    channels?: Record<string, ApiChannelDefinition>;
    [key: string]: unknown;
}

export interface ApiMethodDefinition {
    summary?: string;
    description?: string;
    parameters?: ApiParameter[];
    responses?: Record<string, ApiResponse>;
    [key: string]: unknown;
}

export interface ApiChannelDefinition {
    publish?: {
        summary?: string;
        description?: string;
        message?: { payload?: unknown };
    };
    subscribe?: {
        summary?: string;
        description?: string;
        message?: { payload?: unknown };
    };
    parameters?: Record<string, ApiParameterDefinition>;
    [key: string]: unknown;
}

export interface ApiParameter {
    name: string;
    in: "path" | "query" | "header" | "cookie";
    description?: string;
    required?: boolean;
    schema?: Record<string, unknown>;
    [key: string]: unknown;
}

export interface ApiParameterDefinition {
    description?: string;
    schema?: Record<string, unknown>;
    [key: string]: unknown;
}

export interface ApiResponse {
    description?: string;
    payload?: unknown;
    [key: string]: unknown;
}

export interface ApiEndpoint {
    method: string;
    path: string;
    summary: string;
    description?: string;
    parameters?: ApiParameter[];
    responses?: Record<string, ApiResponse>;
}

export interface ApiRequestParams {
    [paramName: string]: string;
}

export interface ApiResponseData {
    loading?: boolean;
    error?: boolean;
    message?: string;
    status?: number;
    statusText?: string;
    headers?: Record<string, string>;
    data?: unknown;
}

type ApiUpsertPropsPage = CrudPropsPage;
type ApiUpsertPropsDialog = CrudPropsDialog<ApiVersion>;
type ApiUpsertPropsPartial = CrudPropsPartial<ApiVersion>;
export type ApiUpsertProps = ApiUpsertPropsPage | ApiUpsertPropsDialog | ApiUpsertPropsPartial;
export type ApiFormProps = FormProps<ApiVersion, ApiVersionShape> & {
    versions: string[];
}
export type ApiViewProps = ObjectViewProps<ApiVersion>
