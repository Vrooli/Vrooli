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
    const [matchView, paramsView] = useRoute(`${APP_LINKS.Routine}/:id`);
    const [matchUpdate, paramsUpdate] = useRoute(`${APP_LINKS.Routine}/edit/:id`);
    const id = useMemo(() => paramsView?.id ?? paramsUpdate?.id ?? '', [paramsView, paramsUpdate]);

    const isAddDialogOpen = useMemo(() => Boolean(matchView) && paramsView?.id === 'add', [matchView, paramsView]);
    const isEditDialogOpen = useMemo(() => Boolean(matchUpdate), [matchUpdate]);

    const onAction = useCallback((action: ObjectDialogAction, data?: any) => {
        switch (action) {
            case ObjectDialogAction.Add:
                setLocation(`${APP_LINKS.Routine}/${data?.id}`, { replace: true });
                break;
            case ObjectDialogAction.Cancel:
                window.history.back();
                break;
            case ObjectDialogAction.Close:
                window.history.back();
                break;
            case ObjectDialogAction.Edit:
                setLocation(`${APP_LINKS.Routine}/edit/${id}`);
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
                onAction={onAction}
                open={isAddDialogOpen}
                title={"Add Routine"}
            >
                <RoutineCreate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onCreated={(data: Routine) => onAction(ObjectDialogAction.Add, data)}
                    session={session}
                />
            </BaseObjectDialog>
            {/* Update dialog */}
            <BaseObjectDialog
                onAction={onAction}
                open={isEditDialogOpen}
                title={"Update Routine"}
            >
                <RoutineUpdate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onUpdated={() => onAction(ObjectDialogAction.Save)}
                    session={session}
                />
            </BaseObjectDialog>
            {/* Main view */}
            <RoutineView session={session} />
        </Box>
    )
}