import { GutterMarker } from "@codemirror/view";
// eslint-disable-next-line import/extensions
import ReactDOMServer from "react-dom/server";
import { IconCommon } from "../../../icons/Icons.js";

export class ErrorMarker extends GutterMarker {
    toDOM() {
        const marker = document.createElement("div");
        marker.innerHTML = ReactDOMServer.renderToString(<IconCommon decorative name="Error" fill="red" />);
        return marker;
    }
}

export class WarnMarker extends GutterMarker {
    toDOM() {
        const marker = document.createElement("div");
        marker.innerHTML = ReactDOMServer.renderToString(<IconCommon decorative name="Warning" fill="yellow" />);
        return marker;
    }
}
