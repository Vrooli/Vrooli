import React from 'react';
import { LINKS } from 'utils';
import { BreadcrumbsBase } from './BreadcrumbsBase';

const paths = [
    ['Home', LINKS.Home],
    ['Mission', LINKS.Mission],
    ['About Us', LINKS.About],
]

const InformationalBreadcrumbs = ({...props}) => BreadcrumbsBase({
    paths: paths,
    ariaLabel: 'About us breadcrumb',
    ...props
})

export { InformationalBreadcrumbs };