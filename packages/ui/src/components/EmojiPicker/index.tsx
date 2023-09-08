import ErrorBoundary from "./components/ErrorBoundary";
import { PickerConfig } from "./config/config";
import EmojiPickerReact from "./EmojiPickerReact";

export { ExportedEmoji as Emoji } from "./components/emoji/ExportedEmoji";


export type Props = PickerConfig

export default function EmojiPicker(props: Props) {
    return (
        <ErrorBoundary>
            <EmojiPickerReact {...props} />
        </ErrorBoundary>
    );
}
