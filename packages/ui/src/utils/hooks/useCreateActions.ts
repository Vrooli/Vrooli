import { LINKS } from "@shared/consts";
import { useLocation } from "@shared/route";
import { ObjectDialogAction } from "components/dialogs/types";
import { useCallback, useMemo } from "react";
import { getObjectUrl } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { ViewDisplayType } from "views/types";

/**
 * Creates logic for handling cancel and create actions when creating a new object. 
 * When done in a page, handles navigation. 
 * When done in a dialog, triggers the appropriate callback.
 * Also handles snack messages.
 */
export const useCreateActions = <T extends { __typename: string, id: string }>(
    display: ViewDisplayType,
    onCancel: () => any, // Only used for dialog display
    onCreated: (data: T) => any, // Only used for dialog display
) => {
    const [, setLocation] = useLocation();

    // Determine if page should be displayed as a dialog or full page
    const hasPreviousPage = useMemo(() => Boolean(sessionStorage.getItem('lastPath')), []);

    const onAction = useCallback((action: ObjectDialogAction, item?: T) => {
        switch (action) {
            case ObjectDialogAction.Add:
                if (display === 'page') {
                    // Only navigate back if there is a previous page
                    const url = getObjectUrl(item as any);
                    setLocation(url, { replace: !hasPreviousPage });
                } else {
                    onCreated(item!);
                }
                PubSub.get().publishSnack({
                    message: `${item?.__typename ?? ''} created!`,
                    severity: 'Success',
                    buttonText: 'Create another',
                    buttonClicked: () => { setLocation(`add`); },
                } as any) //TODO
                break;
            case ObjectDialogAction.Cancel:
            case ObjectDialogAction.Close:
                if (display === 'page') {
                    // Only navigate back if there is a previous page
                    if (hasPreviousPage) window.history.back();
                    else setLocation(LINKS.Home);
                } else {
                    onCancel();
                }
                break;
        }
    }, [display, hasPreviousPage, onCancel, onCreated, setLocation]);

    const handleCancel = useCallback(() => onAction(ObjectDialogAction.Cancel), [onAction])
    const handleCreated = useCallback((data: T) => onAction(ObjectDialogAction.Add, data), [onAction])

    return {
        handleCancel,
        handleCreated,
    };
}