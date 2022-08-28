import {
    IconButton,
    List,
    ListItem,
    ListItemIcon,
    ListItemText,
    Menu,
    useTheme
} from '@mui/material';
import { HelpButton, MenuTitle } from 'components';
import { useMemo } from 'react';
import { ListMenuProps } from '../types';

const titleAria = 'list-menu-title';

export function ListMenu<T>({
    id,
    anchorEl,
    onSelect,
    onClose,
    title = 'Select Item',
    data,
    zIndex,
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
                zIndex,
                '& .MuiMenu-paper': {
                    background: palette.background.default
                },
                '& .MuiMenu-list': {
                    paddingTop: '0',
                }
            }}
        >
            <MenuTitle
                ariaLabel={titleAria}
                title={title}
                onClose={() => { onClose() }}
            />
            <List>
                {items}
            </List>
        </Menu>
    )
}