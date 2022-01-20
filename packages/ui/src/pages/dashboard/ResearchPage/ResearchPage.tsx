import { Box, Typography } from '@mui/material';
import { ResourceList } from 'components';

export const ResearchPage = ({

}) => {
    return (
        <Box id="page">
            <Typography component="h1" variant="h2" textAlign="center">Research Dashboard</Typography>
            <ResourceList />
        </Box>
    )
}