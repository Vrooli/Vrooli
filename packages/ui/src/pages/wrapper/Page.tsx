import { useEffect } from 'react';
import { APP_LINKS } from '@local/shared';
import { useLocation, Redirect } from 'wouter';
import { PageProps } from './types';
import { PubSub } from 'utils';

export const Page = ({
    children,
    mustBeLoggedIn = false,
    redirect = APP_LINKS.Start,
    session,
    sessionChecked,
    title = '',
}: PageProps) => {
    const [location] = useLocation();

    useEffect(() => {
        document.title = title;
    }, [title]);

    // If this page has restricted access
    if (mustBeLoggedIn) {
        if (session.isLoggedIn) return children;
        if (sessionChecked && location !== redirect) { 
            PubSub.get().publishSnack({ message: 'Page restricted. Please log in', severity: 'error' });
            return <Redirect to={redirect} />
        }
        return null;
    }

    return children;
};