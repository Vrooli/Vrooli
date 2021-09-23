import React from 'react';
import { InformationalBreadcrumbs } from 'components';
import {
    Grid,
    Typography
} from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { useTheme } from '@emotion/react';
import { pageStyles } from './styles';
import { combineStyles } from 'utils';

const componentStyles = (theme) => ({
    social: {
        width: '80px',
        height: '80px',
    }
})

const useStyles = makeStyles(combineStyles(pageStyles, componentStyles));

function AboutPage({
    business,
}) {
    const classes = useStyles();
    const theme = useTheme();

    return (
        <div id='page'>
        </div >
    );
}

AboutPage.propTypes = {
}

export { AboutPage };