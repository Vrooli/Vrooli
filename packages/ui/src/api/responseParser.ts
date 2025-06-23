import { DEFAULT_LANGUAGE, exists, type ServerError, type ServerErrorTranslated, type ServerErrorUntranslated, type ServerResponse, type TranslationKeyError } from "@vrooli/shared";
import i18next from "i18next";
import { ClientError } from "./ClientError.js";
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
     * Convert ServerError to ClientError for consistent error handling.
     * This bridges server and client error handling through the VrooliError interface.
     */
    static serverErrorToClientError(serverError: ServerError): ClientError {
        return ClientError.fromServerError(serverError);
    }

    /**
     * Validate that errors are compatible with parser.
     * Ensures errors can be properly processed by the response parser.
     */
    static validateErrorCompatibility(errors: ServerError[]): boolean {
        return errors.every(error => {
            try {
                const clientError = this.serverErrorToClientError(error);
                return clientError.isCompatibleWithParser();
            } catch {
                return false;
            }
        });
    }

    /**
     * Enhanced error processing with VrooliError interface support.
     * Processes server errors into structured client errors with messages and codes.
     */
    static processErrors(response: ServerResponse, languages: string[]): {
        clientErrors: ClientError[];
        messages: string[];
        codes: TranslationKeyError[];
    } {
        if (!response.errors) {
            return { clientErrors: [], messages: [], codes: [] };
        }

        const clientErrors = response.errors.map(error => this.serverErrorToClientError(error));
        const messages = response.errors.map(error => this.errorToMessage({ errors: [error] }, languages));
        const codes = response.errors.map(error => {
            if (this.isTranslatedError(error)) return error.code;
            return DEFAULT_ERROR_CODE;
        });

        return { clientErrors, messages, codes };
    }

    /**
     * Enhanced displayErrors with severity support and VrooliError integration.
     * Displays errors as snack messages with appropriate severity levels.
     */
    static displayErrorsEnhanced(errors?: ServerResponse["errors"], languages: string[] = [DEFAULT_LANGUAGE]) {
        if (!errors) return;
        
        const { clientErrors, messages } = this.processErrors({ errors }, languages);
        
        clientErrors.forEach((clientError, index) => {
            const message = messages[index];
            const severity = clientError.getSeverity();
            PubSub.get().publish("snack", { message, severity });
        });
    }

    /**
     * Displays errors from a server response as snack messages.
     * Uses `errorToMessage` to convert either a translated code or an untranslated message.
     * @param errors The errors to display
     * @deprecated Use displayErrorsEnhanced for better error handling
     */
    static displayErrors(errors?: ServerResponse["errors"]) {
        if (!errors) return;
        for (const error of errors) {
            const message = this.errorToMessage({ errors: [error] }, [DEFAULT_LANGUAGE]); // TODO provide user's languages
            PubSub.get().publish("snack", { message, severity: "Error" });
        }
    }
}
