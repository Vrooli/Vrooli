import { Box, Typography } from '@mui/material';
import { ResourceList } from 'components';

export const DevelopPage = ({

}) => {
    return (
        <Box id="page">
            <Typography component="h1" variant="h2" textAlign="center">Develop Dashboard</Typography>
            <ResourceList />
        </Box>
    )
}