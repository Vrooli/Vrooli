import { useCallback, useMemo } from "react";
import { Box } from "@mui/material"
import { BaseObjectDialog, ProjectView } from "components";
import { ProjectViewPageProps } from "./types";
import { ObjectDialogAction } from "components/dialogs/types";
import { useLocation, useRoute } from '@shared/route';
import { APP_LINKS } from "@shared/consts";
import { ProjectCreate } from "components/views/ProjectCreate/ProjectCreate";
import { Project } from "types";
import { ProjectUpdate } from "components/views/ProjectUpdate/ProjectUpdate";

export const ProjectViewPage = ({
    session
}: ProjectViewPageProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [matchView, paramsView] = useRoute(`${APP_LINKS.Project}/:id`);
    const [matchUpdate, paramsUpdate] = useRoute(`${APP_LINKS.Project}/edit/:id`);
    const id = useMemo(() => paramsView?.id ?? paramsUpdate?.id ?? '', [paramsView, paramsUpdate]);

    const isAddDialogOpen = useMemo(() => Boolean(matchView) && paramsView?.id === 'add', [matchView, paramsView]);
    const isEditDialogOpen = useMemo(() => Boolean(matchUpdate), [matchUpdate]);

    const onAction = useCallback((action: ObjectDialogAction, data?: any) => {
        switch (action) {
            case ObjectDialogAction.Add:
                setLocation(`${APP_LINKS.Project}/${data?.id}`, { replace: true });
                break;
            case ObjectDialogAction.Cancel:
                window.history.back();
                break;
            case ObjectDialogAction.Close:
                window.history.back();
                break;
            case ObjectDialogAction.Edit:
                setLocation(`${APP_LINKS.Project}/edit/${id}`);
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
            paddingBottom: 'calc(56px + env(safe-area-inset-bottom))',
        }}>
            {/* Add dialog */}
            <BaseObjectDialog
                onAction={onAction}
                open={isAddDialogOpen}
                title={"Add Project"}
                zIndex={200}
            >
                <ProjectCreate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onCreated={(data: Project) => onAction(ObjectDialogAction.Add, data)}
                    session={session}
                    zIndex={200}
                />
            </BaseObjectDialog>
            {/* Update dialog */}
            <BaseObjectDialog
                onAction={onAction}
                open={isEditDialogOpen}
                title={"Update Project"}
                zIndex={200}
            >
                <ProjectUpdate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onUpdated={() => onAction(ObjectDialogAction.Save)}
                    session={session}
                    zIndex={200}
                />
            </BaseObjectDialog>
            {/* Main view */}
            <ProjectView session={session} zIndex={200} />
        </Box>
    )
}