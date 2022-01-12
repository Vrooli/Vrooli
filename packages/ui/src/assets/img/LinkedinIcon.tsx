import { SvgProps } from './types';

export const LinkedinIcon = (props: SvgProps) => (
    <svg xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 144 144"
        style={props.style}
        aria-labelledby="linkedin-icon"
        width={props.width}
        height={props.height}
        onClick={() => typeof props.onClick === 'function' && props.onClick()}>
        <title id="linkedin-icon">{props.iconTitle ?? 'LinkedIn Icon'}</title>
        <g>
            <g id="Layer_1-2">
                <path fill="white" d="M133.3,0H10.6C4.8-0.1,0.1,4.6,0,10.4v123.2c0.1,5.8,4.8,10.5,10.6,10.4h122.7c5.8,0.1,10.6-4.6,10.7-10.4    V10.4C143.9,4.6,139.1-0.1,133.3,0z M42.7,122.7H21.4V54h21.4V122.7z M32,44.6c-6.8,0-12.4-5.6-12.3-12.4s5.6-12.4,12.4-12.3    c6.8,0,12.3,5.6,12.4,12.4C44.4,39.1,38.9,44.6,32,44.6C32,44.6,32,44.6,32,44.6z M122.7,122.7h-21.3V89.3c0-8-0.1-18.2-11.1-18.2    c-11.1,0-12.8,8.7-12.8,17.6v34H56.1V54h20.5v9.4h0.3c2.8-5.4,9.8-11.1,20.2-11.1c21.6,0,25.6,14.2,25.6,32.7L122.7,122.7z" />
            </g>
        </g>
    </svg>
)