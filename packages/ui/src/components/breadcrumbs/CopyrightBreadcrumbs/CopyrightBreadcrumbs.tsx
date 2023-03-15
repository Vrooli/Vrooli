import { BreadcrumbsBase } from '../BreadcrumbsBase/BreadcrumbsBase';
import { BUSINESS_NAME, LINKS } from '@shared/consts';
import { CopyrightBreadcrumbsProps } from '../types';
import { useTranslation } from 'react-i18next';

export const CopyrightBreadcrumbs = ({ 
    sx,
    ...props 
}: CopyrightBreadcrumbsProps) => {
    const { t } = useTranslation();
    return BreadcrumbsBase({
        paths: [
            [`© ${new Date().getFullYear()} ${BUSINESS_NAME}`, LINKS.Home],
            [t(`Privacy`), LINKS.PrivacyPolicy],
            [t(`Terms`), LINKS.Terms]
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