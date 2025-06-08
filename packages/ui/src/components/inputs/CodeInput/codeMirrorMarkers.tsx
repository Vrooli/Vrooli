// eslint-disable-next-line import/extensions
import ReactDOMServer from "react-dom/server";
import { IconCommon } from "../../../icons/Icons.js";

// Dynamically create marker classes to avoid static imports
export async function createMarkers() {
    const { GutterMarker } = await import("@codemirror/view");
    
    class ErrorMarker extends GutterMarker {
        toDOM() {
            const marker = document.createElement("div");
            marker.innerHTML = ReactDOMServer.renderToString(<IconCommon decorative name="Error" fill="red" />);
            return marker;
        }
    }

    class WarnMarker extends GutterMarker {
        toDOM() {
            const marker = document.createElement("div");
            marker.innerHTML = ReactDOMServer.renderToString(<IconCommon decorative name="Warning" fill="yellow" />);
            return marker;
        }
    }
    
    return { ErrorMarker, WarnMarker };
}
