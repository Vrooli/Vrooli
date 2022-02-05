import { RoutineView } from 'components';
import { useCallback, useMemo } from 'react';
import { BaseObjectDialog } from '..';
import { RoutineAdd } from 'components/views/RoutineAdd/RoutineAdd';
import { RoutineUpdate } from 'components/views/RoutineUpdate/RoutineUpdate';
import { RoutineDialogProps, ObjectDialogAction, ObjectDialogState } from 'components/dialogs/types';
import { useLocation, useRoute } from 'wouter';
import { useMutation } from '@apollo/client';
import { routine } from 'graphql/generated/routine';
import { routineAddMutation, routineUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { APP_LINKS } from '@local/shared';
import { Pubs } from 'utils';

export const RoutineDialog = ({
    hasPrevious,
    hasNext,
    canEdit = false,
    partialData,
    session
}: RoutineDialogProps) => {
    const [, setLocation] = useLocation();
    const [, params] = useRoute(`${APP_LINKS.SearchRoutines}/:params*`);
    const [state, id] = useMemo(() => Boolean(params?.params) ? (params?.params as string).split("/") : [undefined, undefined], [params]);

    const [add] = useMutation<routine>(routineAddMutation);
    const [update] = useMutation<routine>(routineUpdateMutation);
    const [del] = useMutation<routine>(routineUpdateMutation);

    const onAction = useCallback((action: ObjectDialogAction) => {
        switch (action) {
            case ObjectDialogAction.Add:
                mutationWrapper({
                    mutation: add,
                    input: { },
                    onSuccess: ({ data }) => { 
                        const id = data?.id;
                        if (id) setLocation(`${APP_LINKS.SearchRoutines}/view/${id}`, { replace: true });
                    },
                    onError: (response) => {
                        PubSub.publish(Pubs.Snack, { message: 'Error occurred.', severity: 'error', data: { error: response } });
                    }
                })
                break;
            case ObjectDialogAction.Cancel:
                setLocation(`${APP_LINKS.SearchRoutines}/view`, { replace: true });
                break;
            case ObjectDialogAction.Close:
                window.history.back();
                break;
            case ObjectDialogAction.Edit:
                setLocation(`${APP_LINKS.SearchRoutines}/edit/${id}`, { replace: true });
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
                        if (id) setLocation(`${APP_LINKS.SearchRoutines}/view/${id}`, { replace: true });
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
                return 'Add Routine';
            case 'edit':
                return 'Edit Routine';
            default:
                return '';
        }
    }, [state]);

    const child = useMemo(() => {
        switch(state) {
            case 'add':
                return <RoutineAdd onAdded={() => onAction(ObjectDialogAction.Add)} />
            case 'edit':
                return <RoutineUpdate id="" onUpdated={() => onAction(ObjectDialogAction.Save)} />
            default:
                return <RoutineView session={session} partialData={partialData} />
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