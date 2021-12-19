import { useEffect } from 'react';
import { APP_LINKS } from '@local/shared';
import { useLocation, Redirect } from 'react-router-dom';
import { UserRoles } from 'types';
import { PUBS } from 'utils';

interface Props {
    title?: string;
    sessionChecked: boolean;
    redirect?: string;
    userRoles: UserRoles;
    restrictedToRoles?: string[];
    children: JSX.Element;
}

export const Page = ({
    title = '',
    sessionChecked,
    redirect = APP_LINKS.Home,
    userRoles,
    restrictedToRoles = [],
    children
}: Props) => {
    const location = useLocation();
    console.log('sessio checked', sessionChecked, userRoles);

    useEffect(() => {
        document.title = title;
    }, [title]);

    // If this page has restricted access
    if (restrictedToRoles.length > 0) {
        if (Array.isArray(userRoles)) {
            if (userRoles.some(r => restrictedToRoles.includes(r.role.title))) return children;
        }
        if (sessionChecked && location.pathname !== redirect) { 
            PubSub.publish(PUBS.Snack, { message: 'Page restricted. Please sign in', severity: 'error' });
            return <Redirect to={redirect} />
        }
        return null;
    }

    return children;
};