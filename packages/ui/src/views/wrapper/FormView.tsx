import { Box, Container, useTheme } from '@mui/material';
import { Session } from '@shared/consts';
import { TopBar } from 'components';
import { ViewDisplayType } from 'views/types';

interface Props {
    display?: ViewDisplayType;
    title: string;
    autocomplete?: string;
    children: JSX.Element;
    maxWidth?: string | number;
    session: Session | undefined;
}

export const FormView = ({
    display = 'page',
    title,
    autocomplete = 'on',
    children,
    maxWidth = '90%',
    session,
}: Props) => {
    const { palette } = useTheme();

    return (
        <>
            <TopBar
                display={display}
                onClose={() => {}}
                session={session}
                titleData={{
                    title,
                }}
            />
            <Box sx={{
                backgroundColor: palette.background.paper,
                display: 'grid',
                position: 'relative',
                boxShadow: '0px 2px 4px -1px rgb(0 0 0 / 20%), 0px 4px 5px 0px rgb(0 0 0 / 14%), 0px 1px 10px 0px rgb(0 0 0 / 12%)',
                minWidth: '300px',
                maxWidth: 'min(100%, 700px)',
                borderRadius: '10px',
                overflow: 'hidden',
                left: '50%',
                transform: 'translateX(-50%)',
                marginBottom: '20px'
            }}>
                <Container>
                    {children}
                </Container>
            </Box>
        </>
    );
}