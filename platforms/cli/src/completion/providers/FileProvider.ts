import { promises as fs } from "fs";
import * as path from "path";
import type { CompletionProvider, CompletionResult, CompletionContext } from "../types.js";

export class FileProvider implements CompletionProvider {
    canHandle(context: CompletionContext): boolean {
        return context.type === "file";
    }
    
    async getCompletions(context: CompletionContext): Promise<CompletionResult[]> {
        return this.getFilePaths(context.partial);
    }
    
    private async getFilePaths(partial: string): Promise<CompletionResult[]> {
        try {
            // Handle empty partial - start from current directory
            if (!partial) {
                return this.getDirectoryContents("./");
            }
            
            // Get the directory and filename parts
            const dirname = path.dirname(partial);
            const basename = path.basename(partial);
            
            // If dirname is ".", we're completing in the current directory
            const searchDir = dirname === "." ? "./" : dirname;
            
            // Check if the directory exists
            try {
                await fs.access(searchDir);
            } catch (error) {
                // Directory doesn't exist, can't complete
                return [];
            }
            
            // Get directory contents
            const entries = await this.getDirectoryContents(searchDir);
            
            // Filter by basename
            const filtered = entries.filter(entry => 
                entry.value.startsWith(basename),
            );
            
            // If we have a directory path, prepend it to the completions
            if (dirname !== ".") {
                return filtered.map(entry => ({
                    ...entry,
                    value: path.join(dirname, entry.value),
                }));
            }
            
            return filtered;
        } catch (error) {
            // Fail silently for file completions
            return [];
        }
    }
    
    private async getDirectoryContents(dirPath: string): Promise<CompletionResult[]> {
        try {
            const entries = await fs.readdir(dirPath, { withFileTypes: true });
            const results: CompletionResult[] = [];
            
            for (const entry of entries) {
                // Skip hidden files unless specifically requested
                if (entry.name.startsWith(".")) {
                    continue;
                }
                
                const isDirectory = entry.isDirectory();
                const value = isDirectory ? entry.name + "/" : entry.name;
                
                let description = "";
                if (isDirectory) {
                    description = "Directory";
                } else {
                    // Add file extension description
                    const ext = path.extname(entry.name).toLowerCase();
                    description = this.getFileTypeDescription(ext);
                }
                
                results.push({
                    value,
                    description,
                    type: "file",
                    metadata: {
                        isDirectory,
                        extension: path.extname(entry.name),
                    },
                });
            }
            
            return results.sort((a, b) => {
                // Sort directories first, then files
                const aIsDir = a.metadata?.isDirectory as boolean;
                const bIsDir = b.metadata?.isDirectory as boolean;
                
                if (aIsDir && !bIsDir) return -1;
                if (!aIsDir && bIsDir) return 1;
                
                return a.value.localeCompare(b.value);
            });
        } catch (error) {
            return [];
        }
    }
    
    private getFileTypeDescription(extension: string): string {
        const typeMap: Record<string, string> = {
            ".json": "JSON file",
            ".js": "JavaScript file",
            ".ts": "TypeScript file",
            ".md": "Markdown file",
            ".txt": "Text file",
            ".yaml": "YAML file",
            ".yml": "YAML file",
            ".xml": "XML file",
            ".csv": "CSV file",
            ".log": "Log file",
            ".config": "Configuration file",
            ".conf": "Configuration file",
            ".env": "Environment file",
            ".sh": "Shell script",
            ".bash": "Bash script",
            ".py": "Python script",
            ".sql": "SQL file",
            ".db": "Database file",
            ".sqlite": "SQLite database",
            ".zip": "ZIP archive",
            ".tar": "TAR archive",
            ".gz": "Gzipped file",
            ".pdf": "PDF document",
            ".doc": "Word document",
            ".docx": "Word document",
            ".xls": "Excel spreadsheet",
            ".xlsx": "Excel spreadsheet",
            ".png": "PNG image",
            ".jpg": "JPEG image",
            ".jpeg": "JPEG image",
            ".gif": "GIF image",
            ".svg": "SVG image",
            ".ico": "Icon file",
            ".ttf": "TrueType font",
            ".otf": "OpenType font",
            ".woff": "Web font",
            ".woff2": "Web font",
            ".css": "CSS stylesheet",
            ".scss": "SCSS stylesheet",
            ".sass": "SASS stylesheet",
            ".less": "LESS stylesheet",
            ".html": "HTML file",
            ".htm": "HTML file",
            ".xsl": "XSL stylesheet",
            ".xsd": "XML schema",
            ".dtd": "Document type definition",
            ".rss": "RSS feed",
            ".atom": "Atom feed",
            ".lock": "Lock file",
            ".tmp": "Temporary file",
            ".temp": "Temporary file",
            ".bak": "Backup file",
            ".backup": "Backup file",
            ".cache": "Cache file",
            ".pid": "Process ID file",
            ".key": "Key file",
            ".crt": "Certificate file",
            ".pem": "PEM certificate",
            ".p12": "PKCS12 certificate",
            ".pfx": "PFX certificate",
            ".jks": "Java keystore",
            ".keystore": "Keystore file",
            ".gitignore": "Git ignore file",
            ".gitkeep": "Git keep file",
            ".gitattributes": "Git attributes file",
            ".editorconfig": "Editor configuration",
            ".eslintrc": "ESLint configuration",
            ".prettierrc": "Prettier configuration",
            ".nvmrc": "Node version file",
            ".npmrc": "NPM configuration",
            ".yarnrc": "Yarn configuration",
            ".babelrc": "Babel configuration",
            ".webpack": "Webpack configuration",
            ".rollup": "Rollup configuration",
            ".vite": "Vite configuration",
            ".tsconfig": "TypeScript configuration",
            ".jsconfig": "JavaScript configuration",
            ".dockerfile": "Docker file",
            ".dockerignore": "Docker ignore file",
            ".makefile": "Makefile",
            ".cmake": "CMake file",
            ".gradle": "Gradle file",
            ".maven": "Maven file",
            ".ant": "Ant file",
            ".sbt": "SBT file",
            ".go": "Go source file",
            ".rs": "Rust source file",
            ".c": "C source file",
            ".cpp": "C++ source file",
            ".h": "C header file",
            ".hpp": "C++ header file",
            ".java": "Java source file",
            ".class": "Java class file",
            ".jar": "Java archive",
            ".war": "Web archive",
            ".ear": "Enterprise archive",
            ".cs": "C# source file",
            ".vb": "Visual Basic file",
            ".php": "PHP file",
            ".rb": "Ruby file",
            ".gem": "Ruby gem",
            ".pl": "Perl file",
            ".pm": "Perl module",
            ".r": "R script",
            ".m": "MATLAB file",
            ".scala": "Scala file",
            ".kt": "Kotlin file",
            ".swift": "Swift file",
            ".dart": "Dart file",
            ".lua": "Lua file",
            ".ps1": "PowerShell script",
            ".bat": "Batch file",
            ".cmd": "Command file",
            ".ini": "INI file",
            ".cfg": "Configuration file",
            ".properties": "Properties file",
            ".toml": "TOML file",
            ".hocon": "HOCON file",
            ".proto": "Protocol buffer file",
            ".graphql": "GraphQL schema",
            ".gql": "GraphQL query",
            ".vue": "Vue component",
            ".jsx": "React JSX file",
            ".tsx": "React TSX file",
            ".svelte": "Svelte component",
            ".astro": "Astro component",
            ".pug": "Pug template",
            ".jade": "Jade template",
            ".hbs": "Handlebars template",
            ".mustache": "Mustache template",
            ".ejs": "EJS template",
            ".twig": "Twig template",
            ".liquid": "Liquid template",
            ".njk": "Nunjucks template",
            ".erb": "ERB template",
            ".haml": "HAML template",
            ".slim": "Slim template",
        };
        
        return typeMap[extension] || "File";
    }
}
