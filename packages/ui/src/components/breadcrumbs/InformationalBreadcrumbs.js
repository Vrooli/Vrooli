import React from 'react';
import { LINKS } from 'utils';
import { BreadcrumbsBase } from './BreadcrumbsBase';

const paths = [
    ['About Us', LINKS.About],
    ['Gallery', LINKS.Gallery]
]

const InformationalBreadcrumbs = ({...props}) => BreadcrumbsBase({
    paths: paths,
    ariaLabel: 'About us breadcrumb',
    ...props
})

export { InformationalBreadcrumbs };