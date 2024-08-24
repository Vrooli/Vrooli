import { AutoFillInput, AutoFillResult, DUMMY_ID, LlmTask, endpointGetAutoFill, getTranslation } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { useCallback } from "react";
import { PubSub } from "utils/pubsub";
import { useLazyFetch } from "./useLazyFetch";

type UseAutoFillProps<FormShape = object> = {
    /**
     * @returns The input to send to the autofill endpoint, with 
     * its shape depending on the task
     */
    getAutoFillInput: () => AutoFillInput["data"];
    /**
     * Shapes the autofill result into the form values to update 
     * the form with, while also returning a copy of the original 
     * form values for undoing the autofill
     * @param values The values returned from the autofill endpoint
     * @returns An object with the original form values and the 
     * values to update the form with
     */
    shapeAutoFillResult: (values: AutoFillResult["data"]) => { originalValues: FormShape, updatedValues: FormShape };
    /**
     * Handles updating the form
     * @param values The values to update the form with
     */
    handleUpdate: (values: FormShape) => unknown;
    task: LlmTask;
}

const LOADING_SNACK_ID = "autofill-loading";
const SUCCESS_SNACK_DURATION_MS = 15_000;
const ERROR_SNACK_DURATION_MS = 10_000;

/**
 * A basic function to clean up the input object before sending it to the server. 
 * Since the input object comes from a form (and formik requires all fields to be
 * present), we can remove fields which represent no data (i.e. empty strings).
 * @param input The input object to clean
 * @returns The cleaned input object
 */
function cleanInput(input: AutoFillInput["data"]): AutoFillInput["data"] {
    Object.entries(input).forEach(([key, value]) => {
        if (typeof value === "string" && value.trim() === "") {
            delete input[key];
        }
    });
    return input;
}

/**
 * Finds relevant autofill data from the current translation
 * @param values The current form values, where translation data is stored in 
 * a `translations` array
 * @params language The current language being edited
 * @returns The translation data to send to the autofill endpoint, notably with 
 * fields like `language` and `id` omitted
 */
export function getAutoFillTranslationData<
    Translation extends { language: string },
>(
    values: { translations?: Translation[] | null | undefined },
    language: string,
): object {
    const currentTranslation = { ...getTranslation(values, [language], true) } as Record<string, unknown>;
    delete currentTranslation.id;
    delete currentTranslation.language;
    delete currentTranslation.__typename;
    return currentTranslation;
}

/**
 * Creates updated translations based on autofill results and the current language.
 * @param values The current form values, where translation data is stored in
 * @param autofillData The data received from the autofill process, which contains translation 
 * and other data
 * @param language The current language being edited
 * @param translatedFields Fields that should be included in the updated translations
 * @returns An object containing the updated translations array and an object with the non-translated fields
 */
export function createUpdatedTranslations<
    Translation extends { language: string },
>(
    values: { translations?: Translation[] | null | undefined },
    autofillData: object,
    language: string,
    translatedFields: string[],
): { updatedTranslations: Translation[], rest: Record<string, unknown> } {
    const updatedTranslations: Translation[] = [];
    const updatedTranslationFields: Partial<Translation> = {};
    const rest: Record<string, unknown> = { ...autofillData };

    // Extract translated fields from autofill data
    translatedFields.forEach(field => {
        if (Object.prototype.hasOwnProperty.call(autofillData, field)) {
            updatedTranslationFields[field] = autofillData[field];
            delete rest[field];
        }
    });

    // We can only update translations if they existed in the first place
    if (!values.translations || !Array.isArray(values.translations) || values.translations.length === 0) {
        return { updatedTranslations, rest };
    }

    // Use the original translations to find the correct index to update
    let languageIndex = values.translations.findIndex(t => t.language === language);
    if (languageIndex < 0) {
        languageIndex = 0; // Fallback to first translation if the language doesn't exist
    }

    values.translations.forEach((translation, index) => {
        if (index === languageIndex) {
            // Merge autofill data into the current translation, excluding non-translated fields
            updatedTranslations.push({
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore This is in case the translation is missing a language
                language,
                ...translation,
                ...updatedTranslationFields,
                id: (translation as { id?: string }).id || DUMMY_ID,
            });
        } else {
            updatedTranslations.push(translation);
        }
    });

    return { updatedTranslations, rest };
}

/**
 * Hook for handling AI autofill functionality in forms
 */
export function useAutoFill<T = object>({
    getAutoFillInput,
    shapeAutoFillResult,
    handleUpdate,
    task,
}: UseAutoFillProps<T>) {
    const [getAutoFill, { loading: isAutoFillLoading }] = useLazyFetch<AutoFillInput, AutoFillResult>(endpointGetAutoFill);

    const autoFill = useCallback(function autoFillCallback() {
        let data = getAutoFillInput();
        data = cleanInput(data);

        PubSub.get().publish("snack", {
            message: "Auto-filling form...",
            id: LOADING_SNACK_ID,
            severity: "Info",
            autoHideDuration: "persist",
        });

        fetchLazyWrapper<AutoFillInput, AutoFillResult>({
            fetch: getAutoFill,
            inputs: { task, data },
            onSuccess: (result) => {
                console.log("got autofill response", result);

                const { originalValues, updatedValues } = shapeAutoFillResult(result);
                handleUpdate(updatedValues);

                PubSub.get().publish("snack", {
                    message: "Form auto-filled",
                    buttonKey: "Undo",
                    buttonClicked: () => { handleUpdate(originalValues); },
                    severity: "Success",
                    autoHideDuration: SUCCESS_SNACK_DURATION_MS,
                    id: LOADING_SNACK_ID,
                });
            },
            onError: (error) => {
                PubSub.get().publish("snack", {
                    message: "Failed to auto-fill form.",
                    severity: "Error",
                    autoHideDuration: ERROR_SNACK_DURATION_MS,
                    id: LOADING_SNACK_ID,
                    data: error,
                });
            },
            spinnerDelay: null, // Disables loading spinner
        });
    }, [getAutoFill, getAutoFillInput, handleUpdate, shapeAutoFillResult, task]);

    return {
        autoFill,
        isAutoFillLoading,
    };
}
