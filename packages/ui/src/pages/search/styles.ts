import { Theme } from '@mui/material';
import { Styles } from '@mui/styles';

export const searchStyles: Styles<Theme, {}> = (theme: Theme) => ({
    cardFlex: {
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))',
        alignItems: 'stretch',
    },
    selector: {
        marginBottom: '1em',
    },
});