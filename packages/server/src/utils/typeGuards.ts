// AI_CHECK: TYPE_SAFETY=phase1-guards | LAST: 2025-07-04 - Created type guard utilities for safer type checking
import { type ModelType } from "@vrooli/shared";
import { type BaseModelLogic } from "../models/base/index.js";
import { type ModelLogicType } from "../models/types.js";

/**
 * Type guard to check if a value has a specific property
 */
export function hasProperty<T extends object, K extends PropertyKey>(
    obj: T,
    key: K,
): obj is T & Record<K, unknown> {
    return key in obj;
}

/**
 * Type guard to check if a value has a string property
 */
export function hasStringProperty<T extends object, K extends PropertyKey>(
    obj: T,
    key: K,
): obj is T & Record<K, string> {
    return key in obj && typeof (obj as any)[key] === "string";
}

/**
 * Type guard to check if a value has an id property
 */
export function hasId<T extends object>(obj: T): obj is T & { id: string } {
    return hasStringProperty(obj, "id");
}

/**
 * Type guard to check if a value has a __typename property
 */
export function hasTypename<T extends object>(obj: T): obj is T & { __typename: `${ModelType}` } {
    return hasStringProperty(obj, "__typename");
}

/**
 * Type guard for checking if a value is a valid ModelLogic
 */
export function isModelLogic(value: unknown): value is BaseModelLogic {
    return (
        value !== null &&
        typeof value === "object" &&
        "__typename" in value &&
        "dbTable" in value &&
        typeof (value as any).dbTable === "string"
    );
}

/**
 * Type guard for checking if an object has auth data shape
 */
export function isAuthData(value: unknown): value is { __typename: `${ModelType}`, id: string } & Record<string, unknown> {
    return (
        value !== null &&
        typeof value === "object" &&
        hasTypename(value) &&
        hasId(value)
    );
}

/**
 * Type guard for task data with userId
 */
export function hasUserId<T extends object>(obj: T): obj is T & { userId: string } {
    return hasStringProperty(obj, "userId");
}

/**
 * Type guard for task data with status
 */
export function hasStatus<T extends object>(obj: T): obj is T & { status: string } {
    return hasStringProperty(obj, "status");
}

/**
 * Safe property access with type narrowing
 */
export function getProperty<T, K extends keyof T>(
    obj: T,
    key: K,
): T[K] | undefined {
    return obj[key];
}

/**
 * Safe string property access
 */
export function getStringProperty<T extends object>(
    obj: T,
    key: PropertyKey,
): string | undefined {
    if (hasStringProperty(obj, key)) {
        return (obj as any)[key];
    }
    return undefined;
}

/**
 * Type guard for checking if a value is a non-null object
 */
export function isObject(value: unknown): value is Record<string, unknown> {
    return value !== null && typeof value === "object";
}

/**
 * Type guard for checking if a value is a valid model type
 */
export function isValidModelType(value: string): value is `${ModelType}` {
    // This would ideally check against the ModelType enum values
    // For now, we'll do a basic check
    return typeof value === "string" && value.length > 0;
}

/**
 * Extract owner ID from various task data formats
 */
export function extractOwnerId(data: unknown): string | null {
    if (!isObject(data)) return null;
    
    // Direct userId
    if (hasUserId(data)) return data.userId;
    
    // startedById
    if (hasStringProperty(data, "startedById")) {
        return (data as any).startedById;
    }
    
    // Nested userData.id
    if (hasProperty(data, "userData") && isObject(data.userData)) {
        const userData = data.userData as Record<string, unknown>;
        if (hasId(userData)) {
            return userData.id;
        }
    }
    
    return null;
}

/**
 * Type-safe check for model permissions
 */
export function hasPermission<T extends Record<string, unknown>>(
    permissions: T,
    permission: keyof T,
): boolean {
    return permission in permissions && permissions[permission] === true;
}
