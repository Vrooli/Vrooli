/**
 * ChatInvite Form Configuration
 * 
 * Unified configuration for ChatInvite forms that can be used by
 * UI components, validation, and integration testing.
 * 
 * AI_CHECK: TYPE_SAFETY=4 | LAST: 2025-06-28
 */
import { endpointsChatInvite } from "../../api/pairs.js";
import type {
    ChatInvite,
    ChatInviteCreateInput,
    ChatInviteStatus,
    ChatInviteUpdateInput,
    Session,
} from "../../api/types.js";
import type { ChatInviteShape } from "../../shape/models/models.js";
import { shapeChatInvite } from "../../shape/models/models.js";
import { chatInviteValidation } from "../../validation/models/chatInvite.js";
import type { FormConfig } from "./types.js";

/**
 * Core ChatInvite form configuration
 */
export const chatInviteFormConfig: FormConfig<
    ChatInviteShape,
    ChatInviteCreateInput,
    ChatInviteUpdateInput,
    ChatInvite
> = {
    objectType: "ChatInvite",

    validation: {
        schema: chatInviteValidation,
        // ChatInvite doesn't have translation validation
        translationSchema: undefined,
    },

    transformations: {
        /** Convert form data to shape - for ChatInvite, form data is already in shape format */
        formToShape: (formData: ChatInviteShape) => formData,

        /** Shape transformation object - use the shape model directly */
        shapeToInput: shapeChatInvite,

        /** Convert API response back to shape format */
        apiResultToShape: (apiResult: ChatInvite): ChatInviteShape => ({
            __typename: "ChatInvite" as const,
            id: apiResult.id,
            createdAt: apiResult.createdAt,
            updatedAt: apiResult.updatedAt,
            status: apiResult.status,
            message: apiResult.message || null,
            chat: {
                __typename: "Chat" as const,
                id: apiResult.chat.id,
            },
            user: apiResult.user ? {
                __typename: "User" as const,
                id: apiResult.user.id,
            } : {
                __typename: "User" as const,
                id: "",
            },
        }),

        /** Generate initial values for the form */
        getInitialValues: (session?: Session, existing?: Partial<ChatInviteShape>) => {
            return {
                __typename: "ChatInvite" as const,
                id: existing?.id || "",
                createdAt: existing?.createdAt || new Date().toISOString(),
                updatedAt: existing?.updatedAt || new Date().toISOString(),
                status: existing?.status || "Pending" as ChatInviteStatus,
                message: existing?.message || null,
                chat: existing?.chat || {
                    __typename: "Chat" as const,
                    id: "",
                },
                user: existing?.user || {
                    __typename: "User" as const,
                    id: "",
                },
            };
        },
    },

    endpoints: endpointsChatInvite,
};
