import { makeStyles } from '@material-ui/styles';

const useStyles = makeStyles({
    slider: (props: { translate: number; transition: number; width: number; }) => ({
        transform: `translateX(-${props.translate}px)`,
        transition: `transform ease-out ${props.transition}ms`,
        height: '100%',
        width: `${props.width}px`,
        display: 'flex',
    }),
});

interface Props {
    translate: number;
    transition: number;
    width: number;
    children: JSX.Element;
}

export const SliderContent = ({
    translate,
    transition,
    width,
    children
}: Props) => {
    const classes = useStyles({ translate, transition, width });
    return (
        <div className={classes.slider}>
            {children}
        </div>
    )
}