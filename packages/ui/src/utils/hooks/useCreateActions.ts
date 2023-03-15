import { LINKS } from "@shared/consts";
import { useLocation } from "@shared/route";
import { ObjectDialogAction } from "components/dialogs/types";
import { useCallback, useMemo } from "react";
import { uuidToBase36 } from "utils/navigation";
import { PubSub } from "utils/pubsub";

export const useCreateActions = <T extends { __typename: string, id: string }>() => {
    const [location, setLocation] = useLocation();

    // Determine if page should be displayed as a dialog or full page
    const hasPreviousPage = useMemo(() => Boolean(sessionStorage.getItem('lastPath')), [location]);

    const onAction = useCallback((action: ObjectDialogAction, item?: T) => {
        // Only navigate back if there is a previous page
        const pageRoot = window.location.pathname.split('/')[1];
        switch (action) {
            case ObjectDialogAction.Add:
                setLocation(`${uuidToBase36(item?.id ?? '')}`, { replace: !hasPreviousPage });
                PubSub.get().publishSnack({
                    message: `${item?.__typename ?? ''} created!`,
                    severity: 'Success',
                    buttonText: 'Create another',
                    buttonClicked: () => { setLocation(`add`); },
                } as any) //TODO
                break;
            case ObjectDialogAction.Cancel:
            case ObjectDialogAction.Close:
                if (hasPreviousPage) window.history.back();
                else setLocation(LINKS.Home);
                break;
        }
    }, [hasPreviousPage, setLocation]);

    const onCancel = useCallback(() => onAction(ObjectDialogAction.Cancel), [])
    const onCreated = useCallback((data: T) => onAction(ObjectDialogAction.Add, data), [])

    return {
        onCancel,
        onCreated,
    };
}