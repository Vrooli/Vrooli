import { LANDING_LINKS } from '@local/shared';
import { BreadcrumbsBase } from './BreadcrumbsBase';
import { BreadcrumbsBaseProps } from './types';

const paths = [
    ['Privacy', LANDING_LINKS.PrivacyPolicy],
    ['Terms', LANDING_LINKS.Terms]
].map(row => ({ text: row[0], link: row[1] }))

export const PolicyBreadcrumbs = ({...props}: Omit<BreadcrumbsBaseProps, 'paths' | 'ariaLabel'>) => BreadcrumbsBase({
    paths: paths,
    ariaLabel: 'Policies breadcrumb',
    ...props
})