chrome.commands.onCommand.addListener(async (command) => {
    if (command === "capture-screenshot") {
        const screenshotUrl = await chrome.tabs.captureVisibleTab({ format: "png" });
        await sendToApp(screenshotUrl);
    }
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.action === "CAPTURE_SCREENSHOT") {
        chrome.tabs.captureVisibleTab({ format: "png" }, async (screenshotUrl) => {
            await sendToApp(screenshotUrl);
        });
        return true;
    }
});

async function sendToApp(dataUrl: string) {
    const blob = await fetch(dataUrl).then(res => res.blob());

    try {
        await fetch("https://api.yourapp.com/upload", {
            method: "POST",
            headers: { "Content-Type": "image/png" },
            body: blob,
        });
        console.log("Screenshot uploaded successfully.");
    } catch (error) {
        console.error("Failed to upload screenshot:", error);
    }
}
