import { makeStyles } from '@material-ui/styles';
import { Theme, Typography } from '@material-ui/core';
import { ResourceCard } from 'components';

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
    title: {
        textAlign: 'center',
    },
    hs: {
        border: `2px dashed ${theme.palette.text.primary}`,
        borderRadius: '10px',
        display: 'flex',
        padding: '0',
        overflowX: 'scroll',
        // height: 'max(50px, 20vh)',
        // display: 'grid',
        // gridGap: 'calc(var(--gutter) / 2)',
        // gridTemplateColumns: '10px',
        // gridTemplateRows: 'minmax(150px, 1fr)',
        // gridAutoFlow: 'column',
        // gridAutoColumns: 'calc(50% - var(--gutter) * 2)',
        // overflowX: 'scroll',
        // scrollSnapType: 'x proximity',
        // paddingBottom: 'calc(.75 * var(--gutter))',
        // marginBottom: 'calc(-.25 * var(--gutter))',
        // '&::before': {
        //     content: '',
        //     width: '10px',
        // },
        // '&::after': {
        //     content: '',
        //     width: '10px',
        // },
    },
    item: {
        // scrollSnapAlign: 'center',
        // padding: 'calc(var(--gutter) / 2 * 1.5)',
        // display: 'flex',
        // flexDirection: 'column',
        // justifyContent: 'center',
        // alignItems: 'center',
        // background: '#fff',
        // borderRadius: '8px',
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
}) => {
    const classes = useStyles();

    return (
        <div>
            <Typography variant="h3" className={classes.title}>{title}</Typography>
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