/**
 * Comment Form Configuration
 * 
 * Unified configuration for Comment forms that can be used by
 * UI components, validation, and integration testing.
 */
import { endpointsComment } from "../../api/pairs.js";
import type {
    Comment,
    CommentCreateInput,
    CommentUpdateInput,
    Session,
} from "../../api/types.js";
import type { CommentShape } from "../../shape/models/models.js";
import { shapeComment } from "../../shape/models/models.js";
import { commentTranslationValidation, commentValidation } from "../../validation/models/comment.js";
import type { FormConfig } from "./types.js";

/**
 * Core Comment form configuration
 */
export const commentFormConfig: FormConfig<
    CommentShape,
    CommentCreateInput,
    CommentUpdateInput,
    Comment
> = {
    objectType: "Comment",

    validation: {
        schema: commentValidation,
        translationSchema: commentTranslationValidation,
    },

    transformations: {
        /** Convert form data to shape - for Comment, form data is already in shape format */
        formToShape: (formData: CommentShape) => formData,

        /** Shape transformation object - use the shape model directly */
        shapeToInput: shapeComment,

        /** Convert API response back to shape format - same logic as commentInitialValues with existing data */
        apiResultToShape: (apiResult: Comment): CommentShape => ({
            __typename: "Comment" as const,
            id: apiResult.id,
            commentedOn: apiResult.commentedOn,
            threadId: null, // API doesn't return threadId but shape expects it
            translations: apiResult.translations?.map(t => ({
                __typename: "CommentTranslation" as const,
                id: t.id,
                language: t.language,
                text: t.text,
            })) || [],
        }),

        /** Generate initial values for the form */
        getInitialValues: (session?: Session, existing?: Partial<CommentShape>) => {
            const language = existing?.translations?.[0]?.language || "en";
            
            return {
                __typename: "Comment" as const,
                id: existing?.id || "",
                commentedOn: existing?.commentedOn || { __typename: "Issue" as any, id: "" },
                threadId: existing?.threadId || null,
                translations: existing?.translations || [{
                    __typename: "CommentTranslation" as const,
                    id: "",
                    language,
                    text: "",
                }],
            };
        },
    },

    endpoints: endpointsComment,
};
