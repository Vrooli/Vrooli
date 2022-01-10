import { SxProps, Theme } from '@mui/material';
import { CSSProperties } from '@mui/styles';

export const formPaper: SxProps = {
    width: '100%',
    marginTop: 3
} as CSSProperties;

export const formSubmit: SxProps = {
    margin: '0 auto',
} as CSSProperties;

export const formNavLink = (t: Theme): SxProps => ({
    color: t.palette.secondary.dark,
    display: 'flex',
    alignItems: 'center',
}) as CSSProperties;
