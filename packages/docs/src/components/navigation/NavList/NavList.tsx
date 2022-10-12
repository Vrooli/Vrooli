import {
    ContactInfo,
    PopupMenu
} from 'components';
import { Container, useTheme } from '@mui/material';
import { useCallback, useEffect, useState } from 'react';

export const NavList = () => {
    const { breakpoints, palette } = useTheme();

    const [isMobile, setIsMobile] = useState(false); // Not shown on mobile
    const updateWindowDimensions = useCallback(() => setIsMobile(window.innerWidth <= breakpoints.values.md), [breakpoints]);
    useEffect(() => {
        updateWindowDimensions();
        window.addEventListener("resize", updateWindowDimensions);
        return () => window.removeEventListener("resize", updateWindowDimensions);
    }, [updateWindowDimensions]);

    return (
        <Container sx={{
            display: 'flex',
            marginTop: '0px',
            marginBottom: '0px',
            right: '0px',
            padding: '0px',
        }}>
            {!isMobile && <PopupMenu
                text="Contact"
                variant="text"
                size="large"
                sx={{
                    background: 'transparent',
                    color: palette.primary.contrastText,
                    textTransform: 'none',
                    fontSize: '1.5em',
                    '&:hover': {
                        color: palette.secondary.light,
                    },
                }}
            >
                <ContactInfo />
            </PopupMenu>}
        </Container>
    );
}