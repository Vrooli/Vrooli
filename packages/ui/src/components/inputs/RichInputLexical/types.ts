import { ElementNode, LexicalNode, TextFormatType, TextNode } from "lexical";

type GenericConstructor<T> = new (...args: any[]) => T;
export type Klass<T extends LexicalNode> = InstanceType<T['constructor']> extends T ? T['constructor'] : GenericConstructor<T> & T['constructor'];

export type Transformer = ElementTransformer | TextFormatTransformer | TextMatchTransformer;
export type ElementTransformer = {
    dependencies: Array<Klass<LexicalNode>>;
    export: (node: LexicalNode, traverseChildren: (node: ElementNode) => string) => string | null;
    regExp: RegExp;
    replace: (parentNode: ElementNode, children: Array<LexicalNode>, match: Array<string>, isImport: boolean) => void;
    type: 'element';
};
export type TextFormatTransformer = Readonly<{
    format: ReadonlyArray<TextFormatType>;
    tag: string;
    intraword?: boolean;
    type: 'text-format';
}>;
export type TextMatchTransformer = Readonly<{
    dependencies: Array<Klass<LexicalNode>>;
    export: (node: LexicalNode, exportChildren: (node: ElementNode) => string, exportFormat: (node: TextNode, textContent: string) => string) => string | null;
    importRegExp: RegExp;
    regExp: RegExp;
    replace: (node: TextNode, match: RegExpMatchArray) => void;
    trigger: string;
    type: 'text-match';
}>;