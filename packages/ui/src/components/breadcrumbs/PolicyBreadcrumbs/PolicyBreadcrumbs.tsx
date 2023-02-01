import { LANDING_LINKS, LANDING_URL } from '@shared/consts';
import { useTranslation } from 'react-i18next';
import { getUserLanguages } from 'utils';
import { BreadcrumbsBase } from '../BreadcrumbsBase/BreadcrumbsBase';
import { PolicyBreadcrumbsProps } from '../types';

export const PolicyBreadcrumbs = ({ 
    session,
    ...props
}: PolicyBreadcrumbsProps) => {
    const { t } = useTranslation();
    const lng = getUserLanguages(session)[0];
    return BreadcrumbsBase({
        paths: [
            [t(`common:Privacy`, { lng }), `${LANDING_URL}${LANDING_LINKS.PrivacyPolicy}`],
            [t(`common:Terms`, { lng }), `${LANDING_URL}${LANDING_LINKS.Terms}`]
        ].map(row => ({ text: row[0], link: row[1] })),
        ariaLabel: 'Policies breadcrumb',
        ...props
    })
}