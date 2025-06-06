/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in chatMessage.test.ts
import * as yup from "yup";
import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id as commonIdSchema, id, intPositiveOrZero } from "../utils/commonFields.js";
import { maxStrErr, minStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";
import { chatValidation } from "./chat.js";

const toolFunctionCallResultSchema = yup.object()
    .shape({
        success: yup.boolean().required("Success flag is required for tool result"),
        output: yup.mixed(), // Conditionally validated by the test below
        error: opt(yup.object({ // Error object is optional; if present, its fields are required
            code: req(yup.string()),
            message: req(yup.string()),
        })),
    })
    .test({
        name: "tool-result-consistency",
        exclusive: true,
        message: "Tool result must be consistent: if 'success' is true, 'output' must be defined and 'error' must be undefined. If 'success' is false, 'error' (with code and message) must be defined and 'output' must be undefined.",
        test: (value: { success: boolean; output?: any; error?: { code?: string; message?: string } } | undefined | null) => {
            if (value == null) {
                // This allows the entire 'result' field (which uses this schema) to be optional.
                return true;
            }
            const { success, output, error } = value;
            // The 'success' field itself is required by yup.boolean().required() above.
            // This test ensures the dependent fields 'output' and 'error' are consistent with 'success'.
            if (typeof success !== "boolean") {
                // Should be caught by yup.boolean().required(), but good for robustness in a complex test.
                return false;
            }

            if (success) {
                // If success is true, output must be present, and error must not.
                return output !== undefined && error === undefined;
            } else { // success is false
                // If success is false, error must be present (and valid), and output must not.
                // The req() builders for error.code and error.message handle their presence if error object is defined.
                return error !== undefined && typeof error.code === "string" && typeof error.message === "string" && output === undefined;
            }
        },
    });

const toolFunctionCallSchema = yup.object({
    id: req(commonIdSchema),
    function: req(yup.object({
        name: req(yup.string()),
        arguments: req(yup.string()),
    })),
    result: opt(toolFunctionCallResultSchema),
});

export const messageConfigObjectValidationSchema = yup.object({
    __version: req(yup.string()),
    resources: req(yup.array().of(yup.mixed())), // Represents BaseResource[], using mixed for now.
    contextHints: opt(yup.array().of(req(yup.string()))),
    eventTopic: opt(yup.string()),
    respondingBots: opt(yup.array().of(req(yup.string()))), // Handles strings like "@all"
    role: opt(yup.string().oneOf(["user", "assistant", "system", "tool"])),
    turnId: opt(yup.number().integer().nullable()),
    toolCalls: opt(yup.array().of(req(toolFunctionCallSchema))), // Each ToolFunctionCall in the array is required
});

const MAX_CHAT_MESSAGE_TEXT_LENGTH = 32768;
const text = yup.string().trim().removeEmptyString().min(1, minStrErr).max(MAX_CHAT_MESSAGE_TEXT_LENGTH, maxStrErr);

export const chatMessageTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        text: req(text),
    }),
    update: () => ({
        text: opt(text),
    }),
});

export const chatMessageValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        versionIndex: opt(intPositiveOrZero),
        config: req(messageConfigObjectValidationSchema),
    }, [
        ["chat", ["Connect"], "one", "req", chatValidation, ["messages"]],
        ["user", ["Connect"], "one", "opt"],
        ["translations", ["Create"], "many", "opt", chatMessageTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        config: opt(messageConfigObjectValidationSchema),
    }, [
        ["translations", ["Create", "Update", "Delete"], "many", "opt", chatMessageTranslationValidation],
    ], [], d),
};
/* c8 ignore stop */
