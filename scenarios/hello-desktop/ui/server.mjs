import http from "node:http";
import { readFile } from "node:fs/promises";
import { extname, join, normalize } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("./dist/", import.meta.url));
const port = Number(process.env.UI_PORT || 23201);
const contentTypes = {
  ".css": "text/css; charset=utf-8",
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".json": "application/json; charset=utf-8",
};

const server = http.createServer(async (request, response) => {
  if (request.url === "/health") {
    response.writeHead(200, { "content-type": "application/json" });
    response.end(JSON.stringify({ status: "healthy" }));
    return;
  }

  const requested = request.url?.split("?", 1)[0] || "/";
  const relative = requested === "/" ? "index.html" : requested.slice(1);
  const path = normalize(join(root, relative));
  if (!path.startsWith(root)) {
    response.writeHead(400);
    response.end("invalid path");
    return;
  }

  try {
    const body = await readFile(path);
    response.writeHead(200, {
      "content-type": contentTypes[extname(path)] || "application/octet-stream",
    });
    response.end(body);
  } catch {
    response.writeHead(404);
    response.end("not found");
  }
});

server.listen(port, "127.0.0.1", () => {
  console.log(`hello-desktop UI listening on 127.0.0.1:${port}`);
});
