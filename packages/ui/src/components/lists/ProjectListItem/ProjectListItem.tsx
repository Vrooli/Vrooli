// Used to display popular/search results of a particular object type
import { ListItem, ListItemButton, ListItemText, Stack, Tooltip } from '@mui/material';
import { ProjectListItemProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useCallback } from 'react';
import { APP_LINKS, ProjectSortBy, StarFor, VoteFor } from '@local/shared';
import { useLocation } from 'wouter';
import { StarButton, TagList, UpvoteDownvote } from 'components';
import { voteMutation } from 'graphql/mutation';
import { vote } from 'graphql/generated/vote';
import { useMutation } from '@apollo/client';
import { LabelledSortOption, labelledSortOptions } from 'utils';
import { Project } from 'types';

export function ProjectListItem({
    session,
    index,
    data,
    isOwn = false,
    onClick,
}: ProjectListItemProps) {
    const [, setLocation] = useLocation();
    const [vote] = useMutation<vote>(voteMutation);
    console.log('projectlistitem', data);

    const handleClick = useCallback(() => {
        // If onClick provided, call if
        if (onClick) onClick(data);
        // Otherwise, navigate to the object's page
        else setLocation(`${APP_LINKS.Project}/${data.id}`)
    }, [onClick, setLocation, data]);

    const handleVote = useCallback((e: any, isUpvote: boolean | null) => {
        // Prevent propagation of normal click event
        e.stopPropagation();
        // Send vote mutation
        vote({
            variables: {
                input: {
                    isUpvote,
                    voteFor: VoteFor.Project,
                    forId: data.id
                }
            }
        });
    }, [data.id, vote]);

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
                        score={data.score}
                        isUpvoted={data.isUpvoted}
                        onVote={handleVote}
                    />
                    <Stack direction="column" spacing={1} pl={2} sx={{ width: '-webkit-fill-available' }}>
                        <ListItemText
                            primary={data.name}
                            sx={{ ...multiLineEllipsis(1) }}
                        />
                        <ListItemText
                            primary={data.description}
                            sx={{ ...multiLineEllipsis(2), color: (t) => t.palette.text.secondary }}
                        />
                        {/* Tags */}
                        {Array.isArray(data.tags) && data.tags.length > 0 ? <TagList session={session} parentId={data.id ?? ''} tags={data.tags ?? []} /> : null}
                    </Stack>
                    {
                        isOwn ? null : <StarButton
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
export const projectOptionLabel = (o: Project) => o.name ?? '';