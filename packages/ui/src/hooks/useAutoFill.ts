import { AutoFillInput, AutoFillResult, LlmTask, endpointGetAutoFill } from "@local/shared";
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
