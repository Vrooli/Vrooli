import { DUMMY_ID, LINKS, OrArray } from "@local/shared";
import { ObjectDialogAction } from "components/dialogs/types";
import { FormProps } from "forms/types";
import { useCallback, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { NavigableObject } from "types";
import { removeCookieFormData, removeCookiePartialData, setCookiePartialData } from "utils/cookies";
import { ListObject } from "utils/display/listTools";
import { getObjectUrl } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";

type TType = OrArray<{ __typename: ListObject["__typename"], id: string }>;
type UseUpsertActionsProps<Model extends TType> = Pick<FormProps<Model, object>, "onCancel" | "onCompleted" | "onDeleted"> & {
    display: "dialog" | "page" | "partial",
    isCreate: boolean,
    objectId?: string,
    objectType: ListObject["__typename"],
}

/**
 * Creates logic for handling cancel, create, and update actions when 
 * creating a new object or updating an existing one. 
 * When done in a page, handles navigation. 
 * When done in a dialog, triggers the appropriate callback.
 * Also handles snack messages.
 */
export const useUpsertActions = <T extends TType>({
    display,
    isCreate,
    objectId,
    objectType,
    onCancel,
    onCompleted,
    onDeleted,
}: UseUpsertActionsProps<T>) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const isCacheOn = useRef(true);

    /** Helper function to navigate back or to a specific URL */
    const goBack = useCallback((targetUrl?: string) => {
        const hasPreviousPage = Boolean(sessionStorage.getItem("lastPath"));
        console.log("in goback a", targetUrl, hasPreviousPage, sessionStorage.getItem("lastPath"));
        if (!targetUrl && hasPreviousPage) {
            console.log("in goback history.back!");
            window.history.back();
        } else {
            console.log("in goback setlocation! replacing?", !hasPreviousPage);
            setLocation(targetUrl ?? LINKS.Home, { replace: !hasPreviousPage });
        }
    }, [setLocation]);

    /** Helper function to publish a snack message */
    const publishSnack = useCallback((actionType: "Created" | "Updated", count: number) => {
        const rootType = objectType.replace("Version", "");
        const objectTranslation = t(rootType, { count: 1, defaultValue: rootType });
        PubSub.get().publishSnack({
            message: t(`Object${actionType}`, { objectName: objectTranslation, count }),
            severity: "Success",
            ...(actionType === "Created" && {
                buttonKey: "CreateNew",
                buttonClicked: () => setLocation(`${LINKS[rootType]}/add`),
            }),
        });
    }, [objectType, setLocation, t]);

    const onAction = useCallback((action: ObjectDialogAction, item: T) => {
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

        const handleAddOrUpdate = (messageType: "Created" | "Updated") => {
            if (canStore) {
                setCookiePartialData(item as NavigableObject, "full"); // Update cache to view object more quickly
                removeCookieFormData(`${objectType}-${isCreate ? DUMMY_ID : objectId}`); // Remove form backup data from cache
                isCacheOn.current = false;
            }

            if (display === "page") { setLocation(viewUrl ?? LINKS.Home, { replace: true }); }
            else { onCompleted?.(item); }

            if ((isCreate && messageType === "Created") || (!isCreate && messageType === "Updated")) {
                publishSnack(messageType, Array.isArray(item) ? item.length : 1);
            }
        };

        switch (action) {
            case ObjectDialogAction.Add:
                if (item && !Array.isArray(item)) handleAddOrUpdate("Created");
                break;
            case ObjectDialogAction.Cancel:
                // Remove form backup data from cache
                if (canStore) {
                    removeCookieFormData(`${objectType}-${objectId}`);
                    isCacheOn.current = false;
                }
                if (display === "page") goBack(isCreate ? undefined : viewUrl);
                else onCancel?.();
                break;
            case ObjectDialogAction.Close:
                // DO NOT remove form backup data from cache
                if (display === "page") goBack(isCreate ? undefined : viewUrl);
                else onCancel?.();
                break;
            case ObjectDialogAction.Delete:
                // Remove both the object and the form backup data from cache
                if (canStore) {
                    removeCookiePartialData(item as NavigableObject);
                    removeCookieFormData(`${objectType}-${objectId}`);
                    isCacheOn.current = false;
                }
                if (display === "page") goBack();
                else onDeleted?.(item);
                // Don't display snack message, as we don't have enough information for the message's "Undo" button
                break;
            case ObjectDialogAction.Save:
                if (item) handleAddOrUpdate("Updated");
                break;
        }
    }, [display, isCreate, objectType, setLocation, onCompleted, publishSnack, goBack, onCancel, onDeleted]);

    const handleCancel = useCallback(() => onAction(ObjectDialogAction.Cancel, { __typename: objectType, id: objectId } as unknown as T), [objectId, objectType, onAction]);
    const handleCreated = useCallback((data: T) => { onAction(ObjectDialogAction.Add, data); }, [onAction]);
    const handleUpdated = useCallback((data: T) => { onAction(ObjectDialogAction.Save, data); }, [onAction]);
    const handleCompleted = useCallback((data: T) => { onAction(isCreate ? ObjectDialogAction.Add : ObjectDialogAction.Save, data); }, [isCreate, onAction]);
    const handleDeleted = useCallback((data: T) => { onAction(ObjectDialogAction.Delete, data); }, [onAction]);

    return {
        handleCancel,
        handleCreated,
        handleDeleted,
        handleUpdated,
        handleCompleted,
        isCacheOn,
    };
};
