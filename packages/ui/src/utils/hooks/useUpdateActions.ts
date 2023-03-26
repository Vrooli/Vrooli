import { LINKS } from "@shared/consts";
import { useLocation } from "@shared/route";
import { ObjectDialogAction } from "components/dialogs/types";
import { useCallback, useMemo } from "react";
import { ViewDisplayType } from "views/types";

/**
 * Creates logic for handling cancel and update actions when updating an existing object. 
 * When done in a page, handles navigation. 
 * When done in a dialog, triggers the appropriate callback.
 */
export const useUpdateActions = <T extends { __typename: string, id: string }>(
    display: ViewDisplayType,
    onCancel: () => any, // Only used for dialog display
    onUpdated: (data: T) => any, // Only used for dialog display
) => {
    const [location, setLocation] = useLocation();

    // Determine if page should be displayed as a dialog or full page
    const hasPreviousPage = useMemo(() => Boolean(sessionStorage.getItem('lastPath')), [location]);

    const onAction = useCallback((action: ObjectDialogAction, data?: T) => {
        // Only navigate back if there is a previous page
        const pageRoot = window.location.pathname.split('/')[1];
        switch (action) {
            case ObjectDialogAction.Cancel:
            case ObjectDialogAction.Close:
                if (display === 'page') {
                    if (hasPreviousPage) window.history.back();
                    else setLocation(LINKS.Home);
                } else {
                    onCancel();
                }
                break;
            case ObjectDialogAction.Save:
                if (display === 'page') {
                    if (hasPreviousPage) window.history.back();
                    else setLocation(LINKS.Home);
                } else {
                    onUpdated(data!);
                }
                break;
        }
    }, [display, hasPreviousPage, onCancel, onUpdated, setLocation]);

    const handleCancel = useCallback(() => onAction(ObjectDialogAction.Cancel), [onAction])
    const handleUpdated = useCallback((data: T) => onAction(ObjectDialogAction.Save, data), [onAction])

    return {
        handleCancel,
        handleUpdated,
    };
}