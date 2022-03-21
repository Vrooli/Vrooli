import { useCallback, useMemo } from "react";
import { Box } from "@mui/material"
import { BaseObjectDialog, RoutineView } from "components";
import { RoutineViewPageProps } from "./types";
import { ObjectDialogAction } from "components/dialogs/types";
import { useLocation, useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { RoutineCreate } from "components/views/RoutineCreate/RoutineCreate";
import { Routine } from "types";
import { RoutineUpdate } from "components/views/RoutineUpdate/RoutineUpdate";

export const RoutineViewPage = ({
    session
}: RoutineViewPageProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [matchView, paramsView] = useRoute(`${APP_LINKS.Run}/:id`);
    const [matchUpdate, paramsUpdate] = useRoute(`${APP_LINKS.Run}/edit/:id`);
    const id = useMemo(() => paramsView?.id ?? paramsUpdate?.id ?? '', [paramsView, paramsUpdate]);

    const isAddDialogOpen = useMemo(() => Boolean(matchView) && paramsView?.id === 'add', [matchView, paramsView]);
    const isEditDialogOpen = useMemo(() => Boolean(matchUpdate), [matchUpdate]);

    const onAction = useCallback((action: ObjectDialogAction, data?: any) => {
        switch (action) {
            case ObjectDialogAction.Add:
                if (data?.id) setLocation(`${APP_LINKS.Run}/${data?.id}`, { replace: true });
                else setLocation(APP_LINKS.Run, { replace: true });
                break;
            case ObjectDialogAction.Cancel:
                setLocation(`${APP_LINKS.Run}/${id}`, { replace: true });
                break;
            case ObjectDialogAction.Close:
                setLocation(`${APP_LINKS.Run}/${id}`, { replace: true });
                break;
            case ObjectDialogAction.Edit:
                setLocation(`${APP_LINKS.Run}/edit/${id}`, { replace: true });
                break;
            case ObjectDialogAction.Save:
                setLocation(`${APP_LINKS.Run}/${id}`, { replace: true });
                break;
        }
    }, [id, setLocation]);

    return (
        <Box pt="10vh" sx={{ minHeight: '88vh' }}>
            {/* Add dialog */}
            <BaseObjectDialog
                title={"Add Routine"}
                open={isAddDialogOpen}
                hasPrevious={false}
                hasNext={false}
                onAction={onAction}
            >
                <RoutineCreate
                    session={session}
                    onCreated={(data: Routine) => onAction(ObjectDialogAction.Add, data)}
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                />
            </BaseObjectDialog>
            {/* Update dialog */}
            <BaseObjectDialog
                title={"Update Routine"}
                open={isEditDialogOpen}
                hasPrevious={false}
                hasNext={false}
                onAction={onAction}
            >
                <RoutineUpdate
                    session={session}
                    onUpdated={() => onAction(ObjectDialogAction.Save)}
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                />
            </BaseObjectDialog>
            {/* Main view */}
            <RoutineView session={session} />
        </Box>
    )
}