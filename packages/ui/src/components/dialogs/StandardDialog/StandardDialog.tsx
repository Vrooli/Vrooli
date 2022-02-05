import { StandardView } from 'components';
import { useCallback, useMemo } from 'react';
import { BaseObjectDialog } from '..';
import { StandardAdd } from 'components/views/StandardAdd/StandardAdd';
import { StandardUpdate } from 'components/views/StandardUpdate/StandardUpdate';
import { StandardDialogProps, ObjectDialogAction, ObjectDialogState } from 'components/dialogs/types';
import { useLocation, useRoute } from 'wouter';
import { useMutation } from '@apollo/client';
import { standard } from 'graphql/generated/standard';
import { standardAddMutation, standardUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { APP_LINKS } from '@local/shared';
import { Pubs } from 'utils';

export const StandardDialog = ({
    hasPrevious,
    hasNext,
    canEdit = false,
    partialData,
    session
}: StandardDialogProps) => {
    const [, setLocation] = useLocation();
    const [, params] = useRoute(`${APP_LINKS.SearchStandards}/:params*`);
    const [state, id] = useMemo(() => Boolean(params?.params) ? (params?.params as string).split("/") : [undefined, undefined], [params]);

    const [add] = useMutation<standard>(standardAddMutation);
    const [update] = useMutation<standard>(standardUpdateMutation);
    const [del] = useMutation<standard>(standardUpdateMutation);

    const onAction = useCallback((action: ObjectDialogAction) => {
        switch (action) {
            case ObjectDialogAction.Add:
                mutationWrapper({
                    mutation: add,
                    input: { },
                    onSuccess: ({ data }) => { 
                        const id = data?.id;
                        if (id) setLocation(`${APP_LINKS.SearchStandards}/view/${id}`, { replace: true });
                    },
                    onError: (response) => {
                        PubSub.publish(Pubs.Snack, { message: 'Error occurred.', severity: 'error', data: { error: response } });
                    }
                })
                break;
            case ObjectDialogAction.Cancel:
                setLocation(`${APP_LINKS.SearchStandards}/view`, { replace: true });
                break;
            case ObjectDialogAction.Close:
                window.history.back();
                break;
            case ObjectDialogAction.Edit:
                setLocation(`${APP_LINKS.SearchStandards}/edit/${id}`, { replace: true });
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
                        if (id) setLocation(`${APP_LINKS.SearchStandards}/view/${id}`, { replace: true });
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
                return 'Add Standard';
            case 'edit':
                return 'Edit Standard';
            default:
                return '';
        }
    }, [state]);

    const child = useMemo(() => {
        switch(state) {
            case 'add':
                return <StandardAdd onAdded={() => onAction(ObjectDialogAction.Add)} />
            case 'edit':
                return <StandardUpdate id="" onUpdated={() => onAction(ObjectDialogAction.Save)} />
            default:
                return <StandardView session={session} partialData={partialData} />
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