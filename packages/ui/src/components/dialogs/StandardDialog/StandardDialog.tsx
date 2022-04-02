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
    hasPrevious,
    hasNext,
    canEdit = false,
    partialData,
    session
}: StandardDialogProps) => {
    const [, setLocation] = useLocation();
    const [, params] = useRoute(`${APP_LINKS.SearchStandards}/:params*`);
    const [state, id] = useMemo(() => Boolean(params?.params) ? (params?.params as string).split("/") : [undefined, undefined], [params]);

    const onAction = useCallback((action: ObjectDialogAction, data?: any) => {
        switch (action) {
            case ObjectDialogAction.Add:
                if (data?.id) setLocation(`${APP_LINKS.SearchStandards}/view/${data?.id}`, { replace: true });
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
                setLocation(`${APP_LINKS.SearchStandards}/view/${id}`, { replace: true });
                break;
        }
    },  [id, setLocation]);

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
                return <StandardCreate session={session} onCreated={(data: Standard) => onAction(ObjectDialogAction.Add, data)} onCancel={() => onAction(ObjectDialogAction.Cancel)} />
            case 'edit':
                return <StandardUpdate session={session} onUpdated={() => onAction(ObjectDialogAction.Save)} onCancel={() => onAction(ObjectDialogAction.Cancel)} />
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
            onAction={onAction}
            session={session}
        >
            {child}
        </BaseObjectDialog>
    );
}