import { Box, Typography } from '@mui/material';
import { ResourceList } from 'components';
import { centeredText } from 'styles';

export const ResearchPage = ({

}) => {
    return (
        <Box id="page">
            <Typography component="h1" variant="h2" sx={{ ...centeredText }}>Research Dashboard</Typography>
            <ResourceList />
        </Box>
    )
}