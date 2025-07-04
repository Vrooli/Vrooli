/**
 * MemberInvite Form Configuration
 * 
 * Unified configuration for MemberInvite forms that can be used by
 * UI components, validation, and integration testing.
 */
import { endpointsMemberInvite } from "../../api/pairs.js";
import type {
    MemberInvite,
    MemberInviteCreateInput,
    MemberInviteUpdateInput,
    Session,
} from "../../api/types.js";
import { DUMMY_ID } from "../../id/snowflake.js";
import type { MemberInviteShape } from "../../shape/models/models.js";
import { shapeMemberInvite } from "../../shape/models/models.js";
import { memberInviteValidation } from "../../validation/models/memberInvite.js";
import type { FormConfig } from "./types.js";

/**
 * Core MemberInvite form configuration
 */
export const memberInviteFormConfig: FormConfig<
    MemberInviteShape,
    MemberInviteCreateInput,
    MemberInviteUpdateInput,
    MemberInvite
> = {
    objectType: "MemberInvite",

    validation: {
        schema: memberInviteValidation,
        // MemberInvite doesn't have translation validation
        translationSchema: undefined,
    },

    transformations: {
        /** Convert form data to shape - for MemberInvite, form data is already in shape format */
        formToShape: (formData: MemberInviteShape) => formData,

        /** Shape transformation object - use the shape model directly */
        shapeToInput: shapeMemberInvite,

        /** Convert API response back to shape format */
        apiResultToShape: (apiResult: MemberInvite): MemberInviteShape => ({
            __typename: "MemberInvite" as const,
            id: apiResult.id,
            message: apiResult.message || null,
            willBeAdmin: apiResult.willBeAdmin || false,
            willHavePermissions: apiResult.willHavePermissions || "",
            team: {
                __typename: "Team" as const,
                id: apiResult.team.id,
            },
            user: apiResult.user ? {
                __typename: "User" as const,
                id: apiResult.user.id,
                handle: apiResult.user.handle || "",
                isBot: apiResult.user.isBot || false,
                name: apiResult.user.name || "",
                profileImage: apiResult.user.profileImage || null,
                updatedAt: apiResult.user.updatedAt,
            } : {
                __typename: "User" as const,
                id: "",
                handle: "",
                isBot: false,
                name: "",
                profileImage: null,
                updatedAt: new Date().toISOString(),
            },
        }),

        /** Generate initial values for the form */
        getInitialValues: (session?: Session, existing?: Partial<MemberInviteShape>) => {
            return {
                __typename: "MemberInvite" as const,
                id: existing?.id || DUMMY_ID,
                message: existing?.message || null,
                willBeAdmin: existing?.willBeAdmin || false,
                willHavePermissions: existing?.willHavePermissions || "",
                team: existing?.team || {
                    __typename: "Team" as const,
                    id: "",
                },
                user: existing?.user || {
                    __typename: "User" as const,
                    id: "",
                    handle: "",
                    isBot: false,
                    name: "",
                    profileImage: null,
                    updatedAt: new Date().toISOString(),
                },
            };
        },
    },

    endpoints: endpointsMemberInvite,
};
