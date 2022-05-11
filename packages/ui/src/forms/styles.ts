import { SxProps, Theme } from '@mui/material';
import { CSSProperties } from '@mui/styles';

export const formPaper: SxProps = {
    marginTop: 3,
    background: 'transparent',
    boxShadow: 'none',
} as CSSProperties;

export const formSubmit: SxProps = {
    margin: '16px auto',
} as CSSProperties;

export const formNavLink = ({ palette }: Theme): SxProps => ({
    color: palette.secondary.dark,
    display: 'flex',
    alignItems: 'center',
}) as CSSProperties;
