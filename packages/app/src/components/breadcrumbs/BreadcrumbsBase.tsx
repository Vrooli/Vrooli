import { 
    Breadcrumbs, 
    Link 
} from '@material-ui/core';
import { useHistory } from 'react-router-dom';
import { makeStyles } from '@material-ui/styles';
import merge from 'lodash/merge';
import { BreadcrumbsBaseProps } from './types';
import { useMemo } from 'react';

const useStyles = makeStyles(() => ({
    root: {
        cursor: 'pointer',
    },
    li: {
        minHeight: '48px', // Lighthouse recommends this for SEO, as it is more clickable
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'center',
    },
}))

const BreadcrumbsBase = ({
    paths,
    separator = '|',
    ariaLabel = 'breadcrumb',
    textColor = 'textPrimary',
    style,
    className
}: BreadcrumbsBaseProps) => {
    const classes = useStyles();
    const history = useHistory();
    // Add user styling to default root style
    let rootStyle = merge(classes.root, style ?? {});
    // Match separator color to link color, if not specified
    // @ts-expect-error
    if (textColor) rootStyle.color = textColor;

    const pathLinks = useMemo(() => (
        paths.map(p => (
            <Link 
                key={p.text}
                color={textColor}
                onClick={() => history.push(p.link)}
            >
                {window.location.pathname === p.link ? <b>{p.text}</b> : p.text}
            </Link>
        ))
    ), [history, paths, textColor])

    return (
            <Breadcrumbs className={className} style={style ?? {}} classes={{root: classes.root, li: classes.li}} separator={separator} aria-label={ariaLabel}>
                {pathLinks}
            </Breadcrumbs>
    );
}

export { BreadcrumbsBase };