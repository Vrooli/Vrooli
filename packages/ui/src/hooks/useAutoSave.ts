import { FormikProps } from "formik";
import { RefObject, useCallback, useEffect } from "react";

type UseAutoSaveProps = {
    disabled?: boolean;
    formikRef: RefObject<FormikProps<object>>;
    handleSave: () => unknown;
}

const AUTO_SAVE_INTERVAL_MS = 10000;

export function useAutoSave({
    disabled,
    formikRef,
    handleSave,
}: UseAutoSaveProps) {
    const maybeSave = useCallback(function maybeSaveCallback() {
        if (disabled !== true && formikRef.current && formikRef.current.dirty && !formikRef.current.isSubmitting) {
            handleSave();
        }
    }, [disabled, formikRef, handleSave]);

    useEffect(function maybeAutoSaveEffect() {
        const intervalId = setInterval(maybeSave, AUTO_SAVE_INTERVAL_MS);
        return () => {
            clearInterval(intervalId);
        };
    }, [formikRef, maybeSave]);
}
