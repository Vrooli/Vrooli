import React from 'react';
import PropTypes from 'prop-types';
import { 
    Breadcrumbs, 
    Link 
} from '@material-ui/core';
import { useHistory } from 'react-router-dom';
import { makeStyles } from '@material-ui/styles';
import _ from 'lodash';

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

// Breadcrumbs reload all components if using href directly. Not sure why
function BreadcrumbsBase({
    paths,
    separator = '|',
    ariaLabel = 'breadcrumb',
    textColor = 'textPrimary',
    style,
    ...props
}) {
    const classes = useStyles();
    const history = useHistory();
    // Add user styling to default root style
    let rootStyle = _.merge(classes.root, style ?? {});
    // Match separator color to link color, if not specified
    if (textColor && !rootStyle.color) rootStyle.color = textColor;
    return (
            <Breadcrumbs style={style ?? {}} classes={{root: classes.root, li: classes.li}} separator={separator} aria-label={ariaLabel} {...props}>
                {paths.map(p => (
                    <Link 
                        key={p[0]}
                        color={textColor}
                        onClick={() => history.push(p[1])}
                    >
                        {window.location.pathname === p[1] ? <b>{p[0]}</b> : p[0]}
                    </Link>
                ))}
            </Breadcrumbs>
    );
}

BreadcrumbsBase.propTypes = {
    paths: PropTypes.array.isRequired,
    separator: PropTypes.string,
    ariaLabel: PropTypes.string,
    textColor: PropTypes.string,
}

export { BreadcrumbsBase };