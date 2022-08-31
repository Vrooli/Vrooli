import {
    Dialog,
    DialogContent,
    IconButton,
    Stack,
    Tooltip,
    Typography,
    useTheme
} from '@mui/material';
import { DialogTitle, ShareDialog } from 'components';
import { useCallback, useEffect, useState } from 'react';
import { UserSelectDialogProps } from '../types';
import { User } from 'types';
import { SearchList } from 'components/lists';
import { userQuery } from 'graphql/query';
import { useLazyQuery } from '@apollo/client';
import { user, userVariables } from 'graphql/generated/user';
import { parseSearchParams, stringifySearchParams, SearchType, userSearchSchema } from 'utils';
import { useLocation } from '@shared/route';
import { AddIcon } from '@shared/icons';

const helpText =
    `This dialog allows you to connect a user to an object.

The user will have to approve of the request.`

const titleAria = "select-or-invite-user-dialog-title"

export const UserSelectDialog = ({
    handleAdd,
    handleClose,
    isOpen,
    session,
    zIndex,
}: UserSelectDialogProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    /**
     * Before closing, remove all URL search params for advanced search
     */
    const onClose = useCallback(() => {
        // Find all search fields
        const searchFields = [
            ...userSearchSchema.fields.map(f => f.fieldName),
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

    // Share dialog
    const [shareDialogOpen, setShareDialogOpen] = useState(false);
    const openShareDialog = useCallback(() => { setShareDialogOpen(true) }, [setShareDialogOpen]);
    const closeShareDialog = useCallback(() => { setShareDialogOpen(false) }, []);

    // If user selected from search, query for full data
    const [getUser, { data: userData }] = useLazyQuery<user, userVariables>(userQuery);
    const handleUserSelect = useCallback((user: User) => {
        // Query for full user data, if not already known (would be known if the same user was selected last time)
        if (userData?.user?.id === user.id) {
            handleAdd(userData?.user);
            onClose();
        } else {
            getUser({ variables: { input: { id: user.id } } });
        }
    }, [getUser, userData, handleAdd, onClose]);
    useEffect(() => {
        if (userData?.user) {
            handleAdd(userData.user);
            onClose();
        }
    }, [handleAdd, onClose, userData]);

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
            {/* Invite user dialog */}
            <ShareDialog
                onClose={closeShareDialog}
                open={shareDialogOpen}
                zIndex={200}
            />
            <DialogTitle
                ariaLabel={titleAria}
                title={'Add User'}
                helpText={helpText}
                onClose={onClose}
            />
            <DialogContent>
                <Stack direction="column" spacing={2}>
                    <Stack direction="row" alignItems="center" justifyContent="center">
                        <Typography component="h2" variant="h4">Users</Typography>
                        <Tooltip title="Add new" placement="top">
                            <IconButton
                                size="large"
                                onClick={openShareDialog}
                                sx={{ padding: 1 }}
                            >
                                <AddIcon color="secondary" width='1.5em' height='1.5em' />
                            </IconButton>
                        </Tooltip>
                    </Stack>
                    <SearchList
                        itemKeyPrefix='user-list-item'
                        noResultsText={"No results. Maybe you should invite them?"}
                        onObjectSelect={(newValue) => handleUserSelect(newValue)}
                        searchType={SearchType.User}
                        searchPlaceholder={'Select existing users...'}
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