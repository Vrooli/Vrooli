/* eslint-disable no-magic-numbers */
/**
 * Shared fields are defined to reduce bugs that may occur when 
 * there is a mismatch between the database and schemas. Every database 
 * field with a duplicate name has the name format, so as long as 
 * that format matches the fields below, there should be no errors.
 */
import * as yup from "yup";
import "./yupAugmentations.js"; // Import yup augmentations first
import { ReportFor } from "../../api/types.js";
import { validatePublicId } from "../../id/publicId.js";
import { validatePK } from "../../id/snowflake.js";
import { urlRegexDev } from "../../validation/utils/regex.js";
import { type YupMutateParams } from "../../validation/utils/types.js";
import { enumToYup } from "./builders/convert.js";
import { maxNumErr, maxStrErr, minNumErr, minStrErr, reqErr } from "./errors.js";
import { handleRegex, hexColorRegex, urlRegex } from "./regex.js";
import { meetsMinVersion } from "./versions.js";
import {
    BIO_MAX_LENGTH,
    DESCRIPTION_MAX_LENGTH,
    EMAIL_MAX_LENGTH,
    HANDLE_MAX_LENGTH,
    HANDLE_MIN_LENGTH,
    HELP_TEXT_MAX_LENGTH,
    HEX_COLOR_MAX_LENGTH,
    HEX_COLOR_MIN_LENGTH,
    IMAGE_FILE_MAX_LENGTH,
    LANGUAGE_CODE_MAX_LENGTH,
    LANGUAGE_CODE_MIN_LENGTH,
    NAME_MAX_LENGTH,
    NAME_MIN_LENGTH,
    PUSH_NOTIFICATION_KEY_MAX_LENGTH,
    REFERENCING_MAX_LENGTH,
    TAG_MAX_LENGTH,
    TAG_MIN_LENGTH,
    TIMEZONE_FIELD_MAX_LENGTH,
    URL_MAX_LENGTH,
    VERSION_LABEL_MAX_LENGTH,
    VERSION_NOTES_MAX_LENGTH,
} from "./validationConstants.js";

/**
 * Test for minimum version
 */
export function minVersionTest(minVersion: string): [string, string, (value: string | undefined) => boolean] {
    const versionRegex = /^\d+\.\d+\.\d+$/;
    return [
        "version",
        `Minimum version is ${minVersion}`,
        (value: string | undefined) => {
            if (!value) return true;
            return versionRegex.test(value) && meetsMinVersion(value, minVersion);
        },
    ];
}

// db fields
export const id = yup
    .string()
    .transform((value, originalValue) => {
        if (typeof originalValue === "bigint") {
            return originalValue.toString();
        }
        // If it's already a string (or something yup.string() can handle), pass it through
        return value;
    })
    .trim()
    .removeEmptyString() // Use the custom method as it's chained on a StringSchema
    .test(
        "id-validation",
        "Must be a valid ID (Snowflake ID)",
        (value) => value === null || value === undefined || validatePK(value as string),
    );

// Add publicId validation
export const publicId = yup.string().trim().removeEmptyString().test(
    "publicId-validation",
    "Must be a valid public ID (10-12 character alphanumeric string)",
    (value) => value === null || value === undefined || (typeof value === "string" && validatePublicId(value)),
);

// protocol fields
export const email = yup.string().trim().removeEmptyString().email("Please enter a valid email address").max(EMAIL_MAX_LENGTH, maxStrErr);
export const handle = yup.string().trim().removeEmptyString().min(HANDLE_MIN_LENGTH, minStrErr).max(HANDLE_MAX_LENGTH, maxStrErr).matches(handleRegex, "Must be 3-16 characters, and can only contain letters, numbers, and underscores");
export const hexColor = yup.string().trim().removeEmptyString().min(HEX_COLOR_MIN_LENGTH, minStrErr).max(HEX_COLOR_MAX_LENGTH, maxStrErr).matches(hexColorRegex, "Must be a valid hex color");
export const imageFile = yup.string().trim().removeEmptyString().max(IMAGE_FILE_MAX_LENGTH, maxStrErr);
export const pushNotificationKeys = yup.object({
    p256dh: yup.string().trim().removeEmptyString().max(PUSH_NOTIFICATION_KEY_MAX_LENGTH, maxStrErr).required(reqErr),
    auth: yup.string().trim().removeEmptyString().max(PUSH_NOTIFICATION_KEY_MAX_LENGTH, maxStrErr).required(reqErr),
}).default(undefined).nullable();
export function url({ env = "production" }: { env?: YupMutateParams["env"] }) {
    return yup.string().trim().removeEmptyString().max(URL_MAX_LENGTH, maxStrErr).test(
        "link",
        "Must be a URL",
        (value: string | undefined) => {
            return value !== undefined ?
                !env.startsWith("dev") ?
                    urlRegex.test(value) :
                    urlRegexDev.test(value) :
                true;
        },
    );
}

export const bool = yup.boolean().toBool();

export const bigIntString = yup
    .mixed()
    .transform(function transformBigIntString(value, originalValue) {
        // Allow null/undefined to pass through untransformed.
        if (originalValue == null) {
            return originalValue;
        }
        if (typeof originalValue === "string") {
            return originalValue.trim();
        }
        if (typeof originalValue === "number" && !Number.isNaN(originalValue)) {
            return originalValue.toString();
        }
        if (typeof originalValue === "bigint") {
            return originalValue.toString();
        }
        return originalValue;
    })
    .test("is-bigint-string", "Must be a valid integer", function testBigIntString(value) {
        // Allow null/undefined if the field is optional.
        if (value == null) return true;
        if (typeof value !== "string") {
            return this.createError({ message: "Must be a valid integer" });
        }
        return /^-?\d+$/.test(value) || this.createError({ message: "Must be a valid integer" });
    });

// numbers
export const MAX_INT = 2 ** 32 - 1;
export const MAX_DOUBLE = 2 ** 32 - 1;
export const MIN_INT = -(2 ** 32);
export const MIN_DOUBLE = -(2 ** 32);
export const doublePositiveOrZero = yup.number().min(0, minNumErr).max(MAX_DOUBLE, maxNumErr);
export const intPositiveOrZero = yup.number().min(0, minNumErr).max(MAX_INT, maxNumErr).integer();
export const intPositiveOrOne = yup.number().min(1, minNumErr).max(MAX_INT, maxNumErr).integer();
export const double = yup.number().min(MIN_DOUBLE, minNumErr).max(MAX_DOUBLE, maxNumErr);
export const int = yup.number().min(MIN_INT, minNumErr).max(MAX_INT, maxNumErr).integer();
export const index = intPositiveOrZero;

// dates
export const timezone = yup.string().trim().removeEmptyString().max(TIMEZONE_FIELD_MAX_LENGTH, maxStrErr);
export const startTime = yup.date();
export const endTime = yup.date()
    .test(
        "is-greater",
        "End time must be at least a second after start time",
        function testEndTime(value) {
            const startTime = this.resolve(yup.ref("startTime"));

            if (startTime === null || startTime === undefined) {
                return false;
            }

            if (value === null || value === undefined) {
                // req() will catch this
                return true;
            }

            // Ensure both values are Date instances
            const endDate = value instanceof Date ? value : new Date(value);
            const startDate = startTime instanceof Date ? startTime : new Date(startTime as Date | string | number);

            // Calculate the difference in milliseconds
            const diff = endDate.getTime() - startDate.getTime();

            // Check if the end time is at least a second
            return diff >= 1000;
        },
    );
export const originalStartTime = yup.date();
export const newStartTime = yup.date();
export const newEndTime = yup.date()
    .test(
        "is-greater",
        "End time must be at least a second after start time",
        function testEndTime(value) {
            const startTime = this.resolve(yup.ref("newStartTime"));

            if (startTime === null || startTime === undefined) {
                return false;
            }

            if (value === null || value === undefined) {
                // req() will catch this
                return true;
            }

            // Ensure both values are Date instances
            const endDate = value instanceof Date ? value : new Date(value);
            const startDate = startTime instanceof Date ? startTime : new Date(startTime as Date | string | number);

            // Calculate the difference in milliseconds
            const diff = endDate.getTime() - startDate.getTime();

            // Check if the end time is at least a second
            return diff >= 1000;
        },
    );
export const endDate = yup.date();

// strings
export const bio = yup.string().trim().removeEmptyString().max(BIO_MAX_LENGTH, maxStrErr);
export const description = yup.string().trim().removeEmptyString().max(DESCRIPTION_MAX_LENGTH, maxStrErr);
export const helpText = yup.string().trim().removeEmptyString().max(HELP_TEXT_MAX_LENGTH, maxStrErr);
export const referencing = yup.string().trim().removeEmptyString().max(REFERENCING_MAX_LENGTH, maxStrErr);
export const language = yup.string().trim().removeEmptyString().min(LANGUAGE_CODE_MIN_LENGTH, minStrErr).max(LANGUAGE_CODE_MAX_LENGTH, maxStrErr); // Language code
export const name = yup.string().trim().removeEmptyString().min(NAME_MIN_LENGTH, minStrErr).max(NAME_MAX_LENGTH, maxStrErr);
export const tag = yup.string().trim().removeEmptyString().min(TAG_MIN_LENGTH, minStrErr).max(TAG_MAX_LENGTH, maxStrErr);
export function versionLabel({ minVersion = "0.0.1" }: { minVersion?: string }) {
    return yup.string().trim().removeEmptyString().max(VERSION_LABEL_MAX_LENGTH, maxStrErr).test(...minVersionTest(minVersion));
}
export const versionNotes = yup.string().trim().removeEmptyString().max(VERSION_NOTES_MAX_LENGTH, maxStrErr);
export const idArray = yup.array().of(id.required(reqErr));
export const tagArray = yup.array().of(tag.required(reqErr));
export const nodeCondition = yup.string().trim().removeEmptyString().max(8192, maxStrErr);
export const nodeOperation = yup.string().trim().removeEmptyString().max(512, maxStrErr);
export const permissions = yup.string().trim().removeEmptyString().max(8192, maxStrErr);
export const response = yup.string().trim().removeEmptyString().max(8192, maxStrErr);
export const message = yup.string().trim().removeEmptyString().max(4096, maxStrErr);
export const theme = yup.string().trim().removeEmptyString().max(128, maxStrErr);
export const password = yup.string().trim().min(8).max(256);
export const details = yup.string().trim().removeEmptyString().max(8192, maxStrErr);
export const summary = yup.string().trim().removeEmptyString().max(1024, maxStrErr);
export const reportReason = yup.string().trim().removeEmptyString().min(1, minStrErr).max(128, maxStrErr);
export const instructions = yup.string().trim().removeEmptyString().max(8192, maxStrErr);
export const jsonVariable = yup.string().trim().removeEmptyString().max(8192, maxStrErr);
export const phoneNumber = yup.string().trim().removeEmptyString().max(16, maxStrErr);
export const config = yup.object();

// enums
export function reportCreatedFor() {
    return enumToYup(ReportFor);
}
