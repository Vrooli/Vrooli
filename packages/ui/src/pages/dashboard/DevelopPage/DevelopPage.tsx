import { Box, Typography } from '@mui/material';
import { ResourceList } from 'components';
import { centeredText } from 'styles';

export const DevelopPage = ({

}) => {
    return (
        <Box id="page">
            <Typography component="h1" variant="h2" sx={{ ...centeredText }}>Develop Dashboard</Typography>
            <ResourceList />
        </Box>
    )
}