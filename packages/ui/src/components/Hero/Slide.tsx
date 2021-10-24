import { memo } from 'react'
import { makeStyles } from '@material-ui/styles';
import { SERVER_URL } from '@local/shared';
import { getImageSrc } from 'utils';

const useStyles = makeStyles({
    slide: ({ width }: { width: number}) => ({
        height: '100%',
        width: `${width}px`,
        objectFit: 'cover',
        overflow: 'hidden',
    }),
});

interface Props {
    image: any;
    width: number;
}

const Slide = memo(({ image, width }: Props): JSX.Element => {
    const classes = useStyles({ width });
    return (
        <img className={classes.slide} src={image ? `${SERVER_URL}/${getImageSrc(image, width)}` : ''} alt={image?.alt ?? ''} />
    )
})

export { Slide };