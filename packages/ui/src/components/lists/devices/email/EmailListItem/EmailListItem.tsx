// Used to display popular/search results of a particular object type
import { Box, IconButton, ListItem, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { EmailListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback } from 'react';
import { CompleteIcon, DeleteIcon } from '@shared/icons';

const Status = {
    NotVerified: '#a71c2d', // Red
    Verified: '#19972b', // Green
}

export function EmailListItem({
    handleDelete,
    handleVerify,
    index,
    data,
}: EmailListItemProps) {
    const { palette } = useTheme();

    const onDelete = useCallback(() => {
        handleDelete(data);
    }, [data, handleDelete]);

    const onVerify = useCallback(() => {
        handleVerify(data);
    }, [data, handleVerify]);

    return (
        <ListItem
            disablePadding
            sx={{
                display: 'flex',
                padding: 1,
            }}
        >
            {/* Left informational column */}
            <Stack direction="column" spacing={1} pl={2} sx={{ marginRight: 'auto' }}>
                <ListItemText
                    primary={data.emailAddress}
                    sx={{ ...multiLineEllipsis(1) }}
                />
                {/* Verified indicator */}
                <Box sx={{
                    borderRadius: 1,
                    border: `2px solid ${data.verified ? Status.Verified : Status.NotVerified}`,
                    color: data.verified ? Status.Verified : Status.NotVerified,
                    height: 'fit-content',
                    fontWeight: 'bold',
                    marginTop: 'auto',
                    marginBottom: 'auto',
                    textAlign: 'center',
                    padding: 0.25,
                    width: 'fit-content',
                }}>
                    {data.verified ? "Verified" : "Not Verified"}
                </Box>
            </Stack>
            {/* Right action buttons */}
            <Stack direction="row" spacing={1}>
                {!data.verified && <Tooltip title="Resend email verification">
                    <IconButton
                        onClick={onVerify}
                    >
                        <CompleteIcon fill={Status.NotVerified } />
                    </IconButton>
                </Tooltip>}
                <Tooltip title="Delete Email">
                    <IconButton
                        onClick={onDelete}
                    >
                        <DeleteIcon fill={palette.secondary.main} />
                    </IconButton>
                </Tooltip>
            </Stack>
        </ListItem>
    )
}