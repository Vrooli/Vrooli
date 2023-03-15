import { Palette } from "@mui/material";

export const searchButtonStyle = (palette: Palette) => ({
    minHeight: '34px',
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    borderRadius: '50px',
    border: `2px solid ${palette.secondary.main}`,
    margin: 1,
    padding: 0,
    paddingLeft: 1,
    paddingRight: 1,
    cursor: 'pointer',
    '&:hover': {
        transform: 'scale(1.1)',
    },
    transition: 'transform 0.2s ease-in-out',
});