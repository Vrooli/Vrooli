import { StandardView } from 'components';
import { useCallback, useMemo } from 'react';
import { BaseObjectDialog } from '..';
import { StandardCreate } from 'components/views/StandardCreate/StandardCreate';
import { StandardUpdate } from 'components/views/StandardUpdate/StandardUpdate';
import { StandardDialogProps, ObjectDialogAction } from 'components/dialogs/types';
import { useRoute } from '@local/shared';
import { Standard } from 'types';

export const StandardDialog = ({
    partialData,
    session,
    zIndex,
}: StandardDialogProps) => {
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
                    zIndex={zIndex}
                />
            case 'edit':
                return <StandardUpdate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onUpdated={() => onAction(ObjectDialogAction.Save)}
                    session={session}
                    zIndex={zIndex}
                />
            default:
                return <StandardView
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