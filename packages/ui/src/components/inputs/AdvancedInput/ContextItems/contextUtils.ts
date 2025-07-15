// File regex patterns
export const IMAGE_FILE_REGEX = /\.(jpg|jpeg|png|gif|bmp|tiff|ico|webp|svg|heic|heif|ppt|pptx)$/i;
export const TEXT_FILE_REGEX = /\.(md|txt|markdown|word|doc|docx|pdf)$/i;
export const CODE_FILE_REGEX = /\.(js|jsx|ts|tsx|json|xls|xlsx|yaml|yml|xml|html|css|scss|less|py|java|c|cpp|h|hxx|cxx|hpp|hxx|rb|php|go|swift|kotlin|scala|groovy|rust|haskell|erlang|elixir|dart|typescript|kotlin|swift|ruby|php|go|rust|haskell|erlang|elixir|dart|typescript)$/i;
export const VIDEO_FILE_REGEX = /\.(mp4|mov|avi|wmv|flv|mpeg|mpg|m4v|webm|mkv)$/i;
export const ENV_FILE_REGEX = /\.(env|env-example|env-local|env-production|env-development|env-test)$/i;
export const EXECUTABLE_FILE_REGEX = /\.(exe|bat|sh|bash|cmd|ps1|ps2|ps3|ps4|ps5|ps6|ps7|ps8|ps9|ps10)$/i;

export const MAX_LABEL_LENGTH = 20;
export const CONTEXT_ITEM_LIMIT = 20;

export function truncateLabel(label: string, maxLength: number): string {
    if (label.length <= maxLength) return label;
    const extension = label.split(".").pop();
    const nameWithoutExt = label.slice(0, label.lastIndexOf("."));
    if (!extension || nameWithoutExt.length <= maxLength - 4) return label;
    return `${nameWithoutExt.slice(0, maxLength - 4)}...${extension}`;
}
