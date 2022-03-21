import { useCallback, useMemo } from "react";
import { Box } from "@mui/material"
import { BaseObjectDialog, StandardView } from "components";
import { StandardViewPageProps } from "./types";
import { ObjectDialogAction } from "components/dialogs/types";
import { useLocation, useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { StandardCreate } from "components/views/StandardCreate/StandardCreate";
import { Standard } from "types";
import { StandardUpdate } from "components/views/StandardUpdate/StandardUpdate";

export const StandardViewPage = ({
    session
}: StandardViewPageProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [matchView, paramsView] = useRoute(`${APP_LINKS.Standard}/:id`);
    const [matchUpdate, paramsUpdate] = useRoute(`${APP_LINKS.Standard}/edit/:id`);
    const id = useMemo(() => paramsView?.id ?? paramsUpdate?.id ?? '', [paramsView, paramsUpdate]);

    const isAddDialogOpen = useMemo(() => Boolean(matchView) && paramsView?.id === 'add', [matchView, paramsView]);
    const isEditDialogOpen = useMemo(() => Boolean(matchUpdate), [matchUpdate]);

    const onAction = useCallback((action: ObjectDialogAction, data?: any) => {
        switch (action) {
            case ObjectDialogAction.Add:
                if (data?.id) setLocation(`${APP_LINKS.Standard}/${data?.id}`, { replace: true });
                else setLocation(APP_LINKS.Standard, { replace: true });
                break;
            case ObjectDialogAction.Cancel:
                setLocation(`${APP_LINKS.Standard}/${id}`, { replace: true });
                break;
            case ObjectDialogAction.Close:
                setLocation(`${APP_LINKS.Standard}/${id}`, { replace: true });
                break;
            case ObjectDialogAction.Edit:
                setLocation(`${APP_LINKS.Standard}/edit/${id}`, { replace: true });
                break;
            case ObjectDialogAction.Save:
                setLocation(`${APP_LINKS.Standard}/${id}`, { replace: true });
                break;
        }
    }, [id, setLocation]);

    return (
        <Box pt="10vh" sx={{ minHeight: '88vh' }}>
            {/* Add dialog */}
            <BaseObjectDialog
                title={"Add Standard"}
                open={isAddDialogOpen}
                hasPrevious={false}
                hasNext={false}
                onAction={onAction}
            >
                <StandardCreate
                    session={session}
                    onCreated={(data: Standard) => onAction(ObjectDialogAction.Add, data)}
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                />
            </BaseObjectDialog>
            {/* Update dialog */}
            <BaseObjectDialog
                title={"Update Standard"}
                open={isEditDialogOpen}
                hasPrevious={false}
                hasNext={false}
                onAction={onAction}
            >
                <StandardUpdate
                    session={session}
                    onUpdated={() => onAction(ObjectDialogAction.Save)}
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                />
            </BaseObjectDialog>
            {/* Main view */}
            <StandardView session={session} />
        </Box>
    )
}