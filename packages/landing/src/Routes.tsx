import { Suspense, useCallback } from 'react';
import { lazily } from 'react-lazily';
import { Switch, Route } from 'react-router-dom';
import { LANDING_LINKS as LINKS } from '@local/shared';
import { Sitemap } from 'Sitemap';
import { ScrollToTop } from 'components';
import { BUSINESS_NAME } from '@local/shared';

// Lazy loading in the Routes component is a recommended way to improve performance. See https://reactjs.org/docs/code-splitting.html#route-based-code-splitting
const {
    AboutPage,
    HomePage,
    MissionPage,
    NotFoundPage,
    Page,
    PrivacyPolicyPage,
    TermsPage,
} = lazily(() => import('./pages'));

const Routes = () => {

    const title = useCallback((page: string) => `${page} | ${BUSINESS_NAME}`, []);

    return (
        <Suspense fallback={<div>Loading...</div>}>
            <ScrollToTop />
            <Switch>
                <Route
                    path="/sitemap"
                    component={Sitemap}
                />

                {/* ========= START INFORMATIONAL ROUTES ========= */}
                {/* Informational pages to describe Vrooli to potential customers */}
                <Route
                    exact
                    path={LINKS.Home}
                    sitemapIndex={true}
                    priority={1.0}
                    changefreq="monthly"
                    render={() => (
                        <Page title={title('Home')}>
                            <HomePage />
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={LINKS.Mission}
                    sitemapIndex={true}
                    priority={1.0}
                    changefreq="monthly"
                    render={() => (
                        <Page title={title('Mission')}>
                            <MissionPage />
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={LINKS.About}
                    sitemapIndex={true}
                    priority={0.7}
                    render={() => (
                        <Page title={title('About')}>
                            <AboutPage />
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={LINKS.PrivacyPolicy}
                    sitemapIndex={true}
                    priority={0.1}
                    render={() => (
                        <Page title={title('Privacy Policy')}>
                            <PrivacyPolicyPage />
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={LINKS.Terms}
                    sitemapIndex={true}
                    priority={0.1}
                    render={() => (
                        <Page title={title('Terms & Conditions')}>
                            <TermsPage />
                        </Page>
                    )}
                />
                {/* ========= END INFORMATIONAL ROUTES ========= */}

                <Route
                    render={() => (
                        <Page title={title('404')}>
                            <NotFoundPage />
                        </Page>
                    )}
                />
            </Switch>
        </Suspense>
    );
}

export { Routes };