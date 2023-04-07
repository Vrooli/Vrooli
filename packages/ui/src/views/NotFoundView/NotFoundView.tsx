import { Box, Button } from '@mui/material';
import { LINKS } from '@shared/consts';
import { Link } from '@shared/route';
import { TopBar } from 'components/navigation/TopBar/TopBar';
import { useTranslation } from 'react-i18next';

export const NotFoundView = () => {
    const { t } = useTranslation();

    return (
        <>
            <TopBar
                display="page"
                onClose={() => { }}
                titleData={{
                    title: t('PageNotFound', { ns: 'error', defaultValue: 'Page Not Found' }),
                }}
            />
            <Box
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
                <Link to={LINKS.Home}>
                    <Button>{t('GoToHome')}</Button>
                </Link>
            </Box>
        </>
    );
}