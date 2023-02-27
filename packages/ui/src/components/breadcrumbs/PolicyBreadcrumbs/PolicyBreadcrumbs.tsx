import { LANDING_LINKS, LANDING_URL } from '@shared/consts';
import { useTranslation } from 'react-i18next';
import { BreadcrumbsBase } from '../BreadcrumbsBase/BreadcrumbsBase';
import { PolicyBreadcrumbsProps } from '../types';

export const PolicyBreadcrumbs = ({ 
    ...props
}: PolicyBreadcrumbsProps) => {
    const { t } = useTranslation();
    return BreadcrumbsBase({
        paths: [
            [t(`Privacy`), `${LANDING_URL}${LANDING_LINKS.PrivacyPolicy}`],
            [t(`Terms`), `${LANDING_URL}${LANDING_LINKS.Terms}`]
        ].map(row => ({ text: row[0], link: row[1] })),
        ariaLabel: 'Policies breadcrumb',
        ...props
    })
}