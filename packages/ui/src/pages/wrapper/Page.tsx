import { useEffect } from 'react';
import { APP_LINKS } from '@local/shared';
import { useLocation, Redirect } from 'wouter';
import { Pubs } from 'utils';
import { PageProps } from './types';

export const Page = ({
    children,
    redirect = APP_LINKS.Home,
    restrictedToRoles = [],
    session,
    sessionChecked,
    title = '',
}: PageProps) => {
    const [location] = useLocation();

    useEffect(() => {
        document.title = title;
    }, [title]);

    // If this page has restricted access
    if (restrictedToRoles.length > 0) {
        if (Array.isArray(session.roles)) {
            if (session.roles.some(r => restrictedToRoles.includes(r))) return children;
        }
        if (sessionChecked && location !== redirect) { 
            PubSub.publish(Pubs.Snack, { message: 'Page restricted. Please log in', severity: 'error' });
            return <Redirect to={redirect} />
        }
        return null;
    }

    return children;
};