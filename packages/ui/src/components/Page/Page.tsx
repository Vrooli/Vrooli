import { LINKS } from '@shared/consts';
import { Redirect, useLocation } from '@shared/route';
import { PageContainer } from 'components/containers/PageContainer/PageContainer';
import { PubSub } from 'utils/pubsub';
import { PageProps } from '../../views/wrapper/types';

export const Page = ({
    children,
    mustBeLoggedIn = false,
    redirect = LINKS.Start,
    session,
    sessionChecked,
    sx,
}: PageProps) => {
    const [location] = useLocation();

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