// Used to display popular/search results of a particular object type
import { ListItem, ListItemButton, ListItemText, Stack, Tooltip } from '@mui/material';
import { ProjectListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback, useMemo } from 'react';
import { APP_LINKS, MemberRole, ProjectSortBy, StarFor, VoteFor } from '@local/shared';
import { useLocation } from 'wouter';
import { StarButton, TagList, UpvoteDownvote } from 'components';
import { getTranslation, LabelledSortOption, labelledSortOptions } from 'utils';
import { Project } from 'types';

export function ProjectListItem({
    session,
    index,
    data,
    onClick,
}: ProjectListItemProps) {
    const [, setLocation] = useLocation();
    const canEdit = useMemo<boolean>(() => data?.role ? [MemberRole.Admin, MemberRole.Owner].includes(data.role) : false, [data]);

    const { description, name } = useMemo(() => {
        const languages = session?.languages ?? navigator.languages;
        return {
            description: getTranslation(data, 'description', languages, true),
            name: getTranslation(data, 'name', languages, true),
        }
    }, [data, session]);

    const handleClick = useCallback((e: any) => {
        // If onClick provided, call if
        if (onClick) onClick(e, data);
        // Otherwise, navigate to the object's page
        else setLocation(`${APP_LINKS.Project}/${data.id}`)
    }, [onClick, setLocation, data]);

    return (
        <Tooltip placement="top" title="View details">
            <ListItem
                disablePadding
                onClick={handleClick}
                sx={{
                    display: 'flex',
                    background: index % 2 === 0 ? 'default' : '#e9e9e9',
                }}
            >
                <ListItemButton component="div" onClick={handleClick}>
                    <UpvoteDownvote
                        session={session}
                        objectId={data.id ?? ''}
                        voteFor={VoteFor.Project}
                        isUpvoted={data.isUpvoted}
                        score={data.score}
                        onChange={(isUpvoted: boolean | null) => { }}
                    />
                    <Stack direction="column" spacing={1} pl={2} sx={{ width: '-webkit-fill-available' }}>
                        <ListItemText
                            primary={name}
                            sx={{ ...multiLineEllipsis(1) }}
                        />
                        <ListItemText
                            primary={description}
                            sx={{ ...multiLineEllipsis(2), color: (t) => t.palette.text.secondary }}
                        />
                        {/* Tags */}
                        {Array.isArray(data.tags) && data.tags.length > 0 ? <TagList session={session} parentId={data.id ?? ''} tags={data.tags ?? []} /> : null}
                    </Stack>
                    {
                        canEdit ? null : <StarButton
                            session={session}
                            objectId={data.id ?? ''}
                            starFor={StarFor.Project}
                            isStar={data.isStarred}
                            stars={data.stars}
                            onChange={(isStar: boolean) => { }}
                        />
                    }
                </ListItemButton>
            </ListItem>
        </Tooltip>
    )
}

export const ProjectSortOptions: LabelledSortOption<ProjectSortBy>[] = labelledSortOptions(ProjectSortBy);
export const projectDefaultSortOption = ProjectSortOptions[1];