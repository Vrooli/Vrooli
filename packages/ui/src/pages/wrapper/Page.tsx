import { useEffect } from 'react';
import { APP_LINKS } from '@local/shared';
import { useLocation, Redirect } from '@local/shared';
import { PageProps } from './types';
import { PubSub } from 'utils';

export const Page = ({
    children,
    mustBeLoggedIn = false,
    redirect = APP_LINKS.Start,
    session,
    sessionChecked,
    title,
}: PageProps) => {
    const [location] = useLocation();

    // Set sessionStorage with current URL, 
    // so we can see what the previous page was
    useEffect(() => {
        const pathname = window.location.pathname;
        PubSub.get().publishSnack({ message: `Current page: ${pathname}` });
        sessionStorage.setItem('previousPage', window.location.pathname);
        return () => { sessionStorage.removeItem('previousPage'); }
    } , []);

    useEffect(() => {
        if (title) document.title = title;
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