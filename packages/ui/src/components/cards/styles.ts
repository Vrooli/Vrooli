import { SxProps } from '@mui/material';
import { containerShadow } from 'styles';

export const cardRoot: SxProps = {
    ...containerShadow,
    background: '#2167a3',
    color: (t: any) => t.palette.primary.contrastText,
    borderRadius: '16px',
    margin: 0,
    cursor: 'pointer',
    maxWidth: '500px',
    '&:hover': {
        filter: `brightness(120%)`,
        transition: 'filter 0.2s',
    },
}

export const cardContent: SxProps = {
    padding: 1,
    position: 'inherit',
}