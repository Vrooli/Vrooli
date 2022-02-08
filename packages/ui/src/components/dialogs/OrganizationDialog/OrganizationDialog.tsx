import { OrganizationView } from 'components';
import { useCallback, useMemo } from 'react';
import { BaseObjectDialog } from '..';
import { OrganizationAdd } from 'components/views/OrganizationAdd/OrganizationAdd';
import { OrganizationUpdate } from 'components/views/OrganizationUpdate/OrganizationUpdate';
import { OrganizationDialogProps, ObjectDialogAction, ObjectDialogState } from 'components/dialogs/types';
import { useLocation, useRoute } from 'wouter';
import { useMutation } from '@apollo/client';
import { organization } from 'graphql/generated/organization';
import { organizationAddMutation, organizationUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { APP_LINKS } from '@local/shared';
import { Pubs } from 'utils';
import { Organization } from 'types';

export const OrganizationDialog = ({
    hasPrevious,
    hasNext,
    canEdit = false,
    partialData,
    session
}: OrganizationDialogProps) => {
    const [, setLocation] = useLocation();
    const [, params] = useRoute(`${APP_LINKS.SearchOrganizations}/:params*`);
    const [state, id] = useMemo(() => Boolean(params?.params) ? (params?.params as string).split("/") : [undefined, undefined], [params]);
    console.log("OrganizationDialog", { params, state, id });

    const [add] = useMutation<organization>(organizationAddMutation);
    const [update] = useMutation<organization>(organizationUpdateMutation);
    const [del] = useMutation<organization>(organizationUpdateMutation);

    const onAction = useCallback((action: ObjectDialogAction, data?: any) => {
        switch (action) {
            case ObjectDialogAction.Add:
                const id = data?.id;
                if (id) setLocation(`${APP_LINKS.SearchOrganizations}/view/${id}`, { replace: true });
                break;
            case ObjectDialogAction.Cancel:
                setLocation(`${APP_LINKS.SearchOrganizations}/view`, { replace: true });
                break;
            case ObjectDialogAction.Close:
                window.history.back();
                break;
            case ObjectDialogAction.Edit:
                setLocation(`${APP_LINKS.SearchOrganizations}/edit/${id}`, { replace: true });
                break;
            case ObjectDialogAction.Next:
                break;
            case ObjectDialogAction.Previous:
                break;
            case ObjectDialogAction.Save:
                mutationWrapper({
                    mutation: update,
                    input: {},
                    onSuccess: ({ data }) => {
                        const id = data?.id;
                        if (id) setLocation(`${APP_LINKS.SearchOrganizations}/view/${id}`, { replace: true });
                    },
                    onError: (response) => {
                        PubSub.publish(Pubs.Snack, { message: 'Error occurred.', severity: 'error', data: { error: response } });
                    }
                })
                break;
        }
    }, [add, update, del, setLocation, state]);

    const title = useMemo(() => {
        switch (state) {
            case 'add':
                return 'Add Organization';
            case 'edit':
                return 'Edit Organization';
            default:
                return '';
        }
    }, [state]);

    const child = useMemo(() => {
        switch (state) {
            case 'add':
                return <OrganizationAdd session={session} onAdded={(data: Organization) => onAction(ObjectDialogAction.Add, data)} onCancel={() => onAction(ObjectDialogAction.Cancel)} />
            case 'edit':
                return <OrganizationUpdate session={session} onUpdated={() => onAction(ObjectDialogAction.Save)} onCancel={() => onAction(ObjectDialogAction.Cancel)} />
            default:
                return <OrganizationView session={session} partialData={partialData} />
        }
    }, [state, onAction, session, partialData]);

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