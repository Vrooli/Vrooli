import { type FormikProps } from "formik";
import { type RefObject, useCallback, useEffect, useRef } from "react";
import { useDebounce } from "./useDebounce.js";

type UseAutoSaveProps<T = object> = {
    disabled?: boolean;
    formikRef: RefObject<FormikProps<T>>;
    handleSave: () => unknown;
    /** Delay in ms before saving after changes stop. Default: 2000 (2 seconds) */
    debounceMs?: number;
}

const DEFAULT_DEBOUNCE_MS = 2000;
const CHECK_INTERVAL_MS = 250;

export function useAutoSave<T = object>({
    disabled,
    formikRef,
    handleSave,
    debounceMs = DEFAULT_DEBOUNCE_MS,
}: UseAutoSaveProps<T>) {
    // Keep track of the previous values to detect changes
    const previousValuesRef = useRef<T | null>(null);
    // Track whether form is dirty
    const isDirtyRef = useRef(false);
    // Track last saved values to detect new changes after save
    const lastSavedValuesRef = useRef<T | null>(null);

    // Create a save function that only runs if we should save
    const maybeSave = useCallback(function maybeSaveCallback() {
        if (
            disabled !== true &&
            formikRef.current &&
            formikRef.current.dirty &&
            !formikRef.current.isSubmitting
        ) {
            // Store the values we're about to save
            lastSavedValuesRef.current = { ...formikRef.current.values };

            // Call the save handler
            handleSave();
        }
    }, [disabled, formikRef, handleSave]);

    // Create debounced save function
    const [debouncedSave, cancelDebounce] = useDebounce<void>(maybeSave, debounceMs);

    // Check if values have changed since last save
    const hasUnsavedChanges = useCallback((currentValues: T) => {
        // If we've never saved, any dirty state means unsaved changes
        if (!lastSavedValuesRef.current) {
            return formikRef.current?.dirty ?? false;
        }

        // Check if current values differ from last saved values
        return JSON.stringify(lastSavedValuesRef.current) !== JSON.stringify(currentValues);
    }, [formikRef]);

    // Watch for changes in form values
    useEffect(function checkFormValuesEffect() {
        // Check for changes when formik values update
        function checkForChanges() {
            const formik = formikRef.current;
            if (!formik) return;

            const currentValues = formik.values;

            // Update dirty status for window events - consider both Formik's dirty flag
            // and our own comparison with last saved values
            const isCurrentlyDirty = formik.dirty || hasUnsavedChanges(currentValues);
            isDirtyRef.current = isCurrentlyDirty;

            // Detect if values have changed since last check
            const hasChanged =
                !previousValuesRef.current ||
                JSON.stringify(previousValuesRef.current) !== JSON.stringify(currentValues);

            // If values have changed and there are unsaved changes, trigger debounced save
            if (hasChanged && isCurrentlyDirty && !formik.isSubmitting) {
                previousValuesRef.current = { ...currentValues };
                debouncedSave();
            }
        }

        // Set up interval to periodically check for changes
        const intervalId = setInterval(checkForChanges, CHECK_INTERVAL_MS);

        return () => {
            clearInterval(intervalId);
        };
    }, [formikRef, debouncedSave, hasUnsavedChanges]);

    // Add window beforeunload event to save on refresh/navigation
    useEffect(() => {
        function handleBeforeUnload(event: BeforeUnloadEvent) {
            const formik = formikRef.current;

            // Check if there are unsaved changes
            if (isDirtyRef.current && !disabled) {
                // Cancel debounced save
                cancelDebounce();

                if (formik && !formik.isSubmitting) {
                    // Check if values have changed since last save
                    const hasChanges = hasUnsavedChanges(formik.values);

                    if (hasChanges) {
                        // Use synchronous save for beforeunload
                        handleSave();

                        // Modern browsers no longer show this message, but we set it for older browsers
                        event.preventDefault();
                        event.returnValue = "";
                        return "";
                    }
                }
            }
        }

        window.addEventListener("beforeunload", handleBeforeUnload);

        return () => {
            window.removeEventListener("beforeunload", handleBeforeUnload);
        };
    }, [formikRef, handleSave, cancelDebounce, disabled, hasUnsavedChanges]);

    // Save on component unmount
    useEffect(() => {
        return () => {
            const formik = formikRef.current;
            if (
                disabled !== true &&
                formik &&
                formik.dirty &&
                !formik.isSubmitting &&
                hasUnsavedChanges(formik.values)
            ) {
                // Cancel any pending debounced save
                cancelDebounce();
                // Immediately save
                handleSave();
            }
        };
    }, [disabled, formikRef, handleSave, cancelDebounce, hasUnsavedChanges]);
}
