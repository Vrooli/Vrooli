import {
    Dialog,
    DialogContent,
    IconButton,
    Stack,
    Tooltip,
    Typography,
    useTheme
} from '@mui/material';
import { BaseObjectDialog, DialogTitle } from 'components';
import { useCallback, useEffect, useState } from 'react';
import { OrganizationSelectOrCreateDialogProps } from '../types';
import {
    Add as CreateIcon,
} from '@mui/icons-material';
import { Organization } from 'types';
import { SearchList } from 'components/lists';
import { organizationQuery } from 'graphql/query';
import { useLazyQuery } from '@apollo/client';
import { organization, organizationVariables } from 'graphql/generated/organization';
import { OrganizationCreate } from 'components/views/Organization/OrganizationCreate/OrganizationCreate';
import { parseSearchParams, stringifySearchParams, SearchType, organizationSearchSchema } from 'utils';
import { useLocation } from '@shared/route';

const helpText =
    `This dialog allows you to connect a new or existing organization to an object.

This will assign ownership of the object to that organization, instead of your own account.

You must be an admin or owner of the organization for it to appear in the list`

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

    /**
     * Before closing, remove all URL search params for advanced search
     */
    const onClose = useCallback(() => {
        // Find all search fields
        const searchFields = [
            ...organizationSearchSchema.fields.map(f => f.fieldName),
            'advanced',
            'sort',
            'time',
        ];
        // Find current search params
        const params = parseSearchParams(window.location.search);
        // Remove all search params that are advanced search fields
        Object.keys(params).forEach(key => {
            if (searchFields.includes(key)) {
                delete params[key];
            }
        });
        // Update URL
        setLocation(stringifySearchParams(params), { replace: true });
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
    const [getOrganization, { data: organizationData }] = useLazyQuery<organization, organizationVariables>(organizationQuery);
    const handleOrganizationSelect = useCallback((organization: Organization) => {
        // Query for full organization data, if not already known (would be known if the same organization was selected last time)
        if (organizationData?.organization?.id === organization.id) {
            handleAdd(organizationData?.organization);
            onClose();
        } else {
            getOrganization({ variables: { input: { id: organization.id } } });
        }
    }, [getOrganization, organizationData, handleAdd, onClose]);
    useEffect(() => {
        if (organizationData?.organization) {
            handleAdd(organizationData.organization);
            onClose();
        }
    }, [handleAdd, onClose, handleCreateClose, organizationData]);

    return (
        <Dialog
            open={isOpen}
            onClose={onClose}
            scroll="body"
            aria-labelledby={titleAria}
            sx={{
                zIndex,
                '& .MuiDialogContent-root': { overflow: 'visible', background: palette.background.default },
                '& .MuiDialog-paper': { overflow: 'visible' }
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
            <DialogContent>
                <Stack direction="column" spacing={2}>
                    <Stack direction="row" alignItems="center" justifyContent="center">
                        <Typography component="h2" variant="h4">Organizations</Typography>
                        <Tooltip title="Add new" placement="top">
                            <IconButton
                                size="large"
                                onClick={handleCreateOpen}
                                sx={{ padding: 1 }}
                            >
                                <CreateIcon color="secondary" sx={{ width: '1.5em', height: '1.5em' }} />
                            </IconButton>
                        </Tooltip>
                    </Stack>
                    <SearchList
                        itemKeyPrefix='organization-list-item'
                        noResultsText={"None found. Maybe you should create one?"}
                        onObjectSelect={(newValue) => handleOrganizationSelect(newValue)}
                        searchType={SearchType.Organization}
                        searchPlaceholder={'Select existing organization...'}
                        session={session}
                        take={20}
                        where={{ userId: session?.id }}
                        zIndex={zIndex}
                    />
                </Stack>
            </DialogContent>
        </Dialog>
    )
}