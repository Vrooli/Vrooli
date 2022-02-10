import { UserView } from 'components';
import { useCallback, useMemo } from 'react';
import { BaseObjectDialog } from '..';
import { UserDialogProps, ObjectDialogAction, ObjectDialogState } from 'components/dialogs/types';
import { useLocation, useRoute } from 'wouter';
import { useMutation } from '@apollo/client';
import { user } from 'graphql/generated/user';
import { profileUpdateMutation, userDeleteOneMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { APP_LINKS } from '@local/shared';
import { Pubs } from 'utils';

export const UserDialog = ({
    hasPrevious,
    hasNext,
    canEdit = false,
    partialData,
    session
}: UserDialogProps) => {
    const [, setLocation] = useLocation();
    const [, params] = useRoute(`${APP_LINKS.SearchUsers}/:params*`);
    const [state, id] = useMemo(() => Boolean(params?.params) ? (params?.params as string).split("/") : [undefined, undefined], [params]);

    const [update] = useMutation<user>(profileUpdateMutation);
    const [del] = useMutation<user>(userDeleteOneMutation);

    const onAction = useCallback((action: ObjectDialogAction) => {
        switch (action) {
            case ObjectDialogAction.Cancel:
                setLocation(`${APP_LINKS.SearchUsers}/view`, { replace: true });
                break;
            case ObjectDialogAction.Close:
                window.history.back();
                break;
            case ObjectDialogAction.Edit:
                setLocation(`${APP_LINKS.Settings}?page=profile?editing=true`);
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
                        if (id) setLocation(`${APP_LINKS.SearchUsers}/view/${id}`, { replace: true });
                    },
                    onError: (response) => {
                        PubSub.publish(Pubs.Snack, { message: 'Error occurred.', severity: 'error', data: { error: response } });
                    }
                })
                break;
        }
    },  [update, del, setLocation, state]);

    const title = useMemo(() => {
        switch(state) {
            case 'edit':
                return 'Edit Profile';
            default:
                return '';
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
            <UserView session={session} partialData={partialData} />
        </BaseObjectDialog>
    );
}