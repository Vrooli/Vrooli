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
                if (data?.id) setLocation(`${APP_LINKS.Project}/${data?.id}`, { replace: true });
                else setLocation(APP_LINKS.Project, { replace: true });
                break;
            case ObjectDialogAction.Cancel:
                setLocation(`${APP_LINKS.Project}/${id}`, { replace: true });
                break;
            case ObjectDialogAction.Close:
                setLocation(`${APP_LINKS.Project}/${id}`, { replace: true });
                break;
            case ObjectDialogAction.Edit:
                setLocation(`${APP_LINKS.Project}/edit/${id}`, { replace: true });
                break;
            case ObjectDialogAction.Save:
                setLocation(`${APP_LINKS.Project}/${id}`, { replace: true });
                break;
        }
    }, [id, setLocation]);

    return (
        <Box pt="10vh" sx={{ minHeight: '88vh' }}>
            {/* Add dialog */}
            <BaseObjectDialog
                title={"Add Project"}
                open={isAddDialogOpen}
                hasPrevious={false}
                hasNext={false}
                onAction={onAction}
            >
                <ProjectCreate
                    session={session}
                    onCreated={(data: Project) => onAction(ObjectDialogAction.Add, data)}
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                />
            </BaseObjectDialog>
            {/* Update dialog */}
            <BaseObjectDialog
                title={"Update Project"}
                open={isEditDialogOpen}
                hasPrevious={false}
                hasNext={false}
                onAction={onAction}
            >
                <ProjectUpdate
                    session={session}
                    onUpdated={() => onAction(ObjectDialogAction.Save)}
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                />
            </BaseObjectDialog>
            {/* Main view */}
            <ProjectView session={session} />
        </Box>
    )
}