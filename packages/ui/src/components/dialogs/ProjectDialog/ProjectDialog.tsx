import { ProjectView } from 'components';
import { useCallback, useMemo } from 'react';
import { BaseObjectDialog } from '..';
import { ProjectUpdate } from 'components/views/ProjectUpdate/ProjectUpdate';
import { ProjectDialogProps, ObjectDialogAction } from 'components/dialogs/types';
import { useRoute } from '@shared/route';
import { ProjectCreate } from 'components/views/ProjectCreate/ProjectCreate';
import { Project } from 'types';

export const ProjectDialog = ({
    partialData,
    session,
    zIndex,
}: ProjectDialogProps) => {
    const [, params] = useRoute(`/search:params*`);
    const [state] = useMemo(() => Boolean(params?.params) ? (params?.params as string).split("/") : [undefined, undefined], [params]);

    const onAction = useCallback((action: ObjectDialogAction, data?: any) => {
        switch (action) {
            case ObjectDialogAction.Add:
            case ObjectDialogAction.Cancel:
            case ObjectDialogAction.Close:
            case ObjectDialogAction.Edit:
            case ObjectDialogAction.Save:
                window.history.back();
                break;
        }
    }, []);

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
                    zIndex={zIndex}
                />
            case 'edit':
                return <ProjectUpdate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onUpdated={() => onAction(ObjectDialogAction.Save)}
                    session={session}
                    zIndex={zIndex}
                />
            default:
                return <ProjectView
                    partialData={partialData}
                    session={session}
                    zIndex={zIndex}
                />
        }
    }, [onAction, partialData, session, state, zIndex]);

    return (
        <BaseObjectDialog
            onAction={onAction}
            open={Boolean(params?.params)}
            title={title}
            zIndex={zIndex}
        >
            {child}
        </BaseObjectDialog>
    );
}