// Used to display popular/search results of a particular object type
import { Box, LinearProgress, ListItem, ListItemButton, ListItemText, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import { RunListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo } from 'react';
import { APP_LINKS, RunSortBy, StarFor } from '@shared/consts';
import { useLocation } from '@shared/route';
import { TagList, TextLoading } from '..';
import { displayDate, getTranslation, LabelledSortOption, labelledSortOptions, listItemColor } from 'utils';
import { RunStatus } from 'graphql/generated/globalTypes';
import { RoutineIcon } from '@shared/icons';
import { StarButton } from 'components/buttons';

// Color options for profile picture
// [background color, silhouette color]
const colorOptions: [string, string][] = [
    ["#197e2c", "#b5ffc4"],
    ["#b578b6", "#fecfea"],
    ["#4044d6", "#e1c7f3"],
    ["#d64053", "#fbb8c5"],
    ["#d69440", "#e5d295"],
    ["#40a4d6", "#79e0ef"],
    ["#6248e4", "#aac3c9"],
    ["#8ec22c", "#cfe7b4"],
]

function CompletionBar(props) {
    return (
        <Box sx={{ display: 'flex', alignItems: 'center' }}>
            <Box sx={{ width: '100%', mr: 1 }}>
                <LinearProgress variant={props.variant} {...props} sx={{ borderRadius: 1, height: 8 }} />
            </Box>
            <Box sx={{ minWidth: 35 }}>
                <Typography variant="body2" color="text.secondary">{`${Math.round(
                    props.value,
                )}%`}</Typography>
            </Box>
        </Box>
    );
}

export function RunListItem({
    data,
    index,
    loading,
    onClick,
    session,
    tooltip = 'View details',
}: RunListItemProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const profileColors = useMemo(() => colorOptions[Math.floor(Math.random() * colorOptions.length)], []);

    const { canStar, name, percentComplete, startedAt, completedAt } = useMemo(() => {
        const routinePermissions = data?.routine?.permissionsRoutine;
        const languages = session?.languages ?? navigator.languages;
        const completedComplexity = data?.completedComplexity ?? null;
        const totalComplexity = data?.routine?.complexity ?? null;
        const percentComplete = data?.status === RunStatus.Completed ? 100 :
            (completedComplexity && totalComplexity) ? 
            Math.min(Math.round(completedComplexity / totalComplexity * 100), 100) : 
            0
        return {
            canStar: routinePermissions?.canStar === true,
            bio: getTranslation(data?.routine, 'bio', languages, true),
            name: data?.title ?? getTranslation(data?.routine, 'name', languages, true),
            percentComplete,
            startedAt: data?.timeStarted ? displayDate(data.timeStarted) : null,
            completedAt: data?.timeCompleted ? displayDate(data.timeCompleted) : null,
        };
    }, [data, session]);

    const handleClick = useCallback((e: any) => {
        // Prevent propagation
        e.stopPropagation();
        // If data not supplied, don't open
        if (!data) return;
        // If onClick provided, call it
        if (onClick) onClick(e, data);
        // Otherwise, navigate to the object's page
        else setLocation(`${APP_LINKS.Routine}/${data.routine?.id ?? ''}?run="${data.id}"`);
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
                        width="50px"
                        minWidth="50px"
                        height="50px"
                        borderRadius='100%'
                        bgcolor={profileColors[0]}
                        justifyContent='center'
                        alignItems='center'
                        sx={{
                            display: { xs: 'none', sm: 'flex' },
                        }}
                    >
                        <RoutineIcon fill={profileColors[1]} width='80%' height='80%' />
                    </Box>
                    <Stack
                        direction="column"
                        spacing={1}
                        pl={2}
                        sx={{
                            width: '-webkit-fill-available',
                            display: 'grid',
                        }}
                    >
                        {/* Name/Title */}
                        {loading ? <TextLoading /> : <ListItemText
                            primary={name}
                            sx={{ ...multiLineEllipsis(1) }}
                        />}
                        {/* Additional loading bar */}
                        {loading && <TextLoading />}
                        {/* Completed At */}
                        {completedAt && <ListItemText
                            primary={`Completed: ${completedAt}`}
                            sx={{ ...multiLineEllipsis(1), color: palette.text.secondary }}
                        />}
                        {/* Started At */}
                        {!completedAt && startedAt && <ListItemText
                            primary={`Started: ${startedAt}`}
                            sx={{ ...multiLineEllipsis(1), color: palette.text.secondary }}
                        />}
                        {/* Tags */}
                        {Array.isArray(data?.routine?.tags) && (data?.routine as any).tags.length > 0 ? <TagList session={session} parentId={data?.id ?? ''} tags={(data?.routine as any).tags ?? []} /> : null}
                        {/* Progress bar */}
                        <CompletionBar color="secondary" variant={loading ? 'indeterminate' : 'determinate'} value={percentComplete} sx={{ height: '15px' }} />
                    </Stack>
                    {canStar && <StarButton
                        session={session}
                        objectId={data?.id ?? ''}
                        starFor={StarFor.Routine}
                        isStar={data?.routine?.isStarred ?? false}
                        stars={data?.routine?.stars ?? 0}
                        onChange={(isStar: boolean) => { }}
                    />}
                </ListItemButton>
            </ListItem>
        </Tooltip>
    )
}

export const RunSortOptions: LabelledSortOption<RunSortBy>[] = labelledSortOptions(RunSortBy);
export const runDefaultSortOption = RunSortOptions[1];