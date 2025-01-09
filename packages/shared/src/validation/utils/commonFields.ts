/* eslint-disable no-magic-numbers */
/**
 * Shared fields are defined to reduce bugs that may occur when 
 * there is a mismatch between the database and schemas. Every database 
 * field with a duplicate name has the name format, so as long as 
 * that format matches the fields below, there should be no errors.
 */
import * as yup from "yup";
import { ReportFor } from "../../api/types";
import { uuidValidate } from "../../id/uuid";
import { urlRegexDev } from "../../validation/utils/regex";
import { YupMutateParams } from "../../validation/utils/types";
import { enumToYup } from "./builders";
import { maxNumErr, maxStrErr, minNumErr, minStrErr, reqErr } from "./errors";
import { handleRegex, hexColorRegex, urlRegex } from "./regex";
import { meetsMinVersion } from "./versions";

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

yup.addMethod(yup.string, "removeEmptyString", function transformRemoveEmptyString() {
    return this.transform((value: unknown) => {
        return typeof value === "string" && value.trim() !== "" ? value : undefined;
    });
});

yup.addMethod(yup.bool, "toBool", function transformToBool() {
    return this.transform((value: unknown) => {
        if (typeof value === "boolean") return value;
        if (typeof value === "string") return value.trim() === "true" || value.trim() === "yes" || value.trim() === "1";
        if (typeof value === "number") return value === 1;
        return false;
    });
});


// db fields
export const id = yup.string().trim().removeEmptyString().test("uuid", "Must be a valid UUID", (value) => value === null || value === undefined || (typeof value === "string" && uuidValidate(value)));

// protocol fields
export const configCallData = yup.string().trim().removeEmptyString().max(8192, maxStrErr);
export const configFormInput = yup.string().trim().removeEmptyString().max(16384, maxStrErr);
export const configFormOutput = yup.string().trim().removeEmptyString().max(16384, maxStrErr);
export const email = yup.string().trim().removeEmptyString().email("Please enter a valid email address").max(256, maxStrErr);
export const handle = yup.string().trim().removeEmptyString().min(3, minStrErr).max(16, maxStrErr).matches(handleRegex, "Must be 3-16 characters, and can only contain letters, numbers, and underscores");
export const hexColor = yup.string().trim().removeEmptyString().min(4, minStrErr).max(7, maxStrErr).matches(hexColorRegex, "Must be a valid hex color");
export const imageFile = yup.string().trim().removeEmptyString().max(256, maxStrErr);
export const pushNotificationKeys = yup.object().shape({
    p256dh: yup.string().trim().removeEmptyString().max(256, maxStrErr).required(reqErr),
    auth: yup.string().trim().removeEmptyString().max(256, maxStrErr).required(reqErr),
});
export function url({ env = "production" }: { env?: YupMutateParams["env"] }) {
    return yup.string().trim().removeEmptyString().max(1024, maxStrErr).test(
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
export const timezone = yup.string().trim().removeEmptyString().max(64, maxStrErr);
export const startTime = yup.date();
export const endTime = yup.date()
    .test(
        "is-greater",
        "End time must be at least a second after start time",
        function testEndTime(value) {
            const startTime = this.resolve(yup.ref("startTime"));

            if (startTime === null || startTime === undefined) {
                console.log("Start time is missing");
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
                console.log("Start time is missing");
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
export const bio = yup.string().trim().removeEmptyString().max(2048, maxStrErr);
export const description = yup.string().trim().removeEmptyString().max(2048, maxStrErr);
export const helpText = yup.string().trim().removeEmptyString().max(2048, maxStrErr);
export const referencing = yup.string().trim().removeEmptyString().max(2048, maxStrErr);
export const language = yup.string().trim().removeEmptyString().min(2, minStrErr).max(3, maxStrErr); // Language code
export const name = yup.string().trim().removeEmptyString().min(3, minStrErr).max(50, maxStrErr);
export const tag = yup.string().trim().removeEmptyString().min(2, minStrErr).max(64, maxStrErr);
export function versionLabel({ minVersion = "0.0.1" }: { minVersion?: string }) {
    return yup.string().trim().removeEmptyString().max(16, maxStrErr).test(...minVersionTest(minVersion));
}
export const versionNotes = yup.string().trim().removeEmptyString().max(4092, maxStrErr);
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

// enums
export const reportCreatedFor = enumToYup(ReportFor);
