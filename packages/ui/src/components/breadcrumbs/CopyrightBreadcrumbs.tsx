import { BreadcrumbsBase } from 'components';
import { LINKS } from 'utils';
import { BreadcrumbsBaseProps } from './types';

interface Props extends BreadcrumbsBaseProps  {
    business: { BUSINESS_NAME: { Long: string; Short: string; } };
}

export const CopyrightBreadcrumbs = ({ 
    business, 
    ...props 
}: Omit<Props, 'paths' | 'ariaLabel' | 'style'>) => {
    const paths = [
        [`© ${new Date().getFullYear()} ${business?.BUSINESS_NAME?.Long ?? business?.BUSINESS_NAME?.Short ?? 'Home'}`, LINKS.Home],
        ['Privacy', LINKS.PrivacyPolicy],
        ['Terms', LINKS.Terms]
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