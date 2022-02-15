import { LANDING_LINKS, LANDING_URL } from '@local/shared';
import { BreadcrumbsBase } from '../BreadcrumbsBase/BreadcrumbsBase';
import { PolicyBreadcrumbsProps } from '../types';

const paths = [
    ['Privacy', `${LANDING_URL}${LANDING_LINKS.PrivacyPolicy}`],
    ['Terms', `${LANDING_URL}${LANDING_LINKS.Terms}`]
].map(row => ({ text: row[0], link: row[1] }))

export const PolicyBreadcrumbs = ({...props}: PolicyBreadcrumbsProps) => BreadcrumbsBase({
    paths: paths,
    ariaLabel: 'Policies breadcrumb',
    ...props
})