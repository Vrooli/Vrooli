import { ClassNames } from "../../DomUtils/classNames";
import { useOnMouseMove } from "../../hooks/useDisallowMouseMove";
import { useMouseDownHandlers } from "../../hooks/useMouseDownHandlers";
import { useOnScroll } from "../../hooks/useOnScroll";
import { useBodyRef } from "../context/ElementRefContext";
import "./Body.css";
import { EmojiList } from "./EmojiList";
import { EmojiVariationPicker } from "./EmojiVariationPicker";

export function Body() {
    const BodyRef = useBodyRef();
    useOnScroll(BodyRef);
    useMouseDownHandlers(BodyRef);
    useOnMouseMove();

    return (
        <div className={ClassNames.scrollBody} ref={BodyRef}>
            <EmojiVariationPicker />
            <EmojiList />
        </div>
    );
}
