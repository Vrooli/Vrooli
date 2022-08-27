import { Box, ListItem, ListItemButton, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { UserListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo } from 'react';
import { APP_LINKS, StarFor } from '@local/shared';
import { useLocation } from 'wouter';
import { ReportButton, StarButton } from '..';
import { getTranslation, listItemColor, placeholderColor } from 'utils';
import { Person as PersonIcon } from '@mui/icons-material';
import { TextLoading } from '../TextLoading/TextLoading';

export const UserListItem = ({
    data,
    index,
    loading,
    onClick,
    session,
    tooltip = 'View details',
}: UserListItemProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const isOwn = useMemo(() => data?.id === session?.id, [data, session]);

    const profileColors = useMemo(() => placeholderColor(), []);

    const { bio, name, reportsCount } = useMemo(() => {
        const languages = session?.languages ?? navigator.languages;
        return {
            bio: getTranslation(data, 'bio', languages, true),
            name: data?.name ?? (data?.handle ? `$${data.handle}` : ''),
            reportsCount: data?.reportsCount ?? 0,
        }
    }, [data, session]);

    const handleClick = useCallback((e: any) => {
        // Prevent propagation
        e.stopPropagation();
        // If data not supplied, don't open
        if (!data) return;
        // If onClick provided, call it
        if (onClick) onClick(e, data);
        // Otherwise, navigate to the user's profile
        else {
            // Prefer using handle if available
            const link = data.handle ?? data.id;
            setLocation(`${APP_LINKS.Profile}/${link}`);
        }
    }, [onClick, data, setLocation]);

    return (
        <Tooltip placement="top" title={tooltip ?? 'View Details'}>
            <ListItem
                disablePadding
                onClick={handleClick}
                sx={{
                    display: 'flex',
                    background: listItemColor(index, palette),
                }}
            >
                <ListItemButton component="div" onClick={handleClick}>
                    <Box
                        alignItems='center'
                        bgcolor={profileColors[0]}
                        borderRadius='100%'
                        height="50px"
                        justifyContent='center'
                        minWidth="50px"
                        width="50px"
                        sx={{
                            display: 'flex',
                        }}
                    >
                        <PersonIcon sx={{
                            fill: profileColors[1],
                            height: '80%',
                            width: '80%',
                        }} />
                    </Box>
                    <Stack direction="column" spacing={1} pl={2} sx={{ width: '-webkit-fill-available' }}>
                        {/* Name/Title and role */}
                        {loading ? <TextLoading /> :
                            (
                                <Stack direction="row" spacing={1} sx={{
                                    overflow: 'auto',
                                }}>
                                    <ListItemText
                                        primary={name}
                                        sx={{
                                            ...multiLineEllipsis(1),
                                            lineBreak: 'anywhere',
                                        }}
                                    />
                                    {isOwn && <ListItemText
                                        primary={`(You)`}
                                        sx={{
                                            display: 'flex',
                                            alignItems: 'center',
                                            width: '100%',
                                            color: palette.mode === 'light' ? '#fa4f4f' : '#f2a7a7',
                                            flex: 200,
                                        }}
                                    />}
                                </Stack>
                            )
                        }
                        {loading ? <TextLoading /> : <ListItemText
                            primary={bio}
                            sx={{ ...multiLineEllipsis(2), color: palette.text.secondary }}
                        />}
                    </Stack>
                    {/* Star/Comment/Report */}
                    <Stack direction="column" spacing={1}>
                        {!isOwn && <StarButton
                            session={session}
                            objectId={data?.id ?? ''}
                            starFor={StarFor.User}
                            isStar={data?.isStarred}
                            stars={data?.stars}
                        />}
                        {!isOwn && reportsCount > 0 && <ReportButton
                            reportsCount={data?.reportsCount ?? 0}
                            object={data}
                        />}
                    </Stack>
                </ListItemButton>
            </ListItem>
        </Tooltip>
    )
}