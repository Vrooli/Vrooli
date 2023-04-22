import { LINKS } from "@shared/consts";
import { useCallback, useMemo } from "react";
import { ObjectDialogAction } from "../../components/dialogs/types";
import { ViewDisplayType } from "../../views/types";
import { getObjectUrl } from "../navigation/openObject";
import { PubSub } from "../pubsub";
import { useLocation } from "../route";

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
                    PubSub.get().publishSnack({
                        message: `${item?.__typename ?? ""} created!`,
                        severity: "Success",
                        buttonKey: "CreateNew",
                        buttonClicked: () => { setLocation("add"); },
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
