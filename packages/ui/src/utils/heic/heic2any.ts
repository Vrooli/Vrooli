
// AI_CHECK: TYPE_SAFETY=improved-heic-converter-types | LAST: 2025-06-28

const supportedMIMETypes = ["image/png", "image/jpeg", "image/gif"];

const utils = {
    blobToDataURL(blob: Blob): Promise<string> {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onerror = function () {
                reject("ERR_DOM Error on converting blob to data URL");
            };
            reader.onload = (e) => {
                const result = reader.result;
                if (typeof result === "string") {
                    resolve(result);
                } else {
                    reject("ERR_DOM Expected string result from FileReader");
                }
            };
            reader.readAsDataURL(blob);
        });
    },

    dataURItoBlob(dataURI: string): Blob | string {
        try {
            const byteString = atob(dataURI.split(",")[1]);
            const mimeString = dataURI.split(",")[0].split(":")[1].split(";")[0];
            const ab = new ArrayBuffer(byteString.length);
            const ia = new Uint8Array(ab);
            for (let i = 0; i < byteString.length; i++) {
                ia[i] = byteString.charCodeAt(i);
            }
            const blob = new Blob([ab], { type: mimeString });
            return blob;
        } catch (e: unknown) {
            const errorMessage = e instanceof Error ? e.message : String(e);
            return "ERR_DOM Error on converting data URI to blob " + errorMessage;
        }
    },

    imageDataToBlob({
        imageData,
        toType = "image/png",
        quality = 0.92,
    }: {
        imageData: ImageData;
        toType?: string;
        quality?: number;
    }): Promise<Blob> {
        // normalize quality
        if (quality > 1 || quality < 0) {
            quality = 0.92;
        }
        // normalize MIME type
        if (supportedMIMETypes.indexOf(toType) === -1) {
            toType = "image/png";
        }
        return new Promise((resolve, reject) => {
            let canvas: HTMLCanvasElement | null = null;

            try {
                canvas = document.createElement("canvas");
            } catch (e) { /* empty */ }

            if (!canvas) {
                return reject(
                    "ERR_CANVAS Error on converting imagedata to blob: Could not create canvas element",
                );
            }

            canvas.width = imageData.width;
            canvas.height = imageData.height;
            const ctx = canvas.getContext("2d");
            if (!ctx) {
                return reject(
                    "ERR_CANVAS Error on converting imagedata to blob: Could not get canvas context",
                );
            }
            ctx.putImageData(imageData, 0, 0);
            canvas.toBlob(
                (blob) => {
                    if (!blob) {
                        return reject(
                            "ERR_CANVAS Error on converting imagedata to blob: Could not get blob from canvas",
                        );
                    }
                    return resolve(blob);
                },
                toType,
                quality,
            );
        });
    },

    otherImageType(buffer: ArrayBuffer) {
        /**
         * Some confusion might arise when passing a regular image
         * like jpeg/png/gif and getting "format is not supported"
         * so to solve this, we should detect if the image is
         * already browser-readable.
         */
        const arr = new Uint8Array(buffer).subarray(0, 4);
        let header = "";
        for (let i = 0; i < arr.length; i++) {
            header = header + arr[i].toString(16);
        }
        switch (header) {
            case "89504e47":
                return "image/png";
            case "47494638":
                return "image/gif";
            case "ffd8ffe0":
            case "ffd8ffe1":
            case "ffd8ffe2":
            case "ffd8ffe3":
            case "ffd8ffe8":
                return "image/jpeg";
            default:
                return false;
        }
    },

    error(message: string) {
        /**
         * Error coding system:
         *
         * UNKNOWN = 0
         * USER errors = 1
         * LIBHEIF errors = 2
         * GIF errors = 3
         * DOM errors = 4
         * CANVAS errors = 5
         *
         */

        let code = 0;

        if (!message) {
            message = "ERR_UNKNOWN";
        } else if (typeof message !== "string") {
            if (typeof message === "object" && message !== null && "toString" in message && typeof message.toString === "function") {
                message = message.toString();
            } else {
                message = JSON.stringify(message);
            }
        }

        const headers = [
            "ERR_USER",
            "ERR_LIBHEIF",
            "ERR_GIF",
            "ERR_DOM",
            "ERR_CANVAS",
        ];
        for (let index = 0; index < headers.length; index++) {
            const header = headers[index];
            if (message.indexOf(header) === 0) {
                code = index + 1;
            }
        }
        return {
            code,
            message,
        };
    },
};

async function decodeBuffer(buffer: ArrayBuffer): Promise<ImageData[]> {
    // Load libheif dynamically
    const globalWindow = window as Window & { __heic2any__worker?: Worker }; // Global worker cache with proper typing
    if (!globalWindow.__heic2any__worker) {
        console.info("Loading libheif.js");
        globalWindow.__heic2any__worker = new Worker("/pkg/libheif/worker.js");
        if (!globalWindow.__heic2any__worker) {
            throw new Error("Failed to load libheif.js");
        }
    }

    console.log("libheif.js loaded", globalWindow.__heic2any__worker);
    return new Promise((resolve, reject) => {
        const id = (Math.random() * new Date().getTime()).toString();
        const message = { id, buffer };
        const worker = globalWindow.__heic2any__worker as Worker;
        worker.postMessage(message);
        worker.addEventListener(
            "message",
            (message) => {
                if (message.data.id === id) {
                    if (message.data.error) {
                        return reject(message.data.error);
                    }
                    return resolve(message.data.imageDataArr);
                }
            },
        );
    });
}

function heic2any({
    blob,
    toType = "image/png",
    quality = 0.92,
    gifInterval = 0.4,
    multiple = undefined,
}: {
    blob: Blob;
    multiple?: true;
    toType?: string;
    quality?: number;
    gifInterval?: number;
}): Promise<Blob | Blob[]> {
    return new Promise(
        (
            resolve,
            reject: (reason: { code: number; message: string }) => void,
        ) => {
            if (!(blob instanceof Blob)) {
                utils.error("ERR_USER library only accepts BLOBs as input");
            }
            if (typeof multiple !== "boolean") {
                utils.error(
                    "ERR_USER \"multiple\" parameter should be of type \"boolean\"",
                );
            }
            if (typeof quality !== "number") {
                utils.error(
                    "ERR_USER \"quality\" parameter should be of type \"number\"",
                );
            }
            if (typeof gifInterval !== "number") {
                utils.error(
                    "ERR_USER \"gifInterval\" parameter should be of type \"number\"",
                );
            }
            const reader = new FileReader();
            reader.onload = (e) => {
                const target = e.target as FileReader;
                const buffer = target.result as ArrayBuffer;
                const otherImageType = utils.otherImageType(buffer);
                if (otherImageType) {
                    return reject(
                        utils.error(
                            `ERR_USER Image is already browser readable: ${otherImageType}`,
                        ),
                    );
                }
                decodeBuffer(buffer)
                    .then((imageDataArr) => {
                        return Promise.all(
                            imageDataArr.map((imageData) =>
                                utils.imageDataToBlob({
                                    imageData,
                                    toType,
                                    quality,
                                }),
                            ),
                        );
                    })
                    .then((blobs) => {
                        if (toType === "image/gif") {
                            return Promise.all(
                                blobs.map((blob) => utils.blobToDataURL(blob)),
                            );
                        } else if (multiple) {
                            resolve(blobs);
                            return [""];
                        } else {
                            resolve(blobs[0]);
                            return [""];
                        }
                    })
                    .then((dataURIs) => {
                        return "";
                    })
                    .then((resultingGif) => {
                        if (toType === "image/gif" && resultingGif) {
                            const blob = utils.dataURItoBlob(resultingGif);
                            if (typeof blob === "string") {
                                reject(utils.error(blob));
                            } else {
                                resolve(blob);
                            }
                        }
                    })
                    .catch((e) => {
                        reject(utils.error(e));
                    });
            };
            reader.readAsArrayBuffer(blob);
        },
    );
}

export default heic2any;
