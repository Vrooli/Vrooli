import { Box, Container, Typography, useTheme } from '@mui/material';

interface Props {
    title: string;
    autocomplete?: string;
    children: JSX.Element;
    maxWidth?: string | number;
}

export const FormPage = ({
    title,
    autocomplete = 'on',
    children,
    maxWidth = '90%',
}: Props) => {
    const { palette } = useTheme();

    return (
        <Box id='page' sx={{
            maxWidth: maxWidth,
            padding: { sx: '0', md: '0.5em' },
            paddingTop: { xs: '64px', md: '80px' },
        }}>
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
                <Container sx={{
                    backgroundColor: palette.primary.main,
                    color: palette.primary.contrastText,
                    padding: '1em',
                    textAlign: 'center',
                }}>
                    <Typography variant="h3" >{title}</Typography>
                </Container>
                <Container>
                    {children}
                </Container>
            </Box>
        </Box >
    );
}