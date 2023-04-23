import { LINKS } from "@local/consts";
import { useCallback, useMemo } from "react";
import { ObjectDialogAction } from "../../components/dialogs/types";
import { getObjectUrl } from "../navigation/openObject";
import { PubSub } from "../pubsub";
import { useLocation } from "../route";
export const useUpsertActions = (display, isCreate, onCancel, onCompleted) => {
    const [, setLocation] = useLocation();
    const hasPreviousPage = useMemo(() => Boolean(sessionStorage.getItem("lastPath")), []);
    const onAction = useCallback((action, item) => {
        console.log("useUpsertActions.onAction", action, item);
        const viewUrl = item ? getObjectUrl(item) : undefined;
        switch (action) {
            case ObjectDialogAction.Add:
                if (display === "page") {
                    setLocation(viewUrl ?? LINKS.Home, { replace: !hasPreviousPage });
                }
                else {
                    onCompleted(item);
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
                    if (!viewUrl && hasPreviousPage)
                        window.history.back();
                    else
                        setLocation(viewUrl ?? LINKS.Home, { replace: !hasPreviousPage });
                }
                else {
                    onCancel();
                }
                break;
            case ObjectDialogAction.Save:
                if (display === "page") {
                    if (!viewUrl && hasPreviousPage)
                        window.history.back();
                    else
                        setLocation(viewUrl ?? LINKS.Home, { replace: !hasPreviousPage });
                }
                else {
                    onCompleted(item);
                }
                break;
        }
    }, [display, isCreate, setLocation, hasPreviousPage, onCompleted, onCancel]);
    const handleCancel = useCallback(() => onAction(ObjectDialogAction.Cancel), [onAction]);
    const handleCompleted = useCallback((data) => {
        const action = isCreate ? ObjectDialogAction.Add : ObjectDialogAction.Save;
        onAction(action, data);
    }, [isCreate, onAction]);
    return {
        handleCancel,
        handleCompleted,
    };
};
//# sourceMappingURL=useUpsertActions.js.map