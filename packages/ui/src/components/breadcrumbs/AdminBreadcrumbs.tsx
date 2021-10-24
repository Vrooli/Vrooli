import { LINKS } from 'utils';
import { BreadcrumbsBase } from './BreadcrumbsBase';
import { BreadcrumbsBaseProps } from './types';

const paths = [
    ['Customers', LINKS.AdminCustomers],
    ['Images', LINKS.AdminImages],
    ['Contact Info', LINKS.AdminContactInfo]
].map(row => ({ text: row[0], link: row[1] }))

export const AdminBreadcrumbs = (props: Omit<BreadcrumbsBaseProps, 'paths' | 'ariaLabel'>) => BreadcrumbsBase({
    paths: paths,
    ariaLabel: 'Admin breadcrumb',
    ...props
})