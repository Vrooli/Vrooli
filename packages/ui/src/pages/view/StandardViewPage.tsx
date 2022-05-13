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
                setLocation(`${APP_LINKS.Standard}/${data?.id}`, { replace: true });
                break;
            case ObjectDialogAction.Cancel:
                window.history.back();
                break;
            case ObjectDialogAction.Close:
                window.history.back();
                break;
            case ObjectDialogAction.Edit:
                setLocation(`${APP_LINKS.Standard}/edit/${id}`);
                break;
            case ObjectDialogAction.Save:
                window.history.back();
                break;
        }
    }, [id, setLocation]);

    return (
        <Box sx={{
            minHeight: '100vh',
            paddingTop: { xs: '64px', md: '80px' },
        }}>
            {/* Add dialog */}
            <BaseObjectDialog
                hasNext={false}
                hasPrevious={false}
                onAction={onAction}
                open={isAddDialogOpen}
                title={"Add Standard"}
            >
                <StandardCreate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onCreated={(data: Standard) => onAction(ObjectDialogAction.Add, data)}
                    session={session}
                />
            </BaseObjectDialog>
            {/* Update dialog */}
            <BaseObjectDialog
                hasNext={false}
                hasPrevious={false}
                onAction={onAction}
                open={isEditDialogOpen}
                title={"Update Standard"}
            >
                <StandardUpdate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onUpdated={() => onAction(ObjectDialogAction.Save)}
                    session={session}
                />
            </BaseObjectDialog>
            {/* Main view */}
            <StandardView session={session} />
        </Box>
    )
}