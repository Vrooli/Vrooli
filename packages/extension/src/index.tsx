// eslint-disable-next-line import/extensions
import ReactDOM from "react-dom/client";

function App() {
    function handleClick() {
        chrome.runtime.sendMessage({ type: "CAPTURE_SCREENSHOT" });
    }

    return <button onClick={handleClick}>Capture Screenshot</button>;
}

const root = document.getElementById("root");
if (root) {
    ReactDOM.createRoot(root).render(<App />);
}
