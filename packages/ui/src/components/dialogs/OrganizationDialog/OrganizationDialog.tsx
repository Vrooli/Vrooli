import { OrganizationView } from 'components';
import { useCallback, useMemo } from 'react';
import { BaseObjectDialog } from '..';
import { OrganizationCreate } from 'components/views/OrganizationCreate/OrganizationCreate';
import { OrganizationUpdate } from 'components/views/OrganizationUpdate/OrganizationUpdate';
import { OrganizationDialogProps, ObjectDialogAction } from 'components/dialogs/types';
import { useLocation, useRoute } from 'wouter';
import { APP_LINKS } from '@local/shared';
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

    const onAction = useCallback((action: ObjectDialogAction, data?: any) => {
        switch (action) {
            case ObjectDialogAction.Add:
                if (data?.id) setLocation(`${APP_LINKS.SearchOrganizations}/view/${data?.id}`, { replace: true });
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
                if (data?.id) setLocation(`${APP_LINKS.SearchOrganizations}/view/${id}`, { replace: true });
                break;
        }
    }, [id, setLocation]);

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
                return <OrganizationCreate session={session} onCreated={(data: Organization) => onAction(ObjectDialogAction.Add, data)} onCancel={() => onAction(ObjectDialogAction.Cancel)} />
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
            onAction={onAction}
        >
            {child}
        </BaseObjectDialog>
    );
}