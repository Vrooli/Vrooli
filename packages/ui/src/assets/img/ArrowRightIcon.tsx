import { SvgProps } from './types';

export const ArrowRightIcon = (props: SvgProps) => (
    <svg xmlns="http://www.w3.org/2000/svg"
        style={props.style}
        viewBox="0 0 164.70816 189.38828"
        aria-labelledby="combine-node-shape"
        width={props.width}
        height={props.height}
        onClick={() => typeof props.onClick === 'function' && props.onClick()}>
        <title id="combine-node-shape-title">{props.iconTitle ?? 'Combine'}</title>
        <g
            transform="translate(-4.5438074,-4.4435488)">
            <path
                d="M 739.40623,376.9154 452.58786,542.51006 136.44012,725.03804 a 9.7764644,9.7764644 30 0 1 -14.6647,-8.46666 l 0,-331.18933 0,-365.055965 a 9.7764644,9.7764644 150 0 1 14.6647,-8.466666 L 423.25848,177.45408 739.40623,359.98206 a 9.7764644,9.7764644 90 0 1 0,16.93334 z"
                transform="matrix(0.26458333,0,0,0.26458333,-27.675939,1.6522949)"
            />
        </g>
    </svg>
)