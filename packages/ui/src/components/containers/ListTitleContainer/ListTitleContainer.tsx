/**
 * Title container with List inside
 */
// Used to display popular/search results of a particular object type
import { List, Typography } from '@mui/material';
import { ListTitleContainerProps } from '../types';
import { TitleContainer } from 'components';
import { useTranslation } from 'react-i18next';

export function ListTitleContainer({
    children,
    emptyText,
    isEmpty,
    ...props
}: ListTitleContainerProps) {
    const { t } = useTranslation();

    return (
        <TitleContainer {...props}>
            {
                isEmpty ?
                    <Typography variant="h6" sx={{
                        textAlign: 'center',
                        paddingTop: '8px',
                    }}>{emptyText ?? t('NoResults', { ns: 'error' })}</Typography> :
                    <List sx={{ overflow: 'hidden' }}>
                        {children}
                    </List>
            }
        </TitleContainer>
    );
}