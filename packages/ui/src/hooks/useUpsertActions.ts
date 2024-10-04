import { CommonKey, DUMMY_ID, GqlModelType, LINKS, ListObject, NavigableObject, OrArray, getObjectUrl } from "@local/shared";
import { ObjectDialogAction } from "components/dialogs/types";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { SetLocation, useLocation } from "route";
import { removeCookieFormData, removeCookiePartialData, setCookieAllowFormCache, setCookiePartialData } from "utils/localStorage";
import { PubSub } from "utils/pubsub";
import { FormProps } from "../types";

type TType = OrArray<{ __typename: ListObject["__typename"], id: string }>;
type UseUpsertActionsProps<Model extends TType> = Pick<FormProps<Model, object>, "onCancel" | "onCompleted" | "onDeleted"> & {
    display: "dialog" | "page" | "partial",
    isCreate: boolean,
    objectId?: string,
    objectType: ListObject["__typename"],
    onAction?: (action: ObjectDialogAction, item: Model) => void,
    rootObjectId?: string,
    suppressSnack?: boolean,
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
export function asMockObject(objectType: GqlModelType | `${GqlModelType}`, objectId: string, rootObjectId?: string) {
    return {
        __typename: objectType,
        id: objectId,
        ...(rootObjectId ? { root: { __typename: objectType.replace("Version", ""), id: rootObjectId } } : {}),
    };
}

/**
 * Creates logic for handling cancel, create, and update actions when 
 * creating a new object or updating an existing one. 
 * When done in a page, handles navigation. 
 * When done in a dialog, triggers the appropriate callback.
 * Also handles snack messages.
 */
export function useUpsertActions<T extends TType>({
    display,
    isCreate,
    objectId,
    objectType,
    onAction,
    onCancel,
    onCompleted,
    onDeleted,
    rootObjectId,
    suppressSnack,
}: UseUpsertActionsProps<T>) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    /** Helper function to publish a snack message */
    const publishSnack = useCallback((actionType: "Created" | "Updated", count: number, item: NavigableObject) => {
        if (suppressSnack) return;
        const rootType = objectType.replace("Version", "");
        const objectTranslation = t(rootType, { count: 1, defaultValue: rootType });

        // Some actions have buttons
        let buttonKey: CommonKey | undefined;
        let buttonClicked: (() => unknown) | undefined;
        // Create ations typically have a button to create a new object
        if (actionType === "Created") {
            // But for some objects, it makes more sense for the button to open the object
            const shouldOpenObject = [GqlModelType.Report].includes(objectType as GqlModelType);
            if (shouldOpenObject) {
                buttonKey = "View";
                const url = objectType === GqlModelType.Report ? `${LINKS.Reports}${getObjectUrl(item)}` : getObjectUrl(item);
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

    const handleAction = useCallback((action: ObjectDialogAction, item: T) => {
        let viewUrl: string | undefined;
        let objectId: string | undefined;
        let canStore = false;
        if (item && !Array.isArray(item)) {
            viewUrl = getObjectUrl(item);
            // Should hopefully have id if not an array
            if (item.id) {
                objectId = item.id;
                canStore = true;
            } else {
                console.warn("No id found on object. This may be a mistake.", item);
            }
        }

        function handleAddOrUpdate(messageType: "Created" | "Updated", item: NavigableObject) {
            if (canStore) {
                setCookiePartialData(item as NavigableObject, "full"); // Update cache to view object more quickly
                const formId = isCreate ? DUMMY_ID : objectId;
                if (formId) {
                    removeCookieFormData(objectType as GqlModelType, formId); // Remove form backup data from cache
                    setCookieAllowFormCache(objectType as GqlModelType, formId, false);
                }
            }

            if (display === "page") { setLocation(viewUrl ?? LINKS.Home, { replace: true }); }

            if ((isCreate && messageType === "Created") || (!isCreate && messageType === "Updated")) {
                publishSnack(messageType, Array.isArray(item) ? item.length : 1, item);
            }
        }

        switch (action) {
            case ObjectDialogAction.Add:
                if (item && !Array.isArray(item)) handleAddOrUpdate("Created", item);
                onCompleted?.(item);
                break;
            case ObjectDialogAction.Cancel:
                // Remove form backup data from cache
                if (canStore) {
                    const formId = isCreate ? DUMMY_ID : objectId;
                    if (formId) {
                        removeCookieFormData(objectType as GqlModelType, formId);
                        setCookieAllowFormCache(objectType as GqlModelType, formId, false);
                    }
                }
                if (display === "page") goBack(setLocation, isCreate ? undefined : viewUrl);
                else onCancel?.();
                break;
            case ObjectDialogAction.Close:
                // DO NOT remove form backup data from cache
                if (display === "page") goBack(setLocation, isCreate ? undefined : viewUrl);
                else onCancel?.();
                break;
            case ObjectDialogAction.Delete:
                // Remove both the object and the form backup data from cache
                if (canStore) {
                    removeCookiePartialData(item as NavigableObject);
                    const formId = isCreate ? DUMMY_ID : objectId;
                    if (formId) {
                        removeCookieFormData(objectType as GqlModelType, formId);
                        setCookieAllowFormCache(objectType as GqlModelType, formId, false);
                    }
                }
                if (display === "page") goBack(setLocation);
                else onDeleted?.(item);
                // Don't display snack message, as we don't have enough information for the message's "Undo" button
                break;
            case ObjectDialogAction.Save:
                if (item && !Array.isArray(item)) handleAddOrUpdate("Updated", item);
                onCompleted?.(item);
                break;
        }

        onAction?.(action, item);
    }, [onAction, display, isCreate, objectType, setLocation, onCompleted, publishSnack, onCancel, onDeleted]);

    const mockObject = useMemo(function mockObjectMemo() {
        return asMockObject(objectType as GqlModelType, objectId ?? DUMMY_ID, rootObjectId) as unknown as T;
    }, [objectId, objectType, rootObjectId]);

    const handleCancel = useCallback(() => handleAction(ObjectDialogAction.Cancel, mockObject), [mockObject, handleAction]);
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
