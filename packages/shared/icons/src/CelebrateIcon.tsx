import { SvgBase } from './base';
import { SvgProps } from './types';

export const CelebrateIcon = (props: SvgProps) => (
    <SvgBase props={props}>
        <path
            style={{
                fillOpacity: 1,
                fill: props.fill ?? 'white',
            }}
            d="M6.85 7.32c-.6.15-.6.85-.82 1.3-1.57 4.42-3.16 8.83-4.72 13.25-.14.6.6 1.05 1.09.73 4.64-1.66 9.29-3.3 13.92-4.98.6-.3.4-1.06-.07-1.36L7.43 7.45a.7.7 0 0 0-.58-.13ZM2 22l14-5-9-9-5 14ZM18.6 7.3c-.77-.03-1.45.4-1.96.94-1.22 1.17-1.4 1.36-2.61 2.54-.25.36.15.65.39.87.25.2.48.62.86.48.44-.32.8-.73 1.2-1.1.97-.91.89-.9 1.87-1.8.39-.24.68.18.94.41.26.31.64.08.85-.17.21-.24.53-.41.65-.73.06-.37-.33-.55-.54-.79a2.4 2.4 0 0 0-1.65-.65Zm-8.3-3.88c-.44.1-.68.54-1 .82-.47.51.18.84.41 1.2.1.48-.58.6-.58 1.07.11.46.54.7.82 1.04.43.4.8-.1 1.09-.38.92-.9.84-2.53-.13-3.35-.18-.16-.33-.4-.6-.4ZM10 6.13c-.07.17-.5.33-.26.5l.57.57c.46-.43.91-.96.88-1.64.03-.67-.42-1.2-.88-1.64l-.68.7c.32.3.74.67.57 1.17a.8.8 0 0 1-.2.34Zm9 5.13c-.83-.03-1.55.43-2.13.98-.28.27-.66.47-.85.83-.08.52.48.73.78 1.05.34.4.83.12 1.12-.19.3-.23.57-.54.91-.73.46-.12.73.35 1.07.58.35.26.32.53.8.14.3-.28.69-.48.88-.85.11-.5-.03-.42-.32-.71-.58-.52-1.22-1.06-2.04-1.09H19zm-1.38 1-1.12.95.75.63c.45-.35.85-.77 1.34-1.08.48-.24.98.02 1.32.37l.85.73.75-.64c-.54-.42-1-.94-1.61-1.26a2.2 2.2 0 0 0-2.28.3zm-3.2-10.37c-.42.05-.62.5-.92.77-.28.23-.5.67-.16.96.38.43.79.84 1.15 1.29.26.45-.18.8-.45 1.1-.75.83-1.52 1.65-2.25 2.49-.26.44.24.76.48 1.07.24.25.5.67.9.5.45-.36.79-.83 1.19-1.24.6-.69 1.26-1.34 1.83-2.06a2.88 2.88 0 0 0-.44-3.68c-.38-.38-.7-.83-1.12-1.16l-.1-.03zm.4 4.01-2.6 2.86.76.84c.9-1 1.84-2 2.73-3.02a2.4 2.4 0 0 0-.33-3.15l-.95-1.04-.77.84c.42.48.87.92 1.27 1.42.22.39.19.92-.12 1.25z"
            transform="translate(.54 -.3)"
        />
    </SvgBase>
)