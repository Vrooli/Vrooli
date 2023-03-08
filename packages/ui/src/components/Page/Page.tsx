import { useEffect } from 'react';
import { APP_LINKS } from '@shared/consts';
import { useLocation, Redirect } from '@shared/route';
import { PageProps } from '../../views/wrapper/types';
import { getTranslatedTitleAndHelp, PubSub } from 'utils';
import { PageContainer } from 'components';

export const Page = ({
    children,
    mustBeLoggedIn = false,
    redirect = APP_LINKS.Start,
    session,
    sessionChecked,
    sx,
    titleData,
}: PageProps) => {
    const [location] = useLocation();

    useEffect(() => {
        const { title } = getTranslatedTitleAndHelp(titleData);
        if (title) document.title = title;
    }, [titleData]);

    // If this page has restricted access
    if (mustBeLoggedIn) {
        if (session?.isLoggedIn) return children;
        if (sessionChecked && location !== redirect) {
            PubSub.get().publishSnack({ messageKey: 'PageRestricted', severity: 'Error' });
            return <Redirect to={redirect} />
        }
        return null;
    }

    return (
        <PageContainer sx={sx}>
            {children}
        </PageContainer>
    )
};