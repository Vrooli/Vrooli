import { DUMMY_ID, GqlModelType } from "@local/shared";
import { useCallback, useEffect } from "react";
import { getCookieAllowFormCache, removeCookieAllowFormCache, setCookieFormData } from "utils/localStorage";
import { useDebounce } from "./useDebounce";

const DEFAULT_DEBOUNCE_TIME_MS = 200;

function shouldValuesBeSaved(values: object) {
    return Object.keys(values).length > 0;
}

/** Caches updates to the `values` prop in localStorage*/
export function useSaveToCache<T extends object>({
    values,
    objectId,
    debounceTime,
    objectType,
    isCacheOn,
    isCreate,
    disabled = false,
}: {
    values: T;
    objectId: string;
    debounceTime?: number;
    objectType: GqlModelType | `${GqlModelType}`;
    isCacheOn?: boolean;
    isCreate: boolean;
    disabled?: boolean;
}) {
    // Define the function that will handle the cache saving logic
    const saveToCache = useCallback((currentValues: T) => {
        const isCacheAllowed = getCookieAllowFormCache(objectType, objectId);
        if (disabled || isCacheOn === false || !isCacheAllowed) return;
        const formId = isCreate ? DUMMY_ID : objectId;
        setCookieFormData(objectType, formId, currentValues);
    }, [disabled, isCacheOn, isCreate, objectId, objectType]);

    const [saveToCacheDebounced] = useDebounce(saveToCache, debounceTime ?? DEFAULT_DEBOUNCE_TIME_MS);

    // Effect to handle saving values to cache whenever they change
    useEffect(() => {
        // Check if values are not empty or match some condition to be saved
        if (shouldValuesBeSaved(values)) {
            saveToCacheDebounced(values);
        }
        // Cleanup function to save the cache when the component unmounts or values change
        return () => {
            if (shouldValuesBeSaved(values)) {
                saveToCache(values);
            }
        };
    }, [saveToCache, saveToCacheDebounced, values]);

    // On initial load, reset localStorage cache flag
    useEffect(() => {
        removeCookieAllowFormCache(objectType, objectId);
    }, [objectId, objectType]);
}
