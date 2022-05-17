import { StandardView } from 'components';
import { useCallback, useMemo } from 'react';
import { BaseObjectDialog } from '..';
import { StandardCreate } from 'components/views/StandardCreate/StandardCreate';
import { StandardUpdate } from 'components/views/StandardUpdate/StandardUpdate';
import { StandardDialogProps, ObjectDialogAction } from 'components/dialogs/types';
import { useLocation, useRoute } from 'wouter';
import { APP_LINKS } from '@local/shared';
import { Standard } from 'types';

export const StandardDialog = ({
    canEdit = false,
    hasNext,
    hasPrevious,
    partialData,
    session
}: StandardDialogProps) => {
    const [, setLocation] = useLocation();
    const [, params] = useRoute(`${APP_LINKS.SearchStandards}/:params*`);
    const [state, id] = useMemo(() => Boolean(params?.params) ? (params?.params as string).split("/") : [undefined, undefined], [params]);

    const onAction = useCallback((action: ObjectDialogAction, data?: any) => {
        switch (action) {
            case ObjectDialogAction.Add:
                setLocation(`${APP_LINKS.SearchStandards}/view/${data?.id}`, { replace: true });
                break;
            case ObjectDialogAction.Cancel:
                window.history.back();
                break;
            case ObjectDialogAction.Close:
                window.history.back();
                break;
            case ObjectDialogAction.Edit:
                setLocation(`${APP_LINKS.SearchStandards}/edit/${id}`);
                break;
            case ObjectDialogAction.Next:
                break;
            case ObjectDialogAction.Previous:
                break;
            case ObjectDialogAction.Save:
                window.history.back();
                break;
        }
    }, [id, setLocation]);

    const title = useMemo(() => {
        switch (state) {
            case 'add':
                return 'Add Standard';
            case 'edit':
                return 'Edit Standard';
            default:
                return '';
        }
    }, [state]);

    const child = useMemo(() => {
        switch (state) {
            case 'add':
                return <StandardCreate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onCreated={(data: Standard) => onAction(ObjectDialogAction.Add, data)}
                    session={session}
                />
            case 'edit':
                return <StandardUpdate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onUpdated={() => onAction(ObjectDialogAction.Save)}
                    session={session}
                />
            default:
                return <StandardView
                    partialData={partialData}
                    session={session}
                />
        }
    }, [onAction, partialData, session, state]);

    return (
        <BaseObjectDialog
            hasNext={hasNext}
            hasPrevious={hasPrevious}
            onAction={onAction}
            open={Boolean(params?.params)}
            title={title}
        >
            {child}
        </BaseObjectDialog>
    );
}