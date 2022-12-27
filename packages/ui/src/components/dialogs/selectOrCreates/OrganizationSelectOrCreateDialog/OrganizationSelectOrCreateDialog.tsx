import {
    Dialog,
    IconButton,
    Stack,
    Tooltip,
    Typography,
    useTheme
} from '@mui/material';
import { BaseObjectDialog, DialogTitle } from 'components';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { OrganizationSelectOrCreateDialogProps } from '../types';
import { IWrap, Wrap } from 'types';
import { SearchList } from 'components/lists';
import { useLazyQuery } from '@apollo/client';
import { OrganizationCreate } from 'components/views/Organization/OrganizationCreate/OrganizationCreate';
import { SearchType, organizationSearchSchema, removeSearchParams } from 'utils';
import { useLocation } from '@shared/route';
import { AddIcon } from '@shared/icons';
import { getCurrentUser } from 'utils/authentication';
import { FindByIdInput } from '@shared/consts';

const helpText =
    `This dialog allows you to connect a new or existing organization to an object.

This will assign ownership of the object to that organization, instead of your own account.

If you do not own the organization, the owner will have to approve of the request.`

const titleAria = "select-or-create-organization-dialog-title"

export const OrganizationSelectOrCreateDialog = ({
    handleAdd,
    handleClose,
    isOpen,
    session,
    zIndex,
}: OrganizationSelectOrCreateDialogProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    /**
     * Before closing, remove all URL search params for advanced search
     */
    const onClose = useCallback(() => {
        // Clear search params
        removeSearchParams(setLocation, [
            ...organizationSearchSchema.fields.map(f => f.fieldName),
            'advanced',
            'sort',
            'time',
        ]);
        handleClose();
    }, [handleClose, setLocation]);

    // Create new organization dialog
    const [isCreateOpen, setIsCreateOpen] = useState(false);
    const handleCreateOpen = useCallback(() => { setIsCreateOpen(true); }, [setIsCreateOpen]);
    const handleCreated = useCallback((organization: Organization) => {
        setIsCreateOpen(false);
        handleAdd(organization);
        onClose();
    }, [handleAdd, onClose]);
    const handleCreateClose = useCallback(() => {
        setIsCreateOpen(false);
    }, [setIsCreateOpen]);

    // If organization selected from search, query for full data
    const [getOrganization, { data: organizationData }] = useLazyQuery<Wrap<Organization, 'organization'>, IWrap<FindByIdInput>>(organizationQuery);
    const queryingRef = useRef(false);
    const fetchFullData = useCallback((organization: Organization) => {
        // Query for full organization data, if not already known (would be known if the same organization was selected last time)
        if (organizationData?.organization?.id === organization.id) {
            handleAdd(organizationData?.organization);
            onClose();
        } else {
            queryingRef.current = true;
            getOrganization({ variables: { input: { id: organization.id } } });
        }
        // Return false so the list item does not navigate
        return false;
    }, [getOrganization, organizationData, handleAdd, onClose]);
    useEffect(() => {
        if (organizationData?.organization && queryingRef.current) {
            handleAdd(organizationData.organization);
            onClose();
        }
        queryingRef.current = false;
    }, [handleAdd, onClose, handleCreateClose, organizationData]);

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
            {/* Popup for creating a new organization */}
            <BaseObjectDialog
                onAction={handleCreateClose}
                open={isCreateOpen}
                zIndex={zIndex + 1}
            >
                <OrganizationCreate
                    onCreated={handleCreated}
                    onCancel={handleCreateClose}
                    session={session}
                    zIndex={zIndex + 1}
                />
            </BaseObjectDialog>
            <DialogTitle
                ariaLabel={titleAria}
                title={'Add Organization'}
                helpText={helpText}
                onClose={onClose}
            />
            <Stack direction="column" spacing={2}>
                <Stack direction="row" alignItems="center" justifyContent="center">
                    <Typography component="h2" variant="h4">Organizations</Typography>
                    <Tooltip title="Add new" placement="top">
                        <IconButton
                            size="medium"
                            onClick={handleCreateOpen}
                            sx={{ padding: 1 }}
                        >
                            <AddIcon fill={palette.secondary.main} width='1.5em' height='1.5em' />
                        </IconButton>
                    </Tooltip>
                </Stack>
                <SearchList
                    id="organization-select-or-create-list"
                    itemKeyPrefix='organization-list-item'
                    noResultsText={"None found. Maybe you should create one?"}
                    beforeNavigation={fetchFullData}
                    searchType={SearchType.Organization}
                    searchPlaceholder={'Select existing organizations...'}
                    session={session}
                    take={20}
                    where={{ userId }}
                    zIndex={zIndex}
                />
            </Stack>
        </Dialog>
    )
}