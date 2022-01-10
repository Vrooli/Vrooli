import {
    Breadcrumbs,
    Link
} from '@mui/material';
import { BreadcrumbsBaseProps } from '../types';
import { useMemo } from 'react';
import { useLocation } from 'wouter';
import { openLink } from 'utils';

const BreadcrumbsBase = ({
    paths,
    separator = '|',
    ariaLabel = 'breadcrumb',
    textColor,
    sx,
}: BreadcrumbsBaseProps) => {
    const [, setLocation] = useLocation();

    const pathLinks = useMemo(() => (
        paths.map(p => (
            <Link
                key={p.text}
                color={textColor}
                onClick={() => openLink(setLocation, p.link)}
            >
                {window.location.pathname === p.link ? <b>{p.text}</b> : p.text}
            </Link>
        ))
    ), [setLocation, paths, textColor])

    return (
        <Breadcrumbs
            sx={{
                ...sx,
                '& .MuiBreadcrumbs-root': {
                    cursor: 'pointer'
                },
                '& .MuiBreadcrumbs-li > a': {
                    color: sx?.color || 'inherit',
                    minHeight: '48px', // Lighthouse recommends this for SEO, as it is more clickable
                    display: 'flex',
                    flexDirection: 'row',
                    alignItems: 'center',
                }
            }}
            separator={separator}
            aria-label={ariaLabel}
        >
            {pathLinks}
        </Breadcrumbs>
    );
}

export { BreadcrumbsBase };