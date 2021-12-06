import { LANDING_LINKS, WEBSITE } from '@local/shared';
import { BreadcrumbsBase } from './BreadcrumbsBase';
import { BreadcrumbsBaseProps } from './types';

const paths = [
    ['Privacy', `${WEBSITE}${LANDING_LINKS.PrivacyPolicy}`],
    ['Terms', `${WEBSITE}${LANDING_LINKS.Terms}`]
].map(row => ({ text: row[0], link: row[1] }))

export const PolicyBreadcrumbs = ({...props}: Omit<BreadcrumbsBaseProps, 'paths' | 'ariaLabel'>) => BreadcrumbsBase({
    paths: paths,
    ariaLabel: 'Policies breadcrumb',
    ...props
})