import { Box, Typography } from '@mui/material';
import { ResourceList } from 'components';

export const LearnPage = ({

}) => {
    return (
        <Box id="page">
            <Typography component="h1" variant="h3" textAlign="center">Learn Dashboard</Typography>
            <ResourceList />
        </Box>
    )
}