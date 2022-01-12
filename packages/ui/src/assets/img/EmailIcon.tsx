import { SvgProps } from './types';

export const EmailIcon = (props: SvgProps) => (
    <svg xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 512 512"
        style={props.style}
        aria-labelledby="email-icon"
        width={props.width}
        height={props.height}
        onClick={() => typeof props.onClick === 'function' && props.onClick()}>
        <title id="email-icon">{props.iconTitle ?? 'Email Icon'}</title>
        <path d="M339.392 258.624 512 367.744V144.896zM0 144.896v222.848l172.608-109.12zM480 80H32C16.032 80 3.36 91.904.96 107.232L256 275.264l255.04-168.032C508.64 91.904 495.968 80 480 80zM310.08 277.952l-45.28 29.824c-2.688 1.76-5.728 2.624-8.8 2.624-3.072 0-6.112-.864-8.8-2.624l-45.28-29.856L1.024 404.992C3.488 420.192 16.096 432 32 432h448c15.904 0 28.512-11.808 30.976-27.008L310.08 277.952z" fill="#fff" />
    </svg>
)