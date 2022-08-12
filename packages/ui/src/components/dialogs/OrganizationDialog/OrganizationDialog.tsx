import { OrganizationView } from 'components';
import { useCallback, useMemo } from 'react';
import { BaseObjectDialog } from '..';
import { OrganizationCreate } from 'components/views/OrganizationCreate/OrganizationCreate';
import { OrganizationUpdate } from 'components/views/OrganizationUpdate/OrganizationUpdate';
import { OrganizationDialogProps, ObjectDialogAction } from 'components/dialogs/types';
import { useRoute } from 'wouter';
import { Organization } from 'types';

export const OrganizationDialog = ({
    partialData,
    session,
    zIndex,
}: OrganizationDialogProps) => {
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
                return <OrganizationCreate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onCreated={(data: Organization) => onAction(ObjectDialogAction.Add, data)}
                    session={session}
                    zIndex={zIndex}
                />
            case 'edit':
                return <OrganizationUpdate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onUpdated={() => onAction(ObjectDialogAction.Save)}
                    session={session}
                    zIndex={zIndex}
                />
            default:
                return <OrganizationView
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