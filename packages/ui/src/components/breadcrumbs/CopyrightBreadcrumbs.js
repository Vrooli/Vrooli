import React from 'react';
import { LINKS } from 'utils';
import { BreadcrumbsBase } from './BreadcrumbsBase';

const CopyrightBreadcrumbs = ({business, ...props}) => {
    const paths = [
        [`© ${new Date().getFullYear()} ${business?.BUSINESS_NAME?.Long ?? business?.BUSINESS_NAME?.Short ?? 'Home'}`, LINKS.Home],
        ['Privacy', LINKS.PrivacyPolicy],
        ['Terms', LINKS.Terms]
    ]
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

export { CopyrightBreadcrumbs };