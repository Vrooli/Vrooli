import { fileURLToPath } from "node:url";

export default {
  test: {
    root: fileURLToPath(new URL("./src", import.meta.url)),
    include: ["**/*.test.ts"],
    environment: "jsdom",
  },
};
