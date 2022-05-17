/**
 * Title container with List inside
 */
// Used to display popular/search results of a particular object type
import { List } from '@mui/material';
import { TitleContainerProps } from '../types';
import { TitleContainer } from 'components';

export function ListTitleContainer({
    children,
    ...props
}: TitleContainerProps) {

    return (
        <TitleContainer {...props}>
            <List>
                {children}
            </List>
        </TitleContainer>
    );
}