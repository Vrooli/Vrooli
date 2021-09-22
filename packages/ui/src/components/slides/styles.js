export const slideStyles = (theme) => ({
    slideRoot: {
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        minHeight: "100vh",
    },
    slideBackground: {
        position: 'absolute',
        width: '100vw',
        height: '100vh',
        objectFit: 'cover',
    },
    slidePad: {
        padding: theme.spacing(2),
        zIndex: 1,
    },
    titleCenter: {
        textAlign: 'center',
        paddingLeft: '5vw',
        paddingRight: '5vw',
    },
    buttonCenter: {
        display: 'flex',
        margin: 'auto',
    },
    textPop: {
        padding: '0',
        color: 'white',
        textAlign: 'center',
        fontWeight: '600',
        textShadow:
            `-1px -1px 0 black,  
                1px -1px 0 black,
                -1px 1px 0 black,
                1px 1px 0 black`
    }
});