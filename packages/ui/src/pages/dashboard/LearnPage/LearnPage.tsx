import { Box, Typography } from '@mui/material';
import { ResourceListHorizontal } from 'components';

export const LearnPage = ({

}) => {
    return (
        <Box id="page">
            <Typography component="h1" variant="h3" textAlign="center">Learn Dashboard</Typography>
            <ResourceListHorizontal />
        </Box>
    )
}