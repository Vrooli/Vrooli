export const envVariableExists = (name: string) => {
    if (!process.env[name]) {
        console.error(`❗️ ${name} not defined! Please check .env file.`);
        return false;
    }
    return true;
}