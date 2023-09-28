/**
 * Shared fields are defined to reduce bugs that may occur when 
 * there is a mismatch between the database and schemas. Every database 
 * field with a duplicate name has the name format, so as long as 
 * that format matches the fields below, there should be no errors.
 */
import { ReportFor } from "@local/shared";
import * as yup from "yup";
import { enumToYup } from "./builders";
import { toPosInt } from "./builders/toPosInt";
import { maxNumErr, maxStrErr, minNumErr, minStrErr, reqErr } from "./errors";
import { handleRegex, urlRegex } from "./regex";
import { minVersionTest } from "./versions";


yup.addMethod(yup.string, "removeEmptyString", function () {
    return this.when([], (schema, { value }: any) => {
        if (typeof value !== "string" || value === "") return schema.strip();
        return schema;
    });
});


// db fields
export const id = yup.string().trim().removeEmptyString().max(256, maxStrErr);

// protocol fields
export const apiCallData = yup.string().trim().removeEmptyString().max(8192, maxStrErr);
export const email = yup.string().trim().removeEmptyString().email().max(256, maxStrErr);
export const handle = yup.string().trim().removeEmptyString().min(3, minStrErr).max(16, maxStrErr).test(
    "handle",
    "Must be 3-16 characters, and only contain letters and numbers",
    (value: string | undefined) => {
        return value !== undefined ? handleRegex.test(value) : true;
    },
);
export const hexColor = yup.string().trim().removeEmptyString().max(16, maxStrErr);
export const imageFile = yup.string().trim().removeEmptyString();
export const pushNotificationKeys = yup.object().shape({
    p256dh: yup.string().trim().removeEmptyString().max(256, maxStrErr),
    auth: yup.string().trim().removeEmptyString().max(256, maxStrErr),
});
export const smartContractCallData = yup.string().trim().removeEmptyString().max(8192, maxStrErr);
export const url = yup.string().trim().removeEmptyString().max(1024, maxStrErr).test(
    "link",
    "Must be a URL",
    (value: string | undefined) => {
        return value !== undefined ? urlRegex.test(value) : true;
    },
);

export const bool = yup.boolean();

// numbers
const MAX_INT = 2 ** 32 - 1;
const MAX_DOUBLE = 2 ** 32 - 1;
const MIN_INT = -(2 ** 32);
const MIN_DOUBLE = -(2 ** 32);
export const doublePositiveOrZero = yup.number().min(0, minNumErr).max(MAX_DOUBLE, maxNumErr);
export const intPositiveOrZero = yup.number().min(0, minNumErr).max(MAX_INT, maxNumErr).integer();
export const intPositiveOrOne = yup.number().min(1, minNumErr).max(MAX_INT, maxNumErr).integer();
export const double = yup.number().min(MIN_DOUBLE, minNumErr).max(MAX_DOUBLE, maxNumErr);
export const int = yup.number().min(MIN_INT, minNumErr).max(MAX_INT, maxNumErr).integer();
export const index = intPositiveOrZero;

// dates
export const timezone = yup.string().trim().removeEmptyString().max(64, maxStrErr);
export const startTime = yup.date();
export const endTime = yup.date().min(yup.ref("startTime"), "End time must be after start date");
export const originalStartTime = yup.date();
export const newStartTime = yup.date();
export const newEndTime = yup.date().min(yup.ref("newStartTime"), "End time must be after start date");
export const endDate = yup.date();

// strings
export const bio = yup.string().trim().removeEmptyString().max(2048, maxStrErr);
export const description = yup.string().trim().removeEmptyString().max(2048, maxStrErr);
export const helpText = yup.string().trim().removeEmptyString().max(2048, maxStrErr);
export const referencing = yup.string().trim().removeEmptyString().max(2048, maxStrErr);
export const language = yup.string().trim().removeEmptyString().min(2, minStrErr).max(3, maxStrErr); // Language code
export const name = yup.string().trim().removeEmptyString().min(3, minStrErr).max(50, maxStrErr);
export const tag = yup.string().trim().removeEmptyString().min(2, minStrErr).max(64, maxStrErr);
export const versionLabel = (minVersion = "0.0.1") => yup.string().trim().removeEmptyString().max(16, maxStrErr).test(...minVersionTest(minVersion));
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
export const phoneNumber = yup.string().trim().removeEmptyString().transform(toPosInt).max(16, maxStrErr);

// enums
export const reportCreatedFor = enumToYup(ReportFor);
