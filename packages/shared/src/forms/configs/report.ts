/**
 * Report Form Configuration
 * 
 * Unified configuration for Report forms that can be used by
 * UI components, validation, and integration testing.
 * 
 * AI_CHECK: TYPE_SAFETY=shared-type-safety-fixes | LAST: 2025-07-01 - Fixed type safety issues with proper return types, null handling, and type assertions
 */
import { endpointsReport } from "../../api/pairs.js";
import type {
    Report,
    ReportCreateInput,
    ReportUpdateInput,
    Session,
    ReportFor,
} from "../../api/types.js";
import { DUMMY_ID } from "../../id/snowflake.js";
import type { ReportShape } from "../../shape/models/models.js";
import { shapeReport } from "../../shape/models/models.js";
import { reportValidation } from "../../validation/models/report.js";
import type { FormConfig } from "./types.js";

/**
 * Core Report form configuration
 */
export const reportFormConfig: FormConfig<
    ReportShape,
    ReportCreateInput,
    ReportUpdateInput,
    Report
> = {
    objectType: "Report",

    validation: {
        schema: reportValidation,
        // Report doesn't have translation validation
        translationSchema: undefined,
    },

    transformations: {
        /** Convert form data to shape - for Report, form data is already in shape format */
        formToShape: (formData: ReportShape) => formData,

        /** Shape transformation object - use the shape model directly */
        shapeToInput: shapeReport,

        /** Convert API response back to shape format */
        apiResultToShape: (apiResult: Report): ReportShape => ({
            __typename: "Report" as const,
            id: apiResult.id,
            reason: apiResult.reason,
            otherReason: null, // Not present in API, only used for form UI
            details: apiResult.details ?? "",
            language: apiResult.language,
            createdFor: {
                __typename: apiResult.createdFor,
                id: "", // ID needs to be provided from context as it's not in the API response
            },
        }),

        /** Generate initial values for the form */
        getInitialValues: (
            session?: Session, 
            existing?: Partial<ReportShape>,
            createdFor?: { __typename: ReportFor, id: string },
        ): ReportShape => {
            const defaultLanguage = session?.users?.[0]?.languages?.[0] ?? "en";
            const defaultCreatedFor: ReportShape["createdFor"] = createdFor ?? { 
                __typename: "User" as ReportFor, 
                id: "", 
            };

            return {
                __typename: "Report" as const,
                id: existing?.id ?? DUMMY_ID,
                reason: existing?.reason ?? "",
                otherReason: existing?.otherReason ?? null,
                details: existing?.details ?? "",
                language: existing?.language ?? defaultLanguage,
                createdFor: existing?.createdFor ?? defaultCreatedFor,
            };
        },
    },

    endpoints: endpointsReport,
};
