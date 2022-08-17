import { UserView } from 'components';
import { useCallback, useMemo } from 'react';
import { BaseObjectDialog } from '..';
import { UserDialogProps, ObjectDialogAction } from 'components/dialogs/types';
import { useLocation, useRoute } from '@shared/route';
import { useMutation } from '@apollo/client';
import { user, userVariables } from 'graphql/generated/user';
import { profileUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { APP_LINKS } from '@shared/consts';
import { PubSub } from 'utils';

export const UserDialog = ({
    partialData,
    session,
    zIndex,
}: UserDialogProps) => {
    const [, setLocation] = useLocation();
    const [, params] = useRoute(`/search:params*`);
    const [state] = useMemo(() => Boolean(params?.params) ? (params?.params as string).split("/") : [undefined, undefined], [params]);

    const [update] = useMutation<user, userVariables>(profileUpdateMutation);

    const onAction = useCallback((action: ObjectDialogAction) => {
        switch (action) {
            case ObjectDialogAction.Cancel:
            case ObjectDialogAction.Close:
                window.history.back();
                break;
            case ObjectDialogAction.Edit:
                setLocation(`${APP_LINKS.Settings}?page="profile"`);
                break;
            case ObjectDialogAction.Next:
                break;
            case ObjectDialogAction.Previous:
                break;
            // case ObjectDialogAction.Save: TODO
            //     mutationWrapper({
            //         mutation: update,
            //         input: {},
            //         onSuccess: ({ data }) => {
            //             const id = data?.user?.id;
            //             if (id) setLocation(`${APP_LINKS.SearchUsers}/view/${id}`, { replace: true });
            //         },
            //         onError: (response) => {
            //             PubSub.get().publishSnack({ message: 'Error occurred.', severity: 'error', data: { error: response } });
            //         }
            //     })
            //     break;
        }
    }, [setLocation, update]);

    const title = useMemo(() => {
        switch (state) {
            case 'edit':
                return 'Edit Profile';
            default:
                return '';
        }
    }, [state]);

    return (
        <BaseObjectDialog
            onAction={onAction}
            open={Boolean(params?.params)}
            title={title}
            zIndex={zIndex}
        >
            <UserView 
                session={session} 
                partialData={partialData} 
                zIndex={zIndex}
            />
        </BaseObjectDialog>
    );
}