import { Box, Button, IconButton, List, ListItem, ListItemText, Menu, Stack, TextField, Typography, useTheme } from '@mui/material';
import { MouseEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { StandardSelectSwitchProps } from '../types';
import { organizationsQuery } from 'graphql/query';
import { organizations, organizationsVariables } from 'graphql/generated/organizations';
import { APP_LINKS, OrganizationSortBy } from '@local/shared';
import { useLazyQuery } from '@apollo/client';
import { ListOrganization } from 'types';
import { noSelect } from 'styles';
import { Close as CloseIcon } from '@mui/icons-material';
import {
    Edit as CustomIcon,
    Link as StandardIcon,
} from '@mui/icons-material';
import { getTranslation, getUserLanguages } from 'utils';
import { useLocation } from 'wouter';
import { owns } from 'utils/authentication';

const grey = {
    400: '#BFC7CF',
    800: '#2F3A45',
};

export function StandardSelectSwitch({
    session,
    selected,
    onChange,
    disabled,
    ...props
}: StandardSelectSwitchProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const languages = useMemo(() => getUserLanguages(session), [session])

    // Query for organizations the user is a member of, including ones where 
    // the user doesn't have permissions to perform this action
    const [getOrganizationsData, { data: organizationsData, loading }] = useLazyQuery<organizations, organizationsVariables>(organizationsQuery, {
        variables: {
            input: {
                sortBy: OrganizationSortBy.DateUpdatedAsc,
                userId: session?.id,
            }
        }
    });
    const organizations = useMemo(() => organizationsData?.organizations?.edges?.map(e => e.node) ?? [] as ListOrganization[], [organizationsData]);
    useEffect(() => {
        if (session?.id) {
            getOrganizationsData()
        }
    }, [getOrganizationsData, session])

    const [menuAnchorEl, setMenuAnchorEl] = useState<any>(null);
    const handleClick = useCallback((ev: MouseEvent<any>) => {
        if (disabled) return;
        if (Boolean(selected)) {
            onChange(null);
        } else {
            setMenuAnchorEl(ev.currentTarget);
            ev.preventDefault();
        }
    }, [disabled, selected, onChange]);
    const closeMenu = useCallback(() => {
        setMenuAnchorEl(null);
        setSearch('');
    }, []);

    const [search, setSearch] = useState<string>('');
    const handleSearchChange = useCallback((ev: any) => {
        setSearch(ev.target.value);
    }, []);

    const organizationListItems: JSX.Element[] = useMemo(() => {
        const filtered = organizations?.filter((o: ListOrganization) => getTranslation(o, 'name', languages, true)?.toLowerCase()?.includes(search.toLowerCase()) ?? '');
        return filtered?.map((o: ListOrganization, index) => {
            const canSelect: boolean = owns(o.role);
            return (
                <ListItem
                    key={index}
                    button
                    onClick={() => { onChange(o); closeMenu(); }}
                    sx={{
                        background: canSelect ? 'white' : '#F5F5F5',
                        cursor: canSelect ? 'pointer' : 'default',
                    }}
                >
                    <ListItemText primary={getTranslation(o, 'name', languages, true)} sx={{ marginRight: 2 }} />
                    {!canSelect && <Typography color="error">Not an admin</Typography>}
                </ListItem>
            )
        });
    }, [organizations, languages, search, onChange, closeMenu]);

    const Icon = useMemo(() => Boolean(selected) ? StandardIcon : CustomIcon, [selected]);

    return (
        <>
            {/* Popup menu to select organization */}
            <Menu
                id={'select-organization-menu'}
                disableScrollLock={true}
                autoFocus={true}
                open={Boolean(menuAnchorEl)}
                anchorEl={menuAnchorEl}
                anchorOrigin={{
                    vertical: 'bottom',
                    horizontal: 'center',
                }}
                transformOrigin={{
                    vertical: 'top',
                    horizontal: 'center',
                }}
                onClose={(e) => { closeMenu() }}
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
                        background: palette.primary.dark
                    }}
                >
                    <Typography
                        variant="h6"
                        textAlign="center"
                        sx={{
                            width: '-webkit-fill-available',
                            color: palette.primary.contrastText,
                        }}
                    >
                        Select Standard
                    </Typography>
                    <IconButton
                        edge="end"
                        onClick={(e) => { closeMenu() }}
                    >
                        <CloseIcon sx={{ fill: palette.primary.contrastText }} />
                    </IconButton>
                </Box>
                <TextField
                    fullWidth
                    id="filter-organizations"
                    label="Filter"
                    value={search}
                    onChange={handleSearchChange}
                    sx={{ marginLeft: 2, paddingRight: 4, marginTop: 2 }}
                />
                {organizationListItems.length > 0 ? (
                    <List>
                        {organizationListItems}
                    </List>
                ) : (
                    <Box sx={{
                        display: 'block',
                        textAlign: 'center',
                        paddingTop: 2,
                        paddingBottom: 2
                    }}>
                        <Typography variant="body1" color="gray">Not matching organizations</Typography>
                    </Box>
                )
                }
                <Button
                    color="secondary"
                    onClick={() => setLocation(`${APP_LINKS.Organization}/add`, { replace: true })}
                    sx={{
                        display: 'flex',
                        marginLeft: 'auto',
                        marginRight: 'auto',
                    }}
                >
                    Create New
                </Button>
            </Menu>
            {/* Main component */}
            <Stack direction="row" spacing={1} justifyContent="center">
                <Typography variant="h6" sx={{ ...noSelect }}>Standard:</Typography>
                <Box component="span" sx={{
                    display: 'inline-block',
                    position: 'relative',
                    width: '64px',
                    height: '36px',
                    padding: '8px',
                }}>
                    {/* Track */}
                    <Box component="span" sx={{
                        backgroundColor: palette.mode === 'dark' ? grey[800] : grey[400],
                        borderRadius: '16px',
                        width: '100%',
                        height: '65%',
                        display: 'block',
                    }}>
                        {/* Thumb */}
                        <IconButton sx={{
                            backgroundColor: palette.secondary.main,
                            display: 'inline-flex',
                            width: '30px',
                            height: '30px',
                            position: 'absolute',
                            top: 0,
                            transition: 'transform 150ms cubic-bezier(0.4, 0, 0.2, 1)',
                            transform: `translateX(${Boolean(selected) ? '24' : '0'}px)`,
                        }}>
                            <Icon sx={{
                                position: 'absolute',
                                display: 'block',
                                fill: 'white',
                                borderRadius: '8px',
                            }} />
                        </IconButton>
                    </Box>
                    {/* Input */}
                    <input
                        type="checkbox"
                        checked={Boolean(selected)}
                        disabled={disabled}
                        aria-label="user-organization-toggle"
                        onClick={handleClick}
                        style={{
                            position: 'absolute',
                            width: '100%',
                            height: '100%',
                            top: '0',
                            left: '0',
                            opacity: '0',
                            zIndex: '1',
                            margin: '0',
                            cursor: 'pointer',
                        }} />
                </Box >
                <Typography variant="h6" sx={{ ...noSelect }}>{Boolean(selected) ? getTranslation(selected, 'name', languages, true) : 'Custom'}</Typography>
            </Stack>
        </>
    )
}
