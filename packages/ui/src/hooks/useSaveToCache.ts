import { DUMMY_ID, GqlModelType } from "@local/shared";
import { MutableRefObject, useCallback, useEffect } from "react";
import { setCookieFormData } from "utils/cookies";
import { useDebounce } from "./useDebounce";

// Helper function to determine if the values should be saved.
// Implement this with the logic specific to your application needs.
function shouldValuesBeSaved(values) {
    // For example, you might check that the values are not empty
    return Object.keys(values).length > 0;
}

/** Caches updates to the `values` prop in localStorage*/
export const useSaveToCache = <T>({
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
    isCacheOn: MutableRefObject<boolean>;
    isCreate: boolean;
    disabled?: boolean;
}) => {
    // Create a unique cache key using the objectType and objectId (if it exists)
    const formCacheId = isCreate ? `${objectType}-${DUMMY_ID}` : `${objectType}-${objectId}`;

    // Define the function that will handle the cache saving logic
    const saveToCache = useCallback((currentValues: T) => {
        if (disabled || !isCacheOn.current) return;
        setCookieFormData(formCacheId, currentValues as any);
    }, [disabled, formCacheId, isCacheOn]);

    const [saveToCacheDebounced] = useDebounce(saveToCache, debounceTime ?? 200);

    // Effect to handle saving values to cache whenever they change
    useEffect(() => {
        // Copy the ref value to a variable inside the effect
        const cacheOn = isCacheOn.current;

        // Check if values are not empty or match some condition to be saved
        if (!disabled && cacheOn && shouldValuesBeSaved(values)) {
            saveToCacheDebounced(values);
        }

        // Cleanup function to save the cache when the component unmounts or values change
        return () => {
            if (!disabled && cacheOn && shouldValuesBeSaved(values)) {
                saveToCache(values);
            }
        };
    }, [disabled, values, saveToCacheDebounced, saveToCache, isCacheOn]);
};
