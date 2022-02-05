import { ProjectView } from 'components';
import { useCallback, useMemo } from 'react';
import { BaseObjectDialog } from '..';
import { ProjectAdd } from 'components/views/ProjectAdd/ProjectAdd';
import { ProjectUpdate } from 'components/views/ProjectUpdate/ProjectUpdate';
import { ProjectDialogProps, ObjectDialogAction, ObjectDialogState } from 'components/dialogs/types';
import { useLocation, useRoute } from 'wouter';
import { useMutation } from '@apollo/client';
import { project } from 'graphql/generated/project';
import { projectAddMutation, projectUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { APP_LINKS } from '@local/shared';
import { Pubs } from 'utils';

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

    const [add] = useMutation<project>(projectAddMutation);
    const [update] = useMutation<project>(projectUpdateMutation);
    const [del] = useMutation<project>(projectUpdateMutation);

    const onAction = useCallback((action: ObjectDialogAction) => {
        switch (action) {
            case ObjectDialogAction.Add:
                mutationWrapper({
                    mutation: add,
                    input: { },
                    onSuccess: ({ data }) => { 
                        const id = data?.id;
                        if (id) setLocation(`${APP_LINKS.SearchProjects}/view/${id}`, { replace: true });
                    },
                    onError: (response) => {
                        PubSub.publish(Pubs.Snack, { message: 'Error occurred.', severity: 'error', data: { error: response } });
                    }
                })
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
                mutationWrapper({
                    mutation: update,
                    input: { },
                    onSuccess: ({ data }) => { 
                        const id = data?.id;
                        if (id) setLocation(`${APP_LINKS.SearchProjects}/view/${id}`, { replace: true });
                    },
                    onError: (response) => {
                        PubSub.publish(Pubs.Snack, { message: 'Error occurred.', severity: 'error', data: { error: response } });
                    }
                })
                break;
        }
    },  [add, update, del, setLocation, state]);

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
                return <ProjectAdd session={session} onAdded={() => onAction(ObjectDialogAction.Add)} onCancel={() => onAction(ObjectDialogAction.Cancel)} />
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
            canEdit={canEdit}
            state={Object.values(ObjectDialogState).includes(state ?? '') ? state as any : ObjectDialogState.View}
            onAction={onAction}
        >
            {child}
        </BaseObjectDialog>
    );
}