import { Suspense, useCallback } from 'react';
import { lazily } from 'react-lazily';
import { Route, Switch } from '@local/route';
import { BUSINESS_NAME } from '@local/shared';
// import { Sitemap } from 'Sitemap';
import { ScrollToTop } from 'components';
import { Box, CircularProgress } from '@mui/material';
import { Page } from 'pages';

// Lazy loading in the Routes component is a recommended way to improve performance. See https://reactjs.org/docs/code-splitting.html#route-based-code-splitting
const { NotFoundPage } = lazily(() => import('./pages/NotFoundPage'));

const Fallback = <Box sx={{
    position: 'absolute',
    top: '50%',
    left: '50%',
    transform: 'translate(-50%, -50%)',
    zIndex: 100000,
}}>
    <CircularProgress size={100} />
</Box>

export const AllRoutes = () => {

    const title = useCallback((page: string) => `${page} | ${BUSINESS_NAME}`, []);

    return (
        <>
            <ScrollToTop />
            {/* <Route
                    path="/sitemap"
                    element={Sitemap}
                /> */}
            {/* <Switch> */}
                <Route>
                    <Suspense fallback={Fallback}>
                        <Page title={title('404')}>
                            <NotFoundPage />
                        </Page>
                    </Suspense>
                </Route>
            {/* </Switch> */}
        </>
    );
}