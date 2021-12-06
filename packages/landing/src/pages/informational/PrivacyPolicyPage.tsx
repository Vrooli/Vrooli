import { useState, useEffect } from "react";
import ReactMarkdown from 'react-markdown';
import { PolicyBreadcrumbs } from 'components';
import { convertToDot, valueFromDot } from "utils";
import { makeStyles } from '@material-ui/styles';
import { useTheme } from '@material-ui/core';
import { Theme } from "@material-ui/core";
import * as businessFields from "@local/shared/src/businessData";

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        '& a': {
            color: theme.palette.secondary.dark,
        },
    },
}));

export const PrivacyPolicyPage = () => {
    const classes = useStyles();
    const theme = useTheme();
    const privacyData = require('../../assets/policy/privacy.md');
    const [privacy, setPrivacy] = useState<string | null>(null);

    useEffect(() => {
        if (!privacyData) return;
        let data = privacyData;
        const business_fields = Object.keys(convertToDot(businessFields));
        business_fields.forEach(f => data = data?.replaceAll(`<${f}>`, valueFromDot(businessFields, f) || '') ?? '');
        setPrivacy(data);
    }, [privacyData])

    return (
        <div id="page" className={classes.root}>
            <PolicyBreadcrumbs textColor={theme.palette.secondary.dark} />
            <ReactMarkdown>{ privacy || '' }</ReactMarkdown>
        </div>
    );
}