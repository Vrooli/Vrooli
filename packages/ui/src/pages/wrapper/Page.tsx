import { useEffect } from 'react';
import { APP_LINKS } from '@local/shared';
import { useLocation, Redirect } from 'react-router-dom';
import { UserRoles } from 'types';

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

    useEffect(() => {
        document.title = title;
    }, [title]);

    // If this page has restricted access
    if (restrictedToRoles.length > 0) {
        if (Array.isArray(userRoles)) {
            if (userRoles.some(r => restrictedToRoles.includes(r))) return children;
        }
        if (sessionChecked && location.pathname !== redirect) return <Redirect to={redirect} />
        return null;
    }

    return children;
};