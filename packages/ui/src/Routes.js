import React, { Suspense } from 'react';
import { lazily } from 'react-lazily'
import PropTypes from 'prop-types';
import { Switch, Route } from 'react-router-dom';
import { ROLES } from '@local/shared';
import { LINKS } from 'utils';
import { Sitemap } from 'Sitemap';
import {
    ForgotPasswordForm,
    LogInForm,
    ProfileForm,
    ResetPasswordForm,
    SignUpForm
} from 'forms';
import { ScrollToTop } from 'components';

// Lazy loading in the Routes component is a recommended way to improve performance. See https://reactjs.org/docs/code-splitting.html#route-based-code-splitting
// const AboutPage = lazy(() => import('./pages'), 'AboutPage');
// const AdminContactPage = lazy(() => import ('./pages/admin/AdminContactPage/AdminContactPage'), 'AdminContactPage');
const {
    AboutPage,
    AdminContactPage,
    AdminCustomerPage,
    AdminGalleryPage,
    AdminHeroPage,
    AdminMainPage,
    AdminInventoryPage,
    AdminOrderPage,
    CartPage,
    FormPage,
    GalleryPage,
    HomePage,
    NotFoundPage,
    Page,
    PrivacyPolicyPage,
    ShoppingPage,
    TermsPage,
} = lazily(() => import('./pages'));

function Routes({
    session,
    onSessionUpdate,
    business,
    userRoles,
    cart,
    onRedirect
}) {

    const common = {
        sessionChecked: session !== null && session !== undefined,
        onSessionUpdate: onSessionUpdate,
        onRedirect: onRedirect,
        userRoles: userRoles,
        business: business
    }

    const title = (page) => `${page} | ${business?.BUSINESS_NAME?.Short}`;

    return (
        <Suspense fallback={<div>Loading...</div>}>
            <ScrollToTop />
            <Switch>
                {/* START PUBLIC PAGES */}
                <Route
                    path="/sitemap"
                    component={Sitemap}
                />
                <Route
                    exact
                    path={LINKS.Home}
                    sitemapIndex={true}
                    priority={1.0}
                    changefreq="monthly"
                    render={() => (
                        <Page title={title('Home')} {...common}>
                            <HomePage />
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={LINKS.About}
                    sitemapIndex={true}
                    priority={0.7}
                    render={() => (
                        <Page title={title('About')} {...common}>
                            <AboutPage {...common} />
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={LINKS.PrivacyPolicy}
                    sitemapIndex={true}
                    priority={0.1}
                    render={() => (
                        <Page title={title('Privacy Policy')} {...common}>
                            <PrivacyPolicyPage business={business} />
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={LINKS.Terms}
                    sitemapIndex={true}
                    priority={0.1}
                    render={() => (
                        <Page title={title('Terms & Conditions')} {...common}>
                            <TermsPage business={business} />
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={`${LINKS.Gallery}/:img?`}
                    sitemapIndex={true}
                    priority={0.3}
                    render={() => (
                        <Page title={title('Gallery')} {...common}>
                            <GalleryPage />
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={LINKS.Register}
                    sitemapIndex={true}
                    priority={0.9}
                    render={() => (
                        <Page title={title('Sign Up')} {...common}>
                            <FormPage title="Sign Up" maxWidth="700px">
                                <SignUpForm {...common} />
                            </FormPage>
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={`${LINKS.LogIn}/:code?`}
                    sitemapIndex={true}
                    priority={0.8}
                    render={() => (
                        <Page title={title('Log In')} {...common}>
                            <FormPage title="Log In" maxWidth="700px">
                                <LogInForm {...common} />
                            </FormPage>
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={`${LINKS.ForgotPassword}/:code?`}
                    sitemapIndex={true}
                    priority={0.1}
                    render={() => (
                        <Page title={title('Forgot Password')} {...common}>
                            <FormPage title="Forgot Password" maxWidth="700px">
                                <ForgotPasswordForm {...common} />
                            </FormPage>
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={`${LINKS.ResetPassword}/:id?/:code?`}
                    sitemapIndex={true}
                    priority={0.1}
                    render={() => (
                        <Page title={title('Reset Password')} {...common}>
                            <FormPage title="Reset Password" maxWidth="700px">
                                <ResetPasswordForm {...common} />
                            </FormPage>
                        </Page>
                    )}
                />
                {/* END PUBLIC PAGES */}
                {/* START CUSTOMER PAGES */}
                <Route
                    exact
                    path={LINKS.Profile}
                    sitemapIndex={true}
                    priority={0.4}
                    render={() => (
                        <Page title={title('Profile')} {...common} restrictedToRoles={Object.values(ROLES)}>
                            <FormPage title="Profile">
                                <ProfileForm {...common} />
                            </FormPage>
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={`${LINKS.Shopping}/:sku?`}
                    sitemapIndex={true}
                    priority={0.9}
                    render={() => (
                        <Page title={title('Shop')} {...common} restrictedToRoles={Object.values(ROLES)} redirect={LINKS.LogIn}>
                            <ShoppingPage {...common} session={session} cart={cart} />
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={LINKS.Cart}
                    render={() => (
                        <Page title={title('Cart')} {...common} restrictedToRoles={Object.values(ROLES)} redirect={LINKS.LogIn}>
                            <CartPage {...common} cart={cart} />
                        </Page>
                    )}
                />
                {/* END CUSTOMER PAGES */}
                {/* START ADMIN PAGES */}
                <Route
                    exact
                    path={LINKS.Admin}
                    render={() => (
                        <Page title={title('Manage Site')} {...common} restrictedToRoles={[ROLES.Owner, ROLES.Admin]}>
                            <AdminMainPage />
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={LINKS.AdminContactInfo}
                    render={() => (
                        <Page title={"Edit Contact Info"} {...common} restrictedToRoles={[ROLES.Owner, ROLES.Admin]}>
                            <AdminContactPage business={business} />
                        </Page>
                    )}
                />
                <Route exact path={LINKS.AdminCustomers} render={() => (
                    <Page title={"Customer Page"} {...common} restrictedToRoles={[ROLES.Owner, ROLES.Admin]}>
                        <AdminCustomerPage />
                    </Page>
                )} />
                <Route exact path={LINKS.AdminGallery} render={() => (
                    <Page title={"Edit Gallery"} {...common} restrictedToRoles={[ROLES.Owner, ROLES.Admin]}>
                        <AdminGalleryPage />
                    </Page>
                )} />
                <Route exact path={LINKS.AdminHero} render={() => (
                    <Page title={"Edit Hero"} {...common} restrictedToRoles={[ROLES.Owner, ROLES.Admin]}>
                        <AdminHeroPage />
                    </Page>
                )} />
                <Route exact path={LINKS.AdminInventory} render={() => (
                    <Page title={"Edit Inventory Info"} {...common} restrictedToRoles={[ROLES.Owner, ROLES.Admin]}>
                        <AdminInventoryPage />
                    </Page>
                )} />
                <Route exact path={LINKS.AdminOrders} render={() => (
                    <Page title={"Order Page"} {...common} restrictedToRoles={[ROLES.Owner, ROLES.Admin]}>
                        <AdminOrderPage userRoles={userRoles} />
                    </Page>
                )} />
                {/* END ADMIN PAGES */}
                {/* 404 page */}
                <Route
                    render={() => (
                        <Page title={title('404')} {...common}>
                            <NotFoundPage />
                        </Page>
                    )}
                />
            </Switch>
        </Suspense>
    );
}

Routes.propTypes = {
    session: PropTypes.object,
    onSessionUpdate: PropTypes.func.isRequired,
    roles: PropTypes.array,
    cart: PropTypes.object,
    onRedirect: PropTypes.func.isRequired,
}

export { Routes };