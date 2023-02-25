import {
    Dialog,
    IconButton,
    Stack,
    Tooltip,
    Typography,
    useTheme
} from '@mui/material';
import { BaseObjectDialog, DialogTitle, OrganizationCreate, ProjectCreate, RoutineCreate, ShareSiteDialog, StandardCreate } from 'components';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { SelectOrCreateDialogProps, SelectOrCreateObject, SelectOrCreateObjectType } from '../types';
import { SearchList } from 'components/lists';
import { useLazyQuery } from 'api/hooks';
import { SearchType, searchTypeToParams } from 'utils';
import { removeSearchParams, useLocation } from '@shared/route';
import { AddIcon } from '@shared/icons';
import { getCurrentUser } from 'utils/authentication';
import { useTranslation } from 'react-i18next';
import { FindByIdInput } from '@shared/consts';
import { CreatePageProps } from 'pages';
import { CreateProps } from 'components/views/types';
import { isOfType } from '@shared/utils';
import { SearchParams } from 'utils/search/schemas/base';
import { CommonKey } from '@shared/translations';

type CreateViewTypes = ({
    [K in SelectOrCreateObjectType]: K extends (`${string}Version` | 'User') ?
    never :
    K
})[SelectOrCreateObjectType]

/**
 * Maps SelectOrCreateObject types to create components (excluding "User" and types that end with 'Version')
 */
const createMap: { [K in CreateViewTypes]: (props: CreateProps<any>) => JSX.Element } = {
    Api: {} as any,//ApiCreate,
    Note: {} as any,//NoteCreate,
    Organization: OrganizationCreate,
    Project: ProjectCreate,
    Routine: RoutineCreate,
    SmartContract: {} as any,//SmartContractCreate,
    Standard: StandardCreate,
}

export const SelectOrCreateDialog = <T extends SelectOrCreateObject>({
    handleAdd,
    handleClose,
    help,
    isOpen,
    objectType,
    session,
    where,
    zIndex,
}: SelectOrCreateDialogProps<T>) => {
    console.log('selectorcreate 1', objectType);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);
    const { helpText, titleAria } = useMemo(() => {
        return {
            helpText: help ?? t('SelectOrCreateDialogHelp', { objectType: t(objectType, { count: 1, defaultValue: objectType }) }),
            titleAria: `select-or-create-${objectType}-dialog-title`,
        };
    }, [help, objectType, t]);
    const CreateView = useMemo<((props: CreatePageProps) => JSX.Element) | null>(() =>
        objectType === 'User' ? null : createMap[objectType], [objectType]);

    const [{ advancedSearchSchema, endpoint, query }, setSearchParams] = useState<Partial<SearchParams>>({});
    useEffect(() => {
        const fetchParams = async () => {
            const params = searchTypeToParams[objectType];
            if (!params) return;
            setSearchParams(await params());
        };
        fetchParams();
    }, [objectType]);

    /**
     * Before closing, remove all URL search params for advanced search
     */
    const onClose = useCallback(() => {
        // Clear search params
        removeSearchParams(setLocation, [
            ...(advancedSearchSchema?.fields.map(f => f.fieldName) ?? []),
            'advanced',
            'sort',
            'time',
        ]);
        handleClose();
    }, [advancedSearchSchema?.fields, handleClose, setLocation]);

    // Create new item dialog (for all object types except users)
    const [isCreateOpen, setIsCreateOpen] = useState(false);
    const handleCreated = useCallback((item: SelectOrCreateObject) => {
        setIsCreateOpen(false);
        // Versioned objects are always created from the perspective of the version, and not the root.
        // If the object type is a root of a versioned object, we must change the shape before calling handleAdd
        if (isOfType(objectType, 'Api', 'Note', 'Project', 'Routine', 'SmartContract', 'Standard')) {
            const { root, ...rest } = item as any;
            console.log('before handleadd 1')
            handleAdd({ ...root, versions: [rest] } as T);
        }
        // Otherwise, just call handleAdd
        else handleAdd(item as T);
        onClose();
    }, [handleAdd, objectType, onClose]);
    const handleCreateClose = useCallback(() => {
        setIsCreateOpen(false);
    }, [setIsCreateOpen]);
    // Share dialog (for users)
    const [shareDialogOpen, setShareDialogOpen] = useState(false);
    const closeShareDialog = useCallback(() => { setShareDialogOpen(false) }, []);
    // Handle opening either create or share dialog
    const handleCreateOrShareOpen = useCallback(() => {
        if (objectType === 'User') setShareDialogOpen(true);
        else setIsCreateOpen(true);
    }, [objectType]);


    // If item selected from search, query for full data
    const [getItem, { data: itemData }] = useLazyQuery<T, FindByIdInput, string>({ query, endpoint });
    const queryingRef = useRef(false);
    const fetchFullData = useCallback((item: T) => {
        if (!endpoint || !query) return false;
        // Query for full item data, if not already known (would be known if the same item was selected last time)
        if (itemData && itemData[endpoint].id === item.id) {
            console.log('before handleadd 2')
            handleAdd(itemData[endpoint]);
            onClose();
        } else {
            queryingRef.current = true;
            getItem({ variables: { id: item.id } });
        }
        // Return false so the list item does not navigate
        return false;
    }, [endpoint, query, itemData, handleAdd, onClose, getItem]);
    useEffect(() => {
        if (!endpoint) return;
        if (itemData && itemData[endpoint] && queryingRef.current) {
            console.log('before handleadd 3')
            handleAdd(itemData[endpoint]);
            onClose();
        }
        queryingRef.current = false;
    }, [handleAdd, onClose, handleCreateClose, itemData, endpoint]);

    return (
        <Dialog
            open={isOpen}
            onClose={onClose}
            scroll="body"
            aria-labelledby={titleAria}
            sx={{
                zIndex,
                '& .MuiDialogContent-root': {
                    overflow: 'visible',
                    minWidth: 'min(600px, 100%)',
                },
                '& .MuiDialog-paperScrollBody': {
                    overflow: 'visible',
                    background: palette.background.default,
                    margin: { xs: 0, sm: 2, md: 4 },
                    maxWidth: { xs: '100%!important', sm: 'calc(100% - 64px)' },
                    minHeight: { xs: '100vh', sm: 'auto' },
                    display: { xs: 'block', sm: 'inline-block' },
                },
                // Remove ::after element that is added to the dialog
                '& .MuiDialog-container::after': {
                    content: 'none',
                },
            }}
        >
            {/* Invite user dialog */}
            <ShareSiteDialog
                onClose={closeShareDialog}
                open={shareDialogOpen}
                zIndex={zIndex + 1}
            />
            {/* Popup for creating a new item */}
            {CreateView && <BaseObjectDialog
                onAction={handleCreateClose}
                open={isCreateOpen}
                zIndex={zIndex + 1}
            >
                <CreateView
                    onCreated={handleCreated as any}
                    onCancel={handleCreateClose}
                    session={session}
                    zIndex={zIndex + 1}
                />
            </BaseObjectDialog>}
            <DialogTitle
                ariaLabel={titleAria}
                title={t(`Add${objectType.replace('Version', '')}` as CommonKey)}
                helpText={helpText}
                onClose={onClose}
            />
            <Stack direction="column" spacing={2}>
                <Stack direction="row" alignItems="center" justifyContent="center">
                    <Typography component="h2" variant="h4">{t(objectType, { count: 2 })}</Typography>
                    <Tooltip title={t(`AddNew`)} placement="top">
                        <IconButton
                            size="medium"
                            onClick={handleCreateOrShareOpen}
                            sx={{ padding: 1 }}
                        >
                            <AddIcon fill={palette.secondary.main} width='1.5em' height='1.5em' />
                        </IconButton>
                    </Tooltip>
                </Stack>
                <SearchList
                    id={`${objectType}-select-or-create-list`}
                    beforeNavigation={fetchFullData}
                    searchType={objectType as unknown as SearchType}
                    searchPlaceholder={t(`SelectExisting${objectType}`)}
                    session={session}
                    take={20}
                    where={where ?? { userId }}
                    zIndex={zIndex}
                />
            </Stack>
        </Dialog>
    )
}