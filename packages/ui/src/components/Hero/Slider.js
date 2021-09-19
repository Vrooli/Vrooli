import React, { useState, useEffect, useRef, useMemo } from 'react';
import { SliderContent } from './SliderContent';
import { Slide } from './Slide';
import { Dots } from './Dots';
import { makeStyles } from '@material-ui/styles';

const DEFAULT_DELAY = 3000;
const DEFAULT_DURATION = 1000;

const useStyles = makeStyles({
    slider: {
        position: 'relative',
        height: '100vh',
        width: '100vw',
        margin: '0 auto',
        overflow: 'hidden',
        whiteSpace: 'nowrap',
    },
});

const Slider = ({
    images = [],
    autoPlay = true,
    slidingDelay = DEFAULT_DELAY,
    slidingDuration = DEFAULT_DURATION,
}) => {
    const classes = useStyles();
    const [width, setWidth] = useState(window.innerWidth);
    const [slideIndex, setSlideIndex] = useState(0);
    const [translate, setTranslate] = useState(0);
    const [transition, setTransition] = useState(0);
    const sliderRef = useRef()
    const timeoutRef = useRef(null);

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
        const onResize = window.addEventListener('resize', () => setWidth(window.innerWidth))
        if (autoPlay) wait(0);

        return () => {
            window.removeEventListener('resize', onResize)
            clearTimeout(timeoutRef.current);
        }
    }, [autoPlay, wait])

    const slides = useMemo(() => {
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
        <div className={classes.slider} ref={sliderRef}>
            <SliderContent
                translate={translate}
                transition={transition}
                width={width * (slides?.length ?? 0)}
            >
                {slides}
            </SliderContent>
            <Dots quantity={images.length} activeIndex={slideIndex} />
        </div>
    )
}

export { Slider };