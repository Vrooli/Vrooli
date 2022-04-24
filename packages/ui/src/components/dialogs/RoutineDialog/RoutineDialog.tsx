import { RoutineView } from 'components';
import { useCallback, useMemo } from 'react';
import { BaseObjectDialog } from '..';
import { RoutineCreate } from 'components/views/RoutineCreate/RoutineCreate';
import { RoutineUpdate } from 'components/views/RoutineUpdate/RoutineUpdate';
import { RoutineDialogProps, ObjectDialogAction } from 'components/dialogs/types';
import { useLocation, useRoute } from 'wouter';
import { APP_LINKS } from '@local/shared';
import { Routine } from 'types';

export const RoutineDialog = ({
    canEdit = false,
    hasNext,
    hasPrevious,
    partialData,
    session
}: RoutineDialogProps) => {
    const [, setLocation] = useLocation();
    const [, params] = useRoute(`${APP_LINKS.SearchRoutines}/:params*`);
    const [state, id] = useMemo(() => Boolean(params?.params) ? (params?.params as string).split("/") : [undefined, undefined], [params]);

    const onAction = useCallback((action: ObjectDialogAction, data?: any) => {
        switch (action) {
            case ObjectDialogAction.Add:
                if (data?.id) setLocation(`${APP_LINKS.SearchRoutines}/view/${data?.id}`, { replace: true });
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
                setLocation(`${APP_LINKS.SearchRoutines}/view/${id}`, { replace: true });
                break;
        }
    }, [id, setLocation]);

    const title = useMemo(() => {
        switch (state) {
            case 'add':
                return 'Add Routine';
            case 'edit':
                return 'Edit Routine';
            default:
                return '';
        }
    }, [state]);

    const child = useMemo(() => {
        switch (state) {
            case 'add':
                return <RoutineCreate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onCreated={(data: Routine) => onAction(ObjectDialogAction.Add, data)}
                    session={session}
                />
            case 'edit':
                return <RoutineUpdate
                    onCancel={() => onAction(ObjectDialogAction.Cancel)}
                    onUpdated={() => onAction(ObjectDialogAction.Save)}
                    session={session}
                />
            default:
                return <RoutineView
                    partialData={partialData}
                    session={session}
                />
        }
    }, [state]);

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