import { BreadcrumbsBase } from './BreadcrumbsBase';
import { BUSINESS_NAME, LANDING_LINKS } from '@local/shared';
import { BreadcrumbsBaseProps } from './types';

export const CopyrightBreadcrumbs = ({ 
    ...props 
}: Omit<BreadcrumbsBaseProps, 'paths' | 'ariaLabel' | 'style'>) => {
    const paths = [
        [`© ${new Date().getFullYear()} ${BUSINESS_NAME}`, LANDING_LINKS.Home],
        ['Privacy', LANDING_LINKS.PrivacyPolicy],
        ['Terms', LANDING_LINKS.Terms]
    ].map(row => ({ text: row[0], link: row[1] }))
    return BreadcrumbsBase({
        paths: paths,
        ariaLabel: 'Copyright breadcrumb',
        style: {
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
        },
        ...props
    })
}