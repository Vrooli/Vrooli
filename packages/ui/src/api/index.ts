import { ErrorKey } from "@local/shared";

export type Method = "GET" | "POST" | "PUT" | "DELETE";

export type ServerResponse<Output = any> = {
    errors?: {
        message: string;
        code?: ErrorKey;
    }[];
    data?: Output;
    version?: string;
};

export * from "./errorParser";
export * from "./fetchData";
export * from "./fetchWrapper";
export * from "./removeTypename";
export * from "./socket";

