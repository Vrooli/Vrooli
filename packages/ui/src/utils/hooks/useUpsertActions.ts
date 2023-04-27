import { LINKS, useLocation } from "@local/shared";
import { ObjectDialogAction } from "components/dialogs/types";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { getObjectUrl } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { ViewDisplayType } from "views/types";

/**
 * Creates logic for handling cancel, create, and update actions when 
 * creating a new object or updating an existing one. 
 * When done in a page, handles navigation. 
 * When done in a dialog, triggers the appropriate callback.
 * Also handles snack messages.
 */
export const useUpsertActions = <T extends { __typename: string, id: string }>(
    display: ViewDisplayType,
    isCreate: boolean,
    onCancel?: () => any, // Only used for dialog display
    onCompleted?: (data: T) => any, // Only used for dialog display
) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    // We can only use history.back()/replace if there is a previous page
    const hasPreviousPage = useMemo(() => Boolean(sessionStorage.getItem("lastPath")), []);

    const onAction = useCallback((action: ObjectDialogAction, item?: T) => {
        console.log("useUpsertActions.onAction", action, item);
        // URL of view page for the object
        const viewUrl = item ? getObjectUrl(item as any) : undefined;
        switch (action) {
            case ObjectDialogAction.Add:
                if (display === "page") {
                    setLocation(viewUrl ?? LINKS.Home, { replace: !hasPreviousPage });
                } else {
                    onCompleted!(item!);
                }
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
                    onCancel!();
                }
                break;
            case ObjectDialogAction.Save:
                if (display === "page") {
                    if (!viewUrl && hasPreviousPage) window.history.back();
                    else setLocation(viewUrl ?? LINKS.Home, { replace: !hasPreviousPage });
                } else {
                    onCompleted!(item!);
                }
                break;
        }
    }, [display, isCreate, setLocation, hasPreviousPage, onCompleted, onCancel]);

    const handleCancel = useCallback(() => onAction(ObjectDialogAction.Cancel), [onAction]);
    const handleCompleted = useCallback((data: T) => {
        const action = isCreate ? ObjectDialogAction.Add : ObjectDialogAction.Save;
        onAction(action, data);
    }, [isCreate, onAction]);

    return {
        handleCancel,
        handleCompleted,
    };
};
