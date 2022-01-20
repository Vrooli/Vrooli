// Displays a list of resources. If the user can modify the list, 
// it will display options for adding, removing, and sorting
import { Box, Tooltip, Typography } from '@mui/material';
import { ResourceCard, ResourceListItemContextMenu } from 'components';
import { ResourceListProps } from '../types';
import { containerShadow } from 'styles';
import { MouseEvent, useCallback, useMemo, useState } from 'react';
import { Resource } from 'types';

//TODO Temp data for designing card
// Tries to use open graph metadata when fields not specified
const cardData = [
    {
        title: 'Chill Beats',
        description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris',
        url: 'https://www.youtube.com/c/LofiGirl'
    },
    {
        title: 'Code repo',
        url: 'https://github.com/MattHalloran/Vrooli'
    },
    {
        url: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ'
    },
    {
        url: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ'
    },
    {
        url: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ'
    },
    {
        url: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ'
    },
    {
        url: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ'
    },
    {
        url: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ'
    },
    {
        url: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ'
    },
    {
        url: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ'
    },
    {
        url: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ'
    },
    {
        url: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ'
    },
    {
        url: 'http://ogp.me/'
    }
]

export const ResourceList = ({
    title = 'Pinned Resources',
    canEdit = true,
}: ResourceListProps) => {

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const [selected, setSelected] = useState<any | null>(null);
    const contextId = useMemo(() => `resource-context-menu-${selected?.url}`, [selected]);
    const openContext = useCallback((ev: MouseEvent<HTMLButtonElement>, data: any) => {
        console.log('setting context anchor', ev.currentTarget, data);
        setContextAnchor(ev.currentTarget);
        setSelected(data);
        ev.preventDefault();
    }, []);
    const closeContext = useCallback(() => {
        setContextAnchor(null);
        setSelected(null);
    }, []);

    return (
        <Box>
            <ResourceListItemContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                resource={selected}
                onClose={closeContext}
                onAddBefore={() => { }}
                onAddAfter={() => { }}
                onDelete={() => { }}
                onEdit={() => { }}
                onMove={() => { }}
            />
            <Typography component="h2" variant="h4" textAlign="center">{title}</Typography>
            <Tooltip placement="bottom" title="Relevant clicks. Click a card to modify, or drag in a new link to add">
                <Box
                    sx={{
                        ...containerShadow,
                        borderRadius: '16px',
                        background: (t) => t.palette.background.default,
                        border: (t) => `1px ${t.palette.text.primary}`,
                        borderStyle: canEdit ? 'dashed' : 'solid',
                        minHeight: 'min(300px, 25vh)'
                    }}
                >
                    <ul
                        style={{
                            display: 'flex',
                            padding: '0',
                            overflowX: 'scroll',
                            margin: '0',
                        }}
                    >
                        {cardData.map((c: any, index) => (

                            <li
                                style={{
                                    display: 'inline',
                                    margin: '5px',
                                }}
                            >
                                <ResourceCard 
                                    key={`resource-card-${index}`}
                                    data={c} 
                                    onClick={() => {}}
                                    onRightClick={openContext}
                                    aria-owns={Boolean(selected) ? contextId : undefined} 
                                />
                            </li>
                        ))}
                    </ul>
                </Box>
            </Tooltip>
        </Box>
    )
}