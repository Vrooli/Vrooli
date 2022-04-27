import { ProjectView } from 'components';
import { useCallback, useMemo } from 'react';
import { BaseObjectDialog } from '..';
import { ProjectUpdate } from 'components/views/ProjectUpdate/ProjectUpdate';
import { ProjectDialogProps, ObjectDialogAction } from 'components/dialogs/types';
import { useLocation, useRoute } from 'wouter';
import { APP_LINKS } from '@local/shared';
import { ProjectCreate } from 'components/views/ProjectCreate/ProjectCreate';
import { Project } from 'types';

export const ProjectDialog = ({
    canEdit = false,
    hasNext,
    hasPrevious,
    partialData,
    session
}: ProjectDialogProps) => {
    const [, setLocation] = useLocation();
    const [, params] = useRoute(`${APP_LINKS.SearchProjects}/:params*`);
    const [state, id] = useMemo(() => Boolean(params?.params) ? (params?.params as string).split("/") : [undefined, undefined], [params]);

    const onAction = useCallback((action: ObjectDialogAction, data?: any) => {
        switch (action) {
            case ObjectDialogAction.Add:
                if (data?.id) setLocation(`${APP_LINKS.SearchProjects}/view/${data?.id}`, { replace: true });
                break;
            case ObjectDialogAction.Cancel:
                window.history.back();
                break;
            case ObjectDialogAction.Close:
                window.history.back();
                break;
            case ObjectDialogAction.Edit:
                setLocation(`${APP_LINKS.SearchProjects}/edit/${id}`);
                break;
            case ObjectDialogAction.Next:
                break;
            case ObjectDialogAction.Previous:
                break;
            case ObjectDialogAction.Save:
                window.history.back();
                break;
        }
    }, [id, setLocation]);

    const title = useMemo(() => {
        switch (state) {
            case 'add':
                return 'Add Project';
            case 'edit':
                return 'Edit Project';
            default:
                return '';
        }
    }, [state]);

    const child = useMemo(() => {
        switch (state) {
            case 'add':
                return <ProjectCreate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onCreated={(data: Project) => onAction(ObjectDialogAction.Add, data)}
                    session={session}
                />
            case 'edit':
                return <ProjectUpdate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onUpdated={() => onAction(ObjectDialogAction.Save)}
                    session={session}
                />
            default:
                return <ProjectView
                    partialData={partialData}
                    session={session}
                />
        }
    }, [onAction, partialData, session, state]);

    return (
        <BaseObjectDialog
            hasNext={hasNext}
            hasPrevious={hasPrevious}
            onAction={onAction}
            open={Boolean(params?.params)}
            title={title}
        >
            {child}
        </BaseObjectDialog>
    );
}