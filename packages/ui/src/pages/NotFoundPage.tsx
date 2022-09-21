import { Link } from '@shared/route';
import { Box, Button } from '@mui/material';
import { APP_LINKS } from '@shared/consts';
import { PageContainer } from 'components';

export const NotFoundPage = () => {
    return (
        <PageContainer>
            <Box
                sx={{
                    position: 'absolute',
                    top: '50%',
                    left: '50%',
                    transform: 'translateX(-50%) translateY(-50%)',
                }}
            >
                <h1>Page Not Found</h1>
                <h3>Looks like you've followed a broken link or entered a URL that doesn't exist on this site</h3>
                <br />
                <Link to={APP_LINKS.Home}>
                    <Button>Go to Home</Button>
                </Link>
            </Box>
        </PageContainer>
    );
}