// Used to display popular/search results of a particular object type
import { Box, IconButton, ListItem, ListItemButton, ListItemText, Stack, Tooltip } from '@mui/material';
import { WalletListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo } from 'react';
import {
    Check as VerifyIcon,
    Delete as DeleteIcon,
} from '@mui/icons-material';

export function WalletListItem({
    handleDelete,
    handleUpdate,
    handleVerify,
    index,
    data,
}: WalletListItemProps) {
    const isNamed = useMemo(() => data.name && data.name.length > 0, [data.name]);

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
                background: index % 2 === 0 ? '#c8d6e9' : '#e9e9e9',
                color: 'black',
            }}
        >
            <ListItemButton component="div">
                {/* Left informational column */}
                <Stack direction="column" spacing={1} pl={2} sx={{ marginRight: 'auto' }}>
                    <Stack direction="row" spacing={1}>
                        {/* Name (or publich address if not name) */}
                        <ListItemText
                            primary={isNamed ? data.name : data.publicAddress}
                            sx={{ ...multiLineEllipsis(1) }}
                        />
                        {/* Bio/Description */}
                        {isNamed && <ListItemText
                            primary={data.publicAddress}
                            sx={{ ...multiLineEllipsis(1), color: (t) => t.palette.text.secondary }}
                        />}
                    </Stack>
                    {/* Verified indicator */}
                    <Box sx={{

                    }}>
                        {data.verified ? "Verified" : "Not Verified"}
                    </Box>
                </Stack>
                {/* Right action buttons */}
                <Stack direction="row" spacing={1}>
                    {!data.verified && <Tooltip title="Verify Wallet">
                        <IconButton
                            onClick={onVerify}
                        >
                            <VerifyIcon sx={{ fill: (t) => t.palette.secondary.main }} />
                        </IconButton>
                    </Tooltip>}
                    <Tooltip title="Delete Wallet">
                        <IconButton
                            onClick={onDelete}
                        >
                            <DeleteIcon sx={{ fill: (t) => t.palette.secondary.main }} />
                        </IconButton>
                    </Tooltip>
                </Stack>
            </ListItemButton>
        </ListItem>
    )
}