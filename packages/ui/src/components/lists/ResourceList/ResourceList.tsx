import { Box, Tooltip, Typography } from '@mui/material';
import { ResourceCard } from 'components';
import { ResourceListProps } from '../types';
import { centeredText, containerShadow } from 'styles';

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
    title = 'Resources'
}: ResourceListProps) => {
    return (
        <Box>
            <Typography component="h2" variant="h4" sx={{ ...centeredText }}>{title}</Typography>
            <Tooltip placement="bottom" title="Relevant clicks. Click a card to modify, or drag in a new link to add">
                <Box
                    sx={{
                        ...containerShadow,
                        borderRadius: '16px',
                        background: (t) => t.palette.background.default,
                        border: (t) => `1px dashed ${t.palette.text.primary}`,
                        minHeight: 'min(300px, 25vh)'
                    }}
                >
                    {cardData.map((c: any) => (
                        <ul
                            style={{
                                display: 'flex',
                                padding: '0',
                                overflowX: 'scroll',
                            }}
                        >
                            <li
                                style={{
                                    display: 'inline',
                                    margin: '5px',
                                }}
                            >
                                <ResourceCard data={c} />
                            </li>
                        </ul>
                    ))}
                </Box>
            </Tooltip>
        </Box>
    )
}