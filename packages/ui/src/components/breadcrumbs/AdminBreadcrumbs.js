import React from 'react';
import { LINKS } from 'utils';
import { BreadcrumbsBase } from './BreadcrumbsBase';

const paths = [
    ['Customers', LINKS.AdminCustomers],
    ['Hero', LINKS.AdminHero],
    ['Contact Info', LINKS.AdminContactInfo]
]

const AdminBreadcrumbs = ({...props}) => BreadcrumbsBase({
    paths: paths,
    ariaLabel: 'Admin breadcrumb',
    ...props
})

export { AdminBreadcrumbs };