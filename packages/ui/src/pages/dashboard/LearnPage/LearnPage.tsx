import { Box, Typography } from '@mui/material';
import { ResourceList } from 'components';
import { centeredText } from 'styles';

export const LearnPage = ({

}) => {
    return (
        <Box id="page">
            <Typography component="h1" variant="h3" sx={{ ...centeredText }}>Learn Dashboard</Typography>
            <ResourceList />
        </Box>
    )
}