import { makeStyles } from '@mui/styles';
import { Theme, Typography } from '@mui/material';
import { ResourceCard } from 'components';
import { ResourceListProps } from '../types';

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

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        border: `2px dashed ${theme.palette.text.primary}`,
        borderRadius: '10px',
        padding: '0',
    },
    title: {
        textAlign: 'center',
        background: theme.palette.primary.light,
        color: theme.palette.primary.contrastText,
    },
    hs: {
        display: 'flex',
        padding: '0',
        overflowX: 'scroll',
    },
    item: {
        display: 'inline',
        margin: '5px',
    },
    noScrollbar: {
        scrollbarWidth: 'none',
        marginBottom: '0',
        paddingBottom: '0',
        '&::-webkit-scrollbar': {
            display: 'none',
        }
    }
}));

export const ResourceList = ({
    title = 'Resources'
}: ResourceListProps) => {
    const classes = useStyles();

    return (
        <div className={classes.root}>
            <Typography component="h2" variant="h4" className={classes.title}>{title}</Typography>
            <ul className={`${classes.hs}`}>
                {cardData.map((c: any) => (
                    <li className={classes.item}>
                        <ResourceCard resource={c} />
                    </li>
                ))}
            </ul>
        </div>
    )
}