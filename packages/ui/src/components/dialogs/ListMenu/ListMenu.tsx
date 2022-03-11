import {
    Box,
    IconButton,
    List,
    ListItem,
    ListItemIcon,
    ListItemText,
    Menu,
    Typography
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
    const open = Boolean(anchorEl);

    const items = useMemo(() => data?.map(({ label, value, Icon, iconColor, helpData }, index) => {
        const itemText = <ListItemText primary={label} />;
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
            <ListItem button onClick={() => { console.log('on select', value); onSelect(value); onClose(); }} key={index}>
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
            onClose={(e) => { console.log('on close', e); onClose() }}
            sx={{
                '& .MuiMenu-paper': {
                    background: (t) => t.palette.background.paper
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
                    background: (t) => t.palette.primary.dark
                }}
            >
                <Typography
                    variant="h6"
                    textAlign="center"
                    sx={{
                        width: '-webkit-fill-available',
                        color: (t) => t.palette.primary.contrastText,
                    }}
                >
                    {title}
                </Typography>
                <IconButton
                    edge="end"
                    onClick={(e) => { console.log('on close', e); onClose() }}
                >
                    <CloseIcon sx={{ fill: (t) => t.palette.primary.contrastText }} />
                </IconButton>
            </Box>
            <List>
                {items}
            </List>
        </Menu>
    )
}