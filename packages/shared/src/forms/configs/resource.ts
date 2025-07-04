/**
 * Resource Form Configuration
 * 
 * Unified configuration for Resource forms that can be used by
 * UI components, validation, and integration testing.
 */
import { endpointsResource } from "../../api/pairs.js";
import type {
    Resource,
    ResourceCreateInput,
    ResourceType,
    ResourceUpdateInput,
    Session,
} from "../../api/types.js";
import { ResourceUsedFor } from "../../shape/configs/base.js";
import { DUMMY_ID } from "../../id/snowflake.js";
import type { ResourceShape } from "../../shape/models/models.js";
import { shapeResource } from "../../shape/models/models.js";
import { orDefault } from "../../utils/orDefault.js";
import { resourceValidation } from "../../validation/models/resource.js";
import { createTransformFunction } from "../utils/transformHelpers.js";
import type { FormConfig } from "./types.js";

/**
 * Core Resource form configuration
 */
export const resourceFormConfig: FormConfig<
    ResourceShape,
    ResourceCreateInput,
    ResourceUpdateInput,
    Resource
> = {
    objectType: "Resource",

    validation: {
        schema: resourceValidation,
        // Resource doesn't have a separate translation validation
        translationSchema: undefined,
    },

    transformations: {
        /** Convert form data to shape - for Resource, form data is already in shape format */
        formToShape: (formData: ResourceShape) => formData,

        /** Shape transformation object - use the shape model directly */
        shapeToInput: shapeResource,

        /** Convert API response back to shape format */
        apiResultToShape: (apiResult: Resource): ResourceShape => ({
            __typename: "Resource" as const,
            id: apiResult.id,
            isInternal: apiResult.isInternal || false,
            isPrivate: apiResult.isPrivate || false,
            permissions: apiResult.permissions || "",
            resourceType: apiResult.resourceType,
            publicId: apiResult.publicId || null,
            owner: apiResult.owner ? {
                __typename: apiResult.owner.__typename,
                id: apiResult.owner.id,
            } : null,
            parent: apiResult.parent ? {
                __typename: "ResourceVersion" as const,
                id: apiResult.parent.id,
            } : null,
            tags: apiResult.tags?.map(t => ({
                __typename: "Tag" as const,
                id: t.id,
                tag: t.tag,
            })) || null,
            versions: apiResult.versions?.map(v => ({
                __typename: "ResourceVersion" as const,
                id: v.id,
                codeLanguage: v.codeLanguage,
                config: v.config,
                isAutomatable: v.isAutomatable,
                isComplete: v.isComplete,
                isPrivate: v.isPrivate,
                resourceSubType: v.resourceSubType,
                versionLabel: v.versionLabel,
                versionNotes: v.versionNotes,
                publicId: v.publicId || null,
                relatedVersions: null,
                translations: null,
            })) || null,
        }),

        /** Generate initial values for the form */
        getInitialValues: (session?: Session, existing?: Partial<ResourceShape>) => ({
            __typename: "Resource" as const,
            id: existing?.id || DUMMY_ID,
            isInternal: existing?.isInternal || false,
            isPrivate: existing?.isPrivate || false,
            permissions: existing?.permissions || "",
            resourceType: existing?.resourceType || "Note" as ResourceType,
            publicId: existing?.publicId || null,
            // Set default owner to current user if not provided and session is available
            owner: existing?.owner || (session?.users?.[0] ? {
                __typename: "User" as const,
                id: session.users[0].id,
            } : null),
            parent: existing?.parent || null,
            tags: existing?.tags || null,
            versions: existing?.versions || null,
        }),
    },

    endpoints: endpointsResource,
};
