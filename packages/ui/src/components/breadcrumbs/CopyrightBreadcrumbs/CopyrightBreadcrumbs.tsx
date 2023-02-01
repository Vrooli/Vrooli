import { BreadcrumbsBase } from '../BreadcrumbsBase/BreadcrumbsBase';
import { BUSINESS_NAME, LANDING_LINKS, LANDING_URL } from '@shared/consts';
import { CopyrightBreadcrumbsProps } from '../types';
import { getUserLanguages } from 'utils';
import { useTranslation } from 'react-i18next';

export const CopyrightBreadcrumbs = ({ 
    session,
    sx,
    ...props 
}: CopyrightBreadcrumbsProps) => {
    const { t } = useTranslation();
    const lng = getUserLanguages(session)[0];
    return BreadcrumbsBase({
        paths: [
            [`© ${new Date().getFullYear()} ${BUSINESS_NAME}`, `${LANDING_URL}${LANDING_LINKS.Home}`],
            [t(`common:Privacy`, { lng }), `${LANDING_URL}${LANDING_LINKS.PrivacyPolicy}`],
            [t(`common:Terms`, { lng }), `${LANDING_URL}${LANDING_LINKS.Terms}`]
        ].map(row => ({ text: row[0], link: row[1] })),
        ariaLabel: 'Copyright breadcrumb',
        sx: {
            ...sx,
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
        },
        ...props
    })
}