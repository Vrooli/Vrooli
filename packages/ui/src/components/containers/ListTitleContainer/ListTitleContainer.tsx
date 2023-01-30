/**
 * Title container with List inside
 */
// Used to display popular/search results of a particular object type
import { List, Typography } from '@mui/material';
import { ListTitleContainerProps } from '../types';
import { TitleContainer } from 'components';
import { useTranslation } from 'react-i18next';
import { getUserLanguages } from 'utils';

export function ListTitleContainer({
    children,
    emptyText,
    isEmpty,
    session,
    ...props
}: ListTitleContainerProps) {
    const { t } = useTranslation();
    const lng = getUserLanguages(session)[0];

    return (
        <TitleContainer {...props}>
            {
                isEmpty ?
                    <Typography variant="h6" sx={{
                        textAlign: 'center',
                        paddingTop: '8px',
                    }}>{emptyText ?? t(`common:NoResult`, { lng, count: 2 })}</Typography> :
                    <List sx={{ overflow: 'hidden' }}>
                        {children}
                    </List>
            }
        </TitleContainer>
    );
}