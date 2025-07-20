import { type ModelType, type VisibilityType } from "@vrooli/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readManyWithEmbeddingsHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { type EmbeddableType } from "../../services/embedding.js";
import { type ApiEndpoint } from "../../types.js";

/**
 * Type guard to check if a ModelType is embeddable
 */
function isEmbeddableType(type: ModelType | `${ModelType}`): type is EmbeddableType {
    const embeddableTypes: EmbeddableType[] = ["Chat", "Issue", "Meeting", "Reminder", "ResourceVersion", "Run", "Tag", "Team", "User"];
    return embeddableTypes.includes(type as EmbeddableType);
}

/**
 * Common rate limit presets used across endpoints
 */
export const RateLimitPresets = {
    STRICT: { maxUser: 100 },
    LOW: { maxUser: 250 },
    MEDIUM: { maxUser: 500 },
    HIGH: { maxUser: 1000 },
    VERY_HIGH: { maxUser: 2000 },
} as const;

/**
 * Common permission combinations used across endpoints
 */
export const PermissionPresets = {
    READ_PUBLIC: { hasReadPublicPermissions: true },
    READ_PRIVATE: { hasReadPrivatePermissions: true },
    WRITE_PUBLIC: { hasWritePublicPermissions: true },
    WRITE_PRIVATE: { hasWritePrivatePermissions: true },
} as const;

/**
 * Configuration for a single endpoint method
 */
export interface EndpointMethodConfig {
    rateLimit?: { maxUser: number };
    permissions?: Parameters<typeof RequestService.assertRequestFrom>[1];
    visibility?: VisibilityType;
    useEmbeddings?: boolean;
    /** Custom implementation - if provided, will use this instead of the standard helper */
    customImplementation?: (args: { input: any; req: any; info: any }) => Promise<any>;
}

/**
 * Configuration for standard CRUD endpoints
 */
export interface StandardCrudConfig {
    /** The object type (e.g., "Tag", "Bookmark") - optional when only custom endpoints are provided */
    objectType?: ModelType | `${ModelType}`;

    /** Configuration for each endpoint method */
    endpoints: {
        findOne?: EndpointMethodConfig;
        findMany?: EndpointMethodConfig;
        createOne?: EndpointMethodConfig;
        updateOne?: EndpointMethodConfig;
    };

    /** Custom endpoints that don't follow the standard pattern */
    customEndpoints?: Record<string, ApiEndpoint<any, any>>;
}

/**
 * Type helper to extract the endpoints type based on config
 */
type ExtractEndpointsType<T extends StandardCrudConfig> = {
    [K in keyof T["endpoints"]]: T["endpoints"][K] extends EndpointMethodConfig
    ? K extends "findOne" ? ApiEndpoint<any, any>
    : K extends "findMany" ? ApiEndpoint<any, any>
    : K extends "createOne" ? ApiEndpoint<any, any>
    : K extends "updateOne" ? ApiEndpoint<any, any>
    : never
    : never;
} & (T["customEndpoints"] extends Record<string, ApiEndpoint<any, any>> ? T["customEndpoints"] : Record<string, never>);

/**
 * Creates standard CRUD endpoints with consistent patterns
 * 
 * @example
 * ```typescript
 * export const tag: EndpointsTag = createStandardCrudEndpoints({
 *     objectType: "Tag",
 *     endpoints: {
 *         findOne: { 
 *             rateLimit: RateLimitPresets.HIGH,
 *             permissions: PermissionPresets.READ_PUBLIC
 *         },
 *         findMany: { 
 *             rateLimit: RateLimitPresets.HIGH,
 *             permissions: PermissionPresets.READ_PUBLIC,
 *             useEmbeddings: true
 *         },
 *         createOne: { 
 *             rateLimit: RateLimitPresets.MEDIUM,
 *             permissions: PermissionPresets.WRITE_PRIVATE
 *         },
 *         updateOne: { 
 *             rateLimit: RateLimitPresets.MEDIUM,
 *             permissions: PermissionPresets.WRITE_PRIVATE
 *         }
 *     }
 * });
 * ```
 */
export function createStandardCrudEndpoints<T extends StandardCrudConfig>(
    config: T,
): ExtractEndpointsType<T> {
    const result: any = {};
    const { objectType, endpoints } = config;

    // Create findOne endpoint if configured
    if (endpoints.findOne) {
        const { rateLimit, permissions, customImplementation } = endpoints.findOne;
        result.findOne = async ({ input }: any, { req }: any, info: any) => {
            if (rateLimit) {
                await RequestService.get().rateLimit({ ...rateLimit, req });
            }
            if (permissions) {
                RequestService.assertRequestFrom(req, permissions);
            }

            if (customImplementation) {
                return customImplementation({ input, req, info });
            }

            if (!objectType) {
                throw new Error("objectType is required for standard CRUD endpoints without custom implementation");
            }

            return readOneHelper({ info, input, objectType, req });
        };
    }

    // Create findMany endpoint if configured
    if (endpoints.findMany) {
        const { rateLimit, permissions, visibility, useEmbeddings, customImplementation } = endpoints.findMany;
        result.findMany = async ({ input }: any, { req }: any, info: any) => {
            if (rateLimit) {
                await RequestService.get().rateLimit({ ...rateLimit, req });
            }
            if (permissions) {
                RequestService.assertRequestFrom(req, permissions);
            }

            if (customImplementation) {
                return customImplementation({ input, req, info });
            }

            if (!objectType) {
                throw new Error("objectType is required for standard CRUD endpoints without custom implementation");
            }

            if (useEmbeddings) {
                if (!isEmbeddableType(objectType)) {
                    throw new Error(`objectType "${objectType}" does not support embeddings. Only these types support embeddings: Chat, Issue, Meeting, Reminder, ResourceVersion, Run, Tag, Team, User`);
                }
                return readManyWithEmbeddingsHelper({ info, input, objectType, req, ...(visibility && { visibility }) });
            } else {
                return readManyHelper({ info, input, objectType, req, ...(visibility && { visibility }) });
            }
        };
    }

    // Create createOne endpoint if configured
    if (endpoints.createOne) {
        const { rateLimit, permissions, customImplementation } = endpoints.createOne;
        result.createOne = async ({ input }: any, { req }: any, info: any) => {
            if (rateLimit) {
                await RequestService.get().rateLimit({ ...rateLimit, req });
            }
            if (permissions) {
                RequestService.assertRequestFrom(req, permissions);
            }

            if (customImplementation) {
                return customImplementation({ input, req, info });
            }

            if (!objectType) {
                throw new Error("objectType is required for standard CRUD endpoints without custom implementation");
            }

            return createOneHelper({ info, input, objectType, req });
        };
    }

    // Create updateOne endpoint if configured
    if (endpoints.updateOne) {
        const { rateLimit, permissions, customImplementation } = endpoints.updateOne;
        result.updateOne = async ({ input }: any, { req }: any, info: any) => {
            if (rateLimit) {
                await RequestService.get().rateLimit({ ...rateLimit, req });
            }
            if (permissions) {
                RequestService.assertRequestFrom(req, permissions);
            }

            if (customImplementation) {
                return customImplementation({ input, req, info });
            }

            if (!objectType) {
                throw new Error("objectType is required for standard CRUD endpoints without custom implementation");
            }

            return updateOneHelper({ info, input, objectType, req });
        };
    }

    // Add custom endpoints if provided
    if (config.customEndpoints) {
        Object.assign(result, config.customEndpoints);
    }

    return result;
}
