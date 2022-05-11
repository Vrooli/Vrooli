import {
    Box,
    IconButton,
    List,
    ListItem,
    ListItemIcon,
    ListItemText,
    Menu,
    Typography,
    useTheme
} from '@mui/material';
import { HelpButton } from 'components';
import { useMemo } from 'react';
import { noSelect } from 'styles';
import { ListMenuProps } from '../types';
import { Close as CloseIcon } from '@mui/icons-material';

export function ListMenu<T>({
    id,
    anchorEl,
    onSelect,
    onClose,
    title = 'Select Item',
    data,
}: ListMenuProps<T>) {
    const { palette } = useTheme();

    const open = Boolean(anchorEl);

    const items = useMemo(() => data?.map(({ label, value, Icon, iconColor, preview, helpData }, index) => {
        const itemText = <ListItemText primary={label} secondary={preview ? 'Coming Soon' : null} sx={{
            // Style secondary
            '& .MuiListItemText-secondary': {
                color: 'red',
            },
        }}/>;
        const itemIcon = Icon ? (
            <ListItemIcon>
                <Icon sx={{ fill: iconColor || 'default' }} />
            </ListItemIcon>
        ) : null;
        const helpIcon = helpData ? (
            <IconButton edge="end" onClick={(e) => e.stopPropagation()} >
                <HelpButton {...helpData} />
            </IconButton>
        ) : null;
        return (
            <ListItem disabled={preview} button onClick={() => { onSelect(value); onClose(); }} key={index}>
                {itemIcon}
                {itemText}
                {helpIcon}
            </ListItem>
        )
    }), [data, onClose, onSelect])

    return (
        <Menu
            id={id}
            disableScrollLock={true}
            autoFocus={true}
            open={open}
            anchorEl={anchorEl}
            anchorOrigin={{
                vertical: 'bottom',
                horizontal: 'center',
            }}
            transformOrigin={{
                vertical: 'top',
                horizontal: 'center',
            }}
            onClose={(e) => { onClose() }}
            sx={{
                '& .MuiMenu-paper': {
                    background: palette.background.paper
                },
                '& .MuiMenu-list': {
                    paddingTop: '0',
                }
            }}
        >
            <Box
                sx={{
                    ...noSelect,
                    display: 'flex',
                    alignItems: 'center',
                    padding: 1,
                    background: palette.mode === 'light' ? palette.primary.dark : palette.secondary.dark,
                }}
            >
                <Typography
                    variant="h6"
                    textAlign="center"
                    sx={{
                        width: '-webkit-fill-available',
                        color: palette.mode === 'light' ? palette.primary.contrastText : palette.secondary.contrastText,
                    }}
                >
                    {title}
                </Typography>
                <IconButton
                    edge="end"
                    onClick={(e) => { onClose() }}
                >
                    <CloseIcon sx={{ fill: palette.primary.contrastText }} />
                </IconButton>
            </Box>
            <List>
                {items}
            </List>
        </Menu>
    )
}