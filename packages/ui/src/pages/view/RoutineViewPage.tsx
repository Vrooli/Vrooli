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
import { validate as uuidValidate } from 'uuid';
import { parseSearchParams } from "utils";

export const RoutineViewPage = ({
    session
}: RoutineViewPageProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [matchView, paramsView] = useRoute(`${APP_LINKS.Routine}/:id`);
    const [matchUpdate, paramsUpdate] = useRoute(`${APP_LINKS.Routine}/edit/:id`);
    const id = useMemo(() => paramsView?.id ?? paramsUpdate?.id ?? '', [paramsView, paramsUpdate]);

    const { isAddDialogOpen, isEditDialogOpen } = useMemo(() => {
        const searchParams = parseSearchParams(window.location.search);
        return {
            /**
             * Create dialog is open if id is not in the shape of an id, and the
             * build search param is not present
             */
            isAddDialogOpen: matchView && !uuidValidate(paramsView?.id ?? '') && !searchParams.build,
            isEditDialogOpen: matchUpdate,
        }
    }, [matchUpdate, matchView, paramsView?.id]);

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
                zIndex={200}
            >
                <RoutineCreate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onCreated={(data: Routine) => onAction(ObjectDialogAction.Add, data)}
                    session={session}
                    zIndex={200}
                />
            </BaseObjectDialog>
            {/* Update dialog */}
            <BaseObjectDialog
                onAction={onAction}
                open={isEditDialogOpen}
                title={"Update Routine"}
                zIndex={200}
            >
                <RoutineUpdate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onUpdated={() => onAction(ObjectDialogAction.Save)}
                    session={session}
                    zIndex={200}
                />
            </BaseObjectDialog>
            {/* Main view */}
            <RoutineView session={session} zIndex={200} />
        </Box>
    )
}