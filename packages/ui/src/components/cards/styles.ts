import { SxProps } from '@mui/material';

export const cardRoot: SxProps = {
    boxShadow: 6,
    background: (t: any) => t.palette.primary.light,
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