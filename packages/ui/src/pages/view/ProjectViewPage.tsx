import { useCallback, useMemo } from "react";
import { Box } from "@mui/material"
import { BaseObjectDialog, ProjectView } from "components";
import { ProjectViewPageProps } from "./types";
import { ObjectDialogAction } from "components/dialogs/types";
import { useLocation, useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
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
        <Box pt="10vh" sx={{ minHeight: '88vh' }}>
            {/* Add dialog */}
            <BaseObjectDialog
                hasNext={false}
                hasPrevious={false}
                onAction={onAction}
                open={isAddDialogOpen}
                title={"Add Project"}
            >
                <ProjectCreate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onCreated={(data: Project) => onAction(ObjectDialogAction.Add, data)}
                    session={session}
                />
            </BaseObjectDialog>
            {/* Update dialog */}
            <BaseObjectDialog
                hasNext={false}
                hasPrevious={false}
                onAction={onAction}
                open={isEditDialogOpen}
                title={"Update Project"}
            >
                <ProjectUpdate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onUpdated={() => onAction(ObjectDialogAction.Save)}
                    session={session}
                />
            </BaseObjectDialog>
            {/* Main view */}
            <ProjectView session={session} />
        </Box>
    )
}