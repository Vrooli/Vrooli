import { DUMMY_ID, GqlModelType } from "@local/shared";
import { useCallback, useEffect } from "react";
import { getCookieAllowFormCache, removeCookieAllowFormCache, setCookieFormData } from "utils/cookies";
import { useDebounce } from "./useDebounce";

const shouldValuesBeSaved = (values: object) => {
    return Object.keys(values).length > 0;
};

/** Caches updates to the `values` prop in localStorage*/
export const useSaveToCache = <T extends object>({
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
}) => {
    // Create a unique cache key using the objectType and objectId (if it exists)
    const formCacheId = isCreate ? `${objectType}-${DUMMY_ID}` : `${objectType}-${objectId}`;

    // Define the function that will handle the cache saving logic
    const saveToCache = useCallback((currentValues: T) => {
        const isCacheAllowed = getCookieAllowFormCache(objectType, objectId);
        if (disabled || isCacheOn === false || !isCacheAllowed) return;
        setCookieFormData(formCacheId, currentValues);
    }, [disabled, formCacheId, isCacheOn, objectId, objectType]);

    const [saveToCacheDebounced] = useDebounce(saveToCache, debounceTime ?? 200);

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
};
