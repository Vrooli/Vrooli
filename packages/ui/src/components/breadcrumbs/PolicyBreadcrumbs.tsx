import { LANDING_LINKS, LANDING_URL } from '@local/shared';
import { BreadcrumbsBase } from './BreadcrumbsBase';
import { BreadcrumbsBaseProps } from './types';

const paths = [
    ['Privacy', `${LANDING_URL}${LANDING_LINKS.PrivacyPolicy}`],
    ['Terms', `${LANDING_URL}${LANDING_LINKS.Terms}`]
].map(row => ({ text: row[0], link: row[1] }))

export const PolicyBreadcrumbs = ({...props}: Omit<BreadcrumbsBaseProps, 'paths' | 'ariaLabel'>) => BreadcrumbsBase({
    paths: paths,
    ariaLabel: 'Policies breadcrumb',
    ...props
})