import { LINKS, ModelType, getObjectUrl, type ListObject, type NavigableObject, type OrArray, type TranslationKeyCommon } from "@vrooli/shared";
import { useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { create } from "zustand";
import { ObjectDialogAction } from "../components/dialogs/types.js";
import { useLocation } from "../route/router.js";
import { type SetLocation } from "../route/types.js";
import { type FormProps, type ViewDisplayType } from "../types.js";
import { getCookieFormData, removeCookieFormData, removeCookiePartialData, setCookieFormData, setCookiePartialData } from "../utils/localStorage.js";
import { PubSub } from "../utils/pubsub.js";
import { useDebounce } from "./useDebounce.js";

const DEFAULT_DEBOUNCE_TIME_MS = 200;

type FormCacheState = {
    cacheEnabled: boolean;
    enableCache: () => void;
    disableCache: () => void;
    getCacheData: (pathname: string) => object | null;
    setCacheData: (pathname: string, values: object) => void;
    clearCache: (pathname: string) => void;
};

export const useFormCacheStore = create<FormCacheState>()((set, get) => ({
    cacheEnabled: true,
    enableCache: () => set({ cacheEnabled: true }),
    disableCache: () => set({ cacheEnabled: false }),
    getCacheData: (pathname) => {
        if (!get().cacheEnabled) return null;
        return getCookieFormData(pathname) ?? null;
    },
    setCacheData: (pathname, values) => {
        if (!get().cacheEnabled) return;
        setCookieFormData(pathname, values);
    },
    clearCache: (pathname) => {
        removeCookieFormData(pathname);
    },
}));

function shouldValuesBeSaved(values: object) {
    return Object.keys(values).length > 0;
}

/**
 * Custom hook to cache form data updates to localStorage.
 *
 * @template T - The type of the form values object.
 * @param {object} props - The properties for the hook.
 * @param {number} [props.debounceTime=200] - The debounce time in milliseconds for saving to cache.
 * @param {boolean} [props.disabled=false] - If true, caching is disabled.
 * @param {boolean} [props.isCacheOn] - Explicitly controls whether caching is active. Overrides global cache state if false.
 * @param {boolean} props.isCreate - Indicates if the form is for creating a new entity. This can influence caching logic (e.g., not caching on create).
 * @param {string} props.pathname - The current URL pathname, used as a key for storing/retrieving cached data.
 * @param {T} props.values - The form values to be cached.
 */
export function useSaveToCache<T extends object>({
    debounceTime,
    disabled = false,
    isCacheOn,
    isCreate,
    pathname,
    values,
}: {
    debounceTime?: number;
    disabled?: boolean;
    isCacheOn?: boolean;
    isCreate: boolean;
    pathname: string;
    values: T;
}) {
    const cacheEnabled = useFormCacheStore(state => state.cacheEnabled);
    const enableCache = useFormCacheStore(state => state.enableCache);
    const setCacheData = useFormCacheStore(state => state.setCacheData);

    // Define the function that will handle the cache saving logic
    const saveToCache = useCallback((currentValues: T) => {
        if (disabled || isCacheOn === false) return;
        setCacheData(pathname, currentValues);
    }, [disabled, isCacheOn, isCreate, pathname, setCacheData]);

    const [saveToCacheDebounced] = useDebounce(saveToCache, debounceTime ?? DEFAULT_DEBOUNCE_TIME_MS);

    useEffect(function saveOnValuesChangeEffect() {
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
    }, [cacheEnabled, saveToCache, saveToCacheDebounced, values]);

    useEffect(function enableFormDataCacheEffect() {
        // On load, enable form data caching
        enableCache();
    }, [enableCache]);
}

type TType = OrArray<{ __typename: ListObject["__typename"], id: string }>;
type UseUpsertActionsProps<Model extends TType> = Pick<FormProps<Model, object>, "onCancel" | "onCompleted" | "onDeleted"> & {
    display: ViewDisplayType | `${ViewDisplayType}`,
    isCreate: boolean,
    objectType: ListObject["__typename"],
    onAction?: (action: ObjectDialogAction, item: Model) => void,
    pathname: string,
    suppressSnack?: boolean,
}

/**
 * Custom hook to manage common actions for upserting (create/update) entities.
 * It handles navigation for page-level forms, triggers callbacks for dialog-based forms,
 * and manages snackbar notifications.
 *
 * @template T - The type of the entity being upserted. Must extend `TType`.
 * @param {object} props - The properties for the hook.
 * @param {ViewDisplayType | string} props.display - The display context of the form (e.g., "Page", "Dialog").
 * @param {boolean} props.isCreate - True if the form is for creating a new entity, false if updating.
 * @param {ListObject["__typename"]} props.objectType - The type of the object being upserted (e.g., "User", "Task").
 * @param {(action: ObjectDialogAction, item: T) => void} [props.onAction] - Optional callback triggered after any action.
 * @param {() => void} [props.onCancel] - Optional callback for cancel actions, typically used in dialogs.
 * @param {(item: T) => void} [props.onCompleted] - Optional callback for successful create/update, typically used in dialogs.
 * @param {(item: T) => void} [props.onDeleted] - Optional callback for successful delete, typically used in dialogs.
 * @param {string} props.pathname - The current URL pathname, used for cache clearing and navigation.
 * @param {boolean} [props.suppressSnack] - If true, snackbar notifications will be suppressed.
 * @returns {object} An object containing handler functions:
 * @returns {() => void} handleCancel - Handles cancel action.
 * @returns {(data: T) => void} handleCreated - Handles successful creation.
 * @returns {(data: T) => void} handleDeleted - Handles successful deletion.
 * @returns {(data: T) => void} handleUpdated - Handles successful update.
 * @returns {(data: T) => void} handleCompleted - Handles successful creation or update based on `isCreate`.
 */
export function useUpsertActions<T extends TType>({
    display,
    isCreate,
    objectType,
    onAction,
    onCancel,
    onCompleted,
    onDeleted,
    pathname,
    suppressSnack,
}: UseUpsertActionsProps<T>) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const clearCache = useFormCacheStore(state => state.clearCache);
    const disableCache = useFormCacheStore(state => state.disableCache);

    /** Helper function to publish a snack message */
    const publishSnack = useCallback((actionType: "Created" | "Updated", count: number, item: NavigableObject) => {
        if (suppressSnack) return;
        const rootType = objectType.replace("Version", "");
        const objectTranslation = t(rootType, { count: 1, defaultValue: rootType });

        // Some actions have buttons
        let buttonKey: TranslationKeyCommon | undefined;
        let buttonClicked: (() => unknown) | undefined;
        // Create ations typically have a button to create a new object
        if (actionType === "Created") {
            // But for some objects, it makes more sense for the button to open the object
            const shouldOpenObject = [ModelType.Report].includes(objectType as ModelType);
            if (shouldOpenObject) {
                buttonKey = "View";
                const url = getObjectUrl(item);
                buttonClicked = () => setLocation(url);
            } else {
                buttonKey = "CreateNew";
                buttonClicked = () => setLocation(`${LINKS[rootType]}/add`);
            }
        }

        PubSub.get().publish("snack", {
            message: t(`Object${actionType}`, { objectName: objectTranslation, count }),
            severity: "Success",
            buttonKey,
            buttonClicked,
        });
    }, [objectType, setLocation, suppressSnack, t]);

    const handleAction = useCallback((action: ObjectDialogAction, item: T | null) => {
        let viewUrl: string | undefined;
        let canStore = false;
        if (item && !Array.isArray(item)) {
            viewUrl = getObjectUrl(item);
            // Should hopefully have id if not an array
            if (item.id) {
                canStore = true;
            } else {
                console.warn("No id found on object. This may be a mistake.", item);
            }
        }

        function handleAddOrUpdate(messageType: "Created" | "Updated", item: NavigableObject) {
            if (canStore) {
                setCookiePartialData(item as NavigableObject, "full"); // Update cache to view object more quickly
                clearCache(pathname);
                disableCache();
            }

            if (display === "Page") setLocation(viewUrl ?? LINKS.Home, { replace: true });

            if ((isCreate && messageType === "Created") || (!isCreate && messageType === "Updated")) {
                publishSnack(messageType, Array.isArray(item) ? item.length : 1, item);
            }
        }

        switch (action) {
            case ObjectDialogAction.Add:
                if (item && !Array.isArray(item)) {
                    handleAddOrUpdate("Created", item);
                    onCompleted?.(item);
                }
                break;
            case ObjectDialogAction.Cancel:
                // Remove form backup data from cache
                if (canStore) {
                    clearCache(pathname);
                    disableCache();
                }
                if (display === "Page") goBack(setLocation, isCreate ? undefined : viewUrl);
                else onCancel?.();
                break;
            case ObjectDialogAction.Close:
                // DO NOT remove form backup data from cache
                if (display === "Page") goBack(setLocation, isCreate ? undefined : viewUrl);
                else onCancel?.();
                break;
            case ObjectDialogAction.Delete:
                // Remove both the object and the form backup data from cache
                if (canStore) {
                    removeCookiePartialData(item as NavigableObject, pathname);
                    clearCache(pathname);
                    disableCache();
                }
                if (display === "Page") goBack(setLocation);
                else if (item) onDeleted?.(item);
                // Don't display snack message, as we don't have enough information for the message's "Undo" button
                break;
            case ObjectDialogAction.Save:
                if (item && !Array.isArray(item)) {
                    handleAddOrUpdate("Updated", item);
                    onCompleted?.(item);
                }
                break;
        }

        if (item) {
            onAction?.(action, item);
        }
    }, [onAction, display, isCreate, clearCache, objectType, disableCache, setLocation, publishSnack, onCompleted, onCancel, onDeleted]);

    const handleCancel = useCallback(() => handleAction(ObjectDialogAction.Cancel, null), [handleAction]);
    const handleCreated = useCallback((data: T) => { handleAction(ObjectDialogAction.Add, data); }, [handleAction]);
    const handleUpdated = useCallback((data: T) => { handleAction(ObjectDialogAction.Save, data); }, [handleAction]);
    const handleCompleted = useCallback((data: T) => { handleAction(isCreate ? ObjectDialogAction.Add : ObjectDialogAction.Save, data); }, [isCreate, handleAction]);
    const handleDeleted = useCallback((data: T) => { handleAction(ObjectDialogAction.Delete, data); }, [handleAction]);

    return {
        handleCancel,
        handleCreated,
        handleDeleted,
        handleUpdated,
        handleCompleted,
    };
}

/**
 * Navigates to the previous page, with recovery logic if the previous page 
 * is not this app (i.e. you loaded the current page directly).
 * @param setLocation Callback to set the location
 * @param targetUrl What the previous page should be. If the previous page does 
 * not match this, the user will be navigated to the targetUrl instead.
 */
export function goBack(setLocation: SetLocation, targetUrl?: string) {
    const lastPath = sessionStorage.getItem("lastPath");
    // Check if last path is the same as the target URL, excluding query params
    const lastPathWithoutQuery = lastPath?.split("?")[0];
    const targetUrlWithoutQuery = targetUrl?.split("?")[0];
    const lastPathMatchesTarget = lastPathWithoutQuery === targetUrlWithoutQuery;
    // If the last path is what we expect or we have a last path but didn't specify a target URL, go back
    if ((lastPath && lastPathMatchesTarget) || (lastPath && !targetUrl)) {
        window.history.back();
    }
    // Otherwise, navigate to the target URL
    else {
        setLocation(targetUrl ?? LINKS.Home, { replace: true });
    }
}

/**
 * Creates a simple object mock for functions that require it, such as `getObjectUrl`.
 * 
 * NOTE: Do not use this willy-nilly. Make sure the function you're using it with is okay with 
 * having this limited set of properties.
 * @param objectType The object type
 * @param objectId The object ID
 * @param rootObjectId The object's root ID, if it is an object version
 * @returns A simple object mock
 */
export function asMockObject(objectType: ModelType | `${ModelType}`, objectId: string, rootObjectId?: string) {
    return {
        __typename: objectType,
        id: objectId,
        ...(rootObjectId ? { root: { __typename: objectType.replace("Version", ""), id: rootObjectId } } : {}),
    };
}

// Export the new standard upsert form hook
export { useStandardUpsertForm, type StandardUpsertFormConfig, type UseStandardUpsertFormProps } from "./useStandardUpsertForm.js";
