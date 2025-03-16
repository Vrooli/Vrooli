import { GutterMarker } from "@codemirror/view";
import { ErrorIcon, WarningIcon } from "icons/common.js";
import ReactDOMServer from "react-dom/server";

export class ErrorMarker extends GutterMarker {
    toDOM() {
        const marker = document.createElement("div");
        marker.innerHTML = ReactDOMServer.renderToString(<ErrorIcon fill="red" />);
        return marker;
    }
}

export class WarnMarker extends GutterMarker {
    toDOM() {
        const marker = document.createElement("div");
        marker.innerHTML = ReactDOMServer.renderToString(<WarningIcon fill="yellow" />);
        return marker;
    }
}
