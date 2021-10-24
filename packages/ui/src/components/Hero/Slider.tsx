import { useState, useEffect, useRef, useMemo } from 'react';
import { SliderContent } from './SliderContent';
import { Slide } from './Slide';
import { Dots } from './Dots';
import { makeStyles } from '@material-ui/styles';

const DEFAULT_DELAY = 3000;
const DEFAULT_DURATION = 1000;

const useStyles = makeStyles({
    slider: {
        position: 'relative',
        height: '100%',
        width: '100%',
        margin: '0 auto',
        overflow: 'hidden',
        whiteSpace: 'nowrap',
    },
});

interface Props {
    images: any[];
    autoPlay?: boolean;
    slidingDelay?: number;
    slidingDuration?: number;
}

export const Slider = ({
    images = [],
    autoPlay = true,
    slidingDelay = DEFAULT_DELAY,
    slidingDuration = DEFAULT_DURATION,
}: Props) => {
    const classes = useStyles();
    const [width, setWidth] = useState(window.innerWidth);
    const [slideIndex, setSlideIndex] = useState(0);
    const [translate, setTranslate] = useState(0);
    const [transition, setTransition] = useState(0);
    const timeoutRef = useRef<NodeJS.Timeout | null>(null);

    // Play and wait have circular dependencies, so they must be memoized together
    const { wait } = useMemo(() => {
        const play = (index) => {
            if (images.length > 0) timeoutRef.current = setTimeout(wait, slidingDuration, index === images.length - 1 ? 0 : index + 1);
            setTransition(slidingDuration);
            setTranslate(width * (index + 1));
        };
        const wait = (index) => {
            setSlideIndex(index);
            if (images.length > 0) timeoutRef.current = setTimeout(play, slidingDelay, index);
            setTransition(0);
            setTranslate(width * index);
        }
        return { play, wait };
    }, [timeoutRef, images, slidingDelay, slidingDuration, width])

    useEffect(() => {
        const onResize = () => setWidth(window.innerWidth);
        window.addEventListener('resize', onResize);
        if (autoPlay) wait(0);

        return () => {
            window.removeEventListener('resize', onResize)
            if (timeoutRef.current) clearTimeout(timeoutRef.current);
        }
    }, [autoPlay, wait])

    const slides: JSX.Element[] = useMemo(() => {
        if (images?.length > 0) {
            let copy = [...images, images[0]];
            return copy.map((s, i) => (
                <Slide width={width} key={'slide-'+i} image={s} />
            ));
        } else {
            return [];
        }
    }, [width, images])

    return (
        <div className={classes.slider}>
            <SliderContent
                translate={translate}
                transition={transition}
                width={width * (slides.length)}
            >
                <>{slides}</>
            </SliderContent>
            <Dots quantity={images.length} activeIndex={slideIndex} />
        </div>
    )
}