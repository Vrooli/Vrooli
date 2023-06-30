import fs from "fs";
import path from "path";

export const loadSecrets = (env: "development" | "production") => {
    // Set the secrets path based on the environment
    const secretsPath = `/run/secrets/vrooli/${env === "development" ? "dev" : "prod"}`;

    // Read all files in the secrets directory
    const files = fs.readdirSync(secretsPath);

    // For each file, read the content and add it to process.env
    for (const file of files) {
        const filePath = path.join(secretsPath, file);
        const secretValue = fs.readFileSync(filePath, "utf-8");
        process.env[file] = secretValue.trim();
    }
};
