export const formStyles = (theme) => ({
    form: {
        width: '100%',
        marginTop: theme.spacing(3),
    },
    submit: {
        margin: theme.spacing(3, 0, 2),
    },
    linkRight: {
        flexDirection: 'row-reverse',
    },
    clickSize: {
        color: theme.palette.secondary.dark,
        minHeight: '48px', // Lighthouse recommends this for SEO, as it is more clickable
        display: 'flex',
        alignItems: 'center',
    },
});