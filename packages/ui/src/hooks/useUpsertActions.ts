import { LINKS } from "@local/shared";
import { Method } from "api";
import { ObjectDialogAction } from "components/dialogs/types";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { setCookiePartialData } from "utils/cookies";
import { ListObject } from "utils/display/listTools";
import { getObjectUrl } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { ViewDisplayType } from "views/types";
import { MakeLazyRequest, useLazyFetch } from "./useLazyFetch";

/**
 * Creates logic for handling cancel, create, and update actions when 
 * creating a new object or updating an existing one. 
 * When done in a page, handles navigation. 
 * When done in a dialog, triggers the appropriate callback.
 * Also handles snack messages.
 */
export const useUpsertActions = <
    T extends { __typename: ListObject["__typename"], id: string },
    TCreateInput extends Record<string, any>,
    TUpdateInput extends Record<string, any>,
>({
    display,
    endpointCreate,
    endpointUpdate,
    isCreate,
    onCancel,
    onCompleted,
    onDeleted,
}: {
    display: ViewDisplayType,
    endpointCreate: { endpoint: string, method: Method },
    endpointUpdate: { endpoint: string, method: Method },
    isCreate: boolean,
    onCancel?: () => any, // Only used for dialog display
    onCompleted?: (data: T) => any, // Only used for dialog display
    onDeleted?: (id: string) => any, // Only used for dialog display
}) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const onAction = useCallback((action: ObjectDialogAction, item?: T) => {
        // URL of view page for the object
        const viewUrl = item ? getObjectUrl(item as any) : undefined;
        const hasPreviousPage = Boolean(sessionStorage.getItem("lastPath"));
        switch (action) {
            case ObjectDialogAction.Add:
                // Add the new object to the cache
                if (item) setCookiePartialData(item, "full");
                // Navigate to the view page for the new object
                if (display === "page") {
                    setLocation(viewUrl ?? LINKS.Home, { replace: true });
                } else {
                    onCompleted?.(item!);
                }
                // Display snack message
                if (isCreate) {
                    const rootType = (item?.__typename ?? "").replace("Version", "");
                    const objectTranslation = t(rootType, { count: 1, defaultValue: rootType });
                    PubSub.get().publishSnack({
                        message: t("ObjectCreated", { objectName: objectTranslation }),
                        severity: "Success",
                        buttonKey: "CreateNew",
                        buttonClicked: () => { setLocation(`${LINKS[rootType]}/add`); },
                    });
                }
                break;
            case ObjectDialogAction.Cancel:
            case ObjectDialogAction.Close:
                if (display === "page") {
                    if (!viewUrl && hasPreviousPage) window.history.back();
                    else setLocation(viewUrl ?? LINKS.Home, { replace: !hasPreviousPage });
                } else {
                    onCancel?.();
                }
                break;
            case ObjectDialogAction.Delete:
                // Leave the page
                if (display === "page") {
                    if (hasPreviousPage) window.history.back();
                    else setLocation(LINKS.Home, { replace: true });
                } else {
                    onDeleted?.(item!.id);
                }
                // Don't display snack message, as we don't have enough information for the message's "Undo" button
                break;
            case ObjectDialogAction.Save:
                // Update the object in the cache
                if (item) setCookiePartialData(item, "full");
                // Navigate to the view page for the object
                if (display === "page") {
                    if (!viewUrl && hasPreviousPage) window.history.back();
                    else setLocation(viewUrl ?? LINKS.Home, { replace: true });
                } else {
                    onCompleted?.(item!);
                }
                // Display snack message
                if (!isCreate) {
                    const rootType = (item?.__typename ?? "").replace("Version", "");
                    const objectTranslation = t(rootType, { count: 1, defaultValue: rootType });
                    PubSub.get().publishSnack({
                        message: t("ObjectUpdated", { objectName: objectTranslation }),
                        severity: "Success",
                    });
                }
                break;
        }
    }, [display, isCreate, setLocation, onCompleted, t, onCancel, onDeleted]);

    const handleCancel = useCallback(() => onAction(ObjectDialogAction.Cancel), [onAction]);
    const handleCreated = useCallback((data: T) => { onAction(ObjectDialogAction.Add, data); }, [onAction]);
    const handleUpdated = useCallback((data: T) => { onAction(ObjectDialogAction.Save, data); }, [onAction]);
    const handleCompleted = useCallback((data: T) => {
        const action = isCreate ? ObjectDialogAction.Add : ObjectDialogAction.Save;
        onAction(action, data);
    }, [isCreate, onAction]);
    const handleDeleted = useCallback((data: T) => { onAction(ObjectDialogAction.Delete, data); }, [onAction]);

    const [fetchCreate, { loading: isCreateLoading }] = useLazyFetch<TCreateInput, T>(endpointCreate);
    const [fetchUpdate, { loading: isUpdateLoading }] = useLazyFetch<TUpdateInput, T>(endpointUpdate);
    const fetch = (isCreate ? fetchCreate : fetchUpdate) as MakeLazyRequest<TCreateInput | TUpdateInput, T>;

    return {
        fetch,
        fetchCreate,
        fetchUpdate,
        handleCancel,
        handleCreated,
        handleDeleted,
        handleUpdated,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    };
};
