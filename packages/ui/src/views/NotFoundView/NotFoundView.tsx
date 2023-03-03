import { Link } from '@shared/route';
import { Box, Button } from '@mui/material';
import { APP_LINKS } from '@shared/consts';
import { NotFoundViewProps } from 'views/types';
import { useTranslation } from 'react-i18next';
import { TopBar } from 'components';

export const NotFoundView = ({
    session
}: NotFoundViewProps) => {
    const { t } = useTranslation();

    return (
        <>
            <TopBar
                display="page"
                onClose={() => {}}
                session={session}
                titleData={{
                    title: t('PageNotFound', { ns: 'error', defaultValue: 'Page Not Found' }),
                }}
            />
            < Box
                sx={{
                    position: 'absolute',
                    top: '50%',
                    left: '50%',
                    transform: 'translateX(-50%) translateY(-50%)',
                }
                }
            >
                <h1>{t('PageNotFound', { ns: 'error', defaultValue: 'Page Not Found' })}</h1>
                <h3>{t('PageNotFoundDetails', { ns: 'error', defaultValue: 'PageNotFoundDetails' })}</h3>
                <br />
                <Link to={APP_LINKS.Home}>
                    <Button>{t('GoToHome')}</Button>
                </Link>
            </Box >
        </>
    );
}