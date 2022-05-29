/**
 * Title container with List inside
 */
// Used to display popular/search results of a particular object type
import { List, Typography } from '@mui/material';
import { ListTitleContainerProps } from '../types';
import { TitleContainer } from 'components';

export function ListTitleContainer({
    children,
    emptyText = 'No results',
    isEmpty,
    ...props
}: ListTitleContainerProps) {

    return (
        <TitleContainer {...props}>
            {
                isEmpty ?
                    <Typography variant="h6" sx={{
                        textAlign: 'center',
                        paddingTop: '8px',
                    }}>{emptyText}</Typography> :
                    <List sx={{ overflow: 'hidden' }}>
                        {children}
                    </List>
            }
        </TitleContainer>
    );
}