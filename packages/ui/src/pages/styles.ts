import { Theme } from '@mui/material';
import { Styles } from "@mui/styles";

export * from './dashboard/styles';
export * from './search/styles';
export * from './view/styles';
export * from './wrapper/styles';

export const pageStyles: Styles<Theme, {}> = (theme: Theme) => ({
    header: {
        textAlign: 'center',
    },
});