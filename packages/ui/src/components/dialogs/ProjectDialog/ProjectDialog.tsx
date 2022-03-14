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
    hasPrevious,
    hasNext,
    canEdit = false,
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
                setLocation(`${APP_LINKS.SearchProjects}/view`, { replace: true });
                break;
            case ObjectDialogAction.Close:
                window.history.back();
                break;
            case ObjectDialogAction.Edit:
                setLocation(`${APP_LINKS.SearchProjects}/edit/${id}`, { replace: true });
                break;
            case ObjectDialogAction.Next:
                break;
            case ObjectDialogAction.Previous:
                break;
            case ObjectDialogAction.Save:
                setLocation(`${APP_LINKS.SearchProjects}/view/${id}`, { replace: true });
                break;
        }
    }, [id, setLocation]);

    const title = useMemo(() => {
        switch(state) {
            case 'add':
                return 'Add Project';
            case 'edit':
                return 'Edit Project';
            default:
                return '';
        }
    }, [state]);

    const child = useMemo(() => {
        switch(state) {
            case 'add':
                return <ProjectCreate session={session} onCreated={(data: Project) => onAction(ObjectDialogAction.Add, data)} onCancel={() => onAction(ObjectDialogAction.Cancel)} />
            case 'edit':
                return <ProjectUpdate session={session} onUpdated={() => onAction(ObjectDialogAction.Save)} onCancel={() => onAction(ObjectDialogAction.Cancel)} />
            default:
                return <ProjectView session={session} partialData={partialData} />
        }
    }, [state]);

    return (
        <BaseObjectDialog
            title={title}
            open={Boolean(params?.params)}
            hasPrevious={hasPrevious}
            hasNext={hasNext}
            onAction={onAction}
        >
            {child}
        </BaseObjectDialog>
    );
}