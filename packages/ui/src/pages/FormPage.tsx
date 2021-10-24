import { Container, Theme, Typography } from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';

const useStyles = makeStyles((theme: Theme) => ({
    formHeader: {
        backgroundColor: theme.palette.primary.main,
        color: theme.palette.primary.contrastText,
        padding: '1em',
        textAlign: 'center'
    },
    container: {
        backgroundColor: theme.palette.background.paper,
        display: 'grid',
        position: 'relative',
        boxShadow: '0px 2px 4px -1px rgb(0 0 0 / 20%), 0px 4px 5px 0px rgb(0 0 0 / 14%), 0px 1px 10px 0px rgb(0 0 0 / 12%)',
        minWidth: '300px',
        maxWidth: 'min(100%, 700px)',
        borderRadius: '10px',
        overflow: 'hidden',
        left: '50%',
        transform: 'translateX(-50%)',
        marginBottom: '20px'
    },
    [theme.breakpoints.down("sm")]: {
        page: {
            padding: '0',
            paddingTop: 'calc(14vh + 20px)',
        }
      },
}));

interface Props {
    title: string;
    autocomplete?: string;
    children: JSX.Element;
    maxWidth?: string | number;
}

export const FormPage = ({
    title,
    autocomplete = 'on',
    children,
    maxWidth = '90%',
}: Props) => {
    const classes = useStyles();

    return (
        <div id='page' style={{maxWidth: maxWidth}}>
            <div className={classes.container}>
                <Container className={classes.formHeader}>
                    <Typography variant="h3" >{title}</Typography>
                </Container>
                <Container>
                    {children}
                </Container>
            </div>
        </div>
    );
}