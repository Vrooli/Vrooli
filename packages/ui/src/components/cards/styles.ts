import { Theme } from '@mui/material';
import { Styles } from '@mui/styles';

export const cardStyles: Styles<Theme, {}> = (theme: Theme) => ({
    cardRoot: {
        background: theme.palette.primary.main,
        color: theme.palette.primary.contrastText,
        borderRadius: 15,
        margin: 3,
        cursor: 'pointer',
    },
    content: {
        padding: 8,
        position: 'inherit',
    },
    icon: {
        fill: theme.palette.secondary.light,
    },
});