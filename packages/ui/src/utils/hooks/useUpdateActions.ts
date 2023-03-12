import { LINKS } from "@shared/consts";
import { useLocation } from "@shared/route";
import { ObjectDialogAction } from "components/dialogs/types";
import { useCallback, useMemo } from "react";

export const useUpdateActions = <T extends { __typename: string, id: string }>() => {
    const [location, setLocation] = useLocation();

    // Determine if page should be displayed as a dialog or full page
    const hasPreviousPage = useMemo(() => Boolean(sessionStorage.getItem('lastPath')), [location]);

    const onAction = useCallback((action: ObjectDialogAction, data?: T) => {
        // Only navigate back if there is a previous page
        const pageRoot = window.location.pathname.split('/')[1];
        switch (action) {
            case ObjectDialogAction.Cancel:
            case ObjectDialogAction.Close:
                if (hasPreviousPage) window.history.back();
                else setLocation(LINKS.Home);
                break;
            case ObjectDialogAction.Save:
                if (hasPreviousPage) window.history.back();
                else setLocation(LINKS.Home);
                break;
        }
    }, [hasPreviousPage, setLocation]);

    const onCancel = useCallback(() => onAction(ObjectDialogAction.Cancel), [])
    const onUpdated = useCallback((data: T) => onAction(ObjectDialogAction.Save, data), [])

    return {
        onCancel,
        onUpdated,
    };
}