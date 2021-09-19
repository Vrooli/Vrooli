import React from 'react';
import { LINKS } from 'utils';
import { BreadcrumbsBase } from './BreadcrumbsBase';

const paths = [
    ['Orders', LINKS.AdminOrders],
    ['Customers', LINKS.AdminCustomers],
    ['Inventory', LINKS.AdminInventory],
    ['Hero', LINKS.AdminHero],
    ['Gallery', LINKS.AdminGallery],
    ['Contact Info', LINKS.AdminContactInfo]
]

const AdminBreadcrumbs = ({...props}) => BreadcrumbsBase({
    paths: paths,
    ariaLabel: 'Admin breadcrumb',
    ...props
})

export { AdminBreadcrumbs };