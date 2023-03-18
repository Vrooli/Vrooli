import {
    IconButton,
    Stack,
    Tooltip,
    Typography,
    useTheme
} from '@mui/material';
import { FindByIdInput } from '@shared/consts';
import { AddIcon } from '@shared/icons';
import { removeSearchParams, useLocation } from '@shared/route';
import { CommonKey } from '@shared/translations';
import { isOfType } from '@shared/utils';
import { useCustomLazyQuery } from 'api/hooks';
import { BaseObjectDialog } from 'components/dialogs/BaseObjectDialog/BaseObjectDialog';
import { DialogTitle } from 'components/dialogs/DialogTitle/DialogTitle';
import { LargeDialog } from 'components/dialogs/LargeDialog/LargeDialog';
import { ShareSiteDialog } from 'components/dialogs/ShareSiteDialog/ShareSiteDialog';
import { SearchList } from 'components/lists/SearchList/SearchList';
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { lazily } from 'react-lazily';
import { getCurrentUser } from 'utils/authentication/session';
import { SearchType, searchTypeToParams } from 'utils/search/objectToSearch';
import { SearchParams } from 'utils/search/schemas/base';
import { SessionContext } from 'utils/SessionContext';
import { CreateProps } from 'views/objects/types';
import { SelectOrCreateDialogProps, SelectOrCreateObject, SelectOrCreateObjectType } from '../types';
const { ApiCreate } = lazily(() => import('../../../../views/objects/api/ApiCreate/ApiCreate'));
const { NoteCreate } = lazily(() => import('../../../../views/objects/note/NoteCreate/NoteCreate'));
const { OrganizationCreate } = lazily(() => import('../../../../views/objects/organization/OrganizationCreate/OrganizationCreate'));
const { ProjectCreate } = lazily(() => import('../../../../views/objects/project/ProjectCreate/ProjectCreate'));
const { RoutineCreate } = lazily(() => import('../../../../views/objects/routine/RoutineCreate/RoutineCreate'));
const { SmartContractCreate } = lazily(() => import('../../../../views/objects/smartContract/SmartContractCreate/SmartContractCreate'));
const { StandardCreate } = lazily(() => import('../../../../views/objects/standard/StandardCreate/StandardCreate'));

type CreateViewTypes = ({
    [K in SelectOrCreateObjectType]: K extends (`${string}Version` | 'User') ?
    never :
    K
})[SelectOrCreateObjectType]

/**
 * Maps SelectOrCreateObject types to create components (excluding "User" and types that end with 'Version')
 */
const createMap: { [K in CreateViewTypes]: (props: CreateProps<any>) => JSX.Element } = {
    Api: ApiCreate,
    Note: NoteCreate,
    Organization: OrganizationCreate,
    Project: ProjectCreate,
    Routine: RoutineCreate,
    SmartContract: SmartContractCreate,
    Standard: StandardCreate,
}

export const SelectOrCreateDialog = <T extends SelectOrCreateObject>({
    handleAdd,
    handleClose,
    help,
    isOpen,
    objectType,
    where,
    zIndex,
}: SelectOrCreateDialogProps<T>) => {
    console.log('selectorcreate 1', objectType);
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);
    const { helpText, titleId } = useMemo(() => {
        return {
            helpText: help ?? t('SelectOrCreateDialogHelp', { objectType: t(objectType, { count: 1, defaultValue: objectType }) }),
            titleId: `select-or-create-${objectType}-dialog-title`,
        };
    }, [help, objectType, t]);
    const CreateView = useMemo<((props: CreateProps<any>) => JSX.Element) | null>(() =>
        objectType === 'User' ? null : (createMap as any)[objectType.replace('Version', '')], [objectType]);

    const [{ advancedSearchSchema, query }, setSearchParams] = useState<Partial<SearchParams>>({});
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
    const [getItem, { data: itemData }] = useCustomLazyQuery<T, FindByIdInput>(query);
    const queryingRef = useRef(false);
    const fetchFullData = useCallback((item: T) => {
        if (!query) return false;
        // Query for full item data, if not already known (would be known if the same item was selected last time)
        if (itemData && itemData.id === item.id) {
            console.log('before handleadd 2')
            handleAdd(itemData);
            onClose();
        } else {
            queryingRef.current = true;
            getItem({ variables: { id: item.id } });
        }
        // Return false so the list item does not navigate
        return false;
    }, [query, itemData, handleAdd, onClose, getItem]);
    useEffect(() => {
        if (!query) return;
        if (itemData && queryingRef.current) {
            console.log('before handleadd 3')
            handleAdd(itemData);
            onClose();
        }
        queryingRef.current = false;
    }, [handleAdd, onClose, handleCreateClose, itemData, query]);

    return (
        <LargeDialog
            id="select-or-create-dialog"
            isOpen={isOpen}
            onClose={onClose}
            titleId={titleId}
            zIndex={zIndex}
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
                    // onCreated={handleCreated as any}
                    // onCancel={handleCreateClose}
                    zIndex={zIndex + 1}
                />
            </BaseObjectDialog>}
            <DialogTitle
                id={titleId}
                title={t(`Add${objectType.replace('Version', '')}` as CommonKey)}
                helpText={helpText}
                onClose={onClose}
            />
            <Stack direction="column" spacing={2}>
                <Stack direction="row" alignItems="center" justifyContent="center">
                    <Typography component="h2" variant="h4">{t(objectType as CommonKey, { count: 2 })}</Typography>
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
                    searchPlaceholder={`SelectExisting${objectType}` as CommonKey}
                    take={20}
                    where={where ?? { userId }}
                    zIndex={zIndex}
                />
            </Stack>
        </LargeDialog>
    )
}