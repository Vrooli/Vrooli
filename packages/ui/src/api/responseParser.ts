import { DEFAULT_LANGUAGE, ServerError, ServerErrorTranslated, ServerErrorUntranslated, ServerResponse, TranslationKeyError, exists } from "@local/shared";
import i18next from "i18next";
import { PubSub } from "../utils/pubsub.js";

const DEFAULT_ERROR_CODE = "ErrorUnknown";
const DEFAULT_ERROR_MESSAGE = "Unknown error occurred.";

export class ServerResponseParser {
    /**
     * Type guard to determine if an error is a translated error.
     */
    static isTranslatedError(error: ServerError): error is ServerErrorTranslated {
        return "code" in error;
    }

    /**
     * Type guard to determine if an error is an untranslated error.
     */
    static isUntranslatedError(error: ServerError): error is ServerErrorUntranslated {
        return "message" in error;
    }

    /**
     * Finds the error code in a ServerResponse object.
     * @param response A response from the server
     * @returns The first error code, which should be a translation key, or DEFAULT_ERROR_CODE if not found
     */
    static errorToCode(response: ServerResponse): TranslationKeyError {
        if (response.errors) {
            for (const error of response.errors) {
                if (this.isTranslatedError(error)) {
                    return error.code;
                }
            }
        }
        // If no translated error code found, return default
        return DEFAULT_ERROR_CODE;
    }

    /**
     * Finds the error message in a ServerResponse object.
     * If there's an untranslated error (with a message), return that.
     * Otherwise, if there's a translated error (with a code), attempt to translate it.
     * @param response A response from the server
     * @param languages User's languages
     * @returns The first available error message, or a translated code if no direct message found.
     */
    static errorToMessage(response: ServerResponse, languages: string[]): string {
        const lng = languages.length > 0 ? languages[0] : DEFAULT_LANGUAGE;

        if (response.errors) {
            // Check for untranslated errors first (they have a direct message)
            for (const error of response.errors) {
                if (this.isUntranslatedError(error)) {
                    return error.message;
                }
            }

            // If no untranslated error is found, look for translated errors
            const code = this.errorToCode(response);
            return i18next.t(code, { lng, defaultValue: DEFAULT_ERROR_MESSAGE }) as unknown as string;
        }

        // If no errors array, translate a generic unknown error
        return i18next.t(DEFAULT_ERROR_CODE, { lng, defaultValue: DEFAULT_ERROR_MESSAGE }) as unknown as string;
    }

    /**
     * Checks if a specific error code or codes exist in the response.
     * @param response The response to check
     * @param codes The error code or codes to check for
     * @returns True if one of the error codes is in the error, false otherwise
     */
    static hasErrorCode(response: ServerResponse, ...codes: TranslationKeyError[]): boolean {
        if (!exists(response) || !exists(response.errors)) return false;
        return response.errors.some((error) => {
            return this.isTranslatedError(error) && codes.includes(error.code);
        });
    }

    /**
     * Displays errors from a server response as snack messages.
     * Uses `errorToMessage` to convert either a translated code or an untranslated message.
     * @param errors The errors to display
     */
    static displayErrors(errors?: ServerResponse["errors"]) {
        if (!errors) return;
        for (const error of errors) {
            const message = this.errorToMessage({ errors: [error] }, [DEFAULT_LANGUAGE]); // TODO provide user's languages
            PubSub.get().publish("snack", { message, severity: "Error" });
        }
    }
}
