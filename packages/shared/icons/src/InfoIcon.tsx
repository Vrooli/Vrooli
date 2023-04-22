import { SvgPath } from "./base";
import { SvgProps } from "./types";

export const InfoIcon = (props: SvgProps) => (
    <SvgPath
        props={props}
        d="M12 2.486c-5.246 0-9.514 4.268-9.514 9.514 0 5.246 4.268 9.514 9.514 9.514 5.246 0 9.514-4.268 9.514-9.514 0-5.246-4.268-9.514-9.514-9.514Zm0 1.381c4.5 0 8.133 3.633 8.133 8.133 0 4.5-3.633 8.133-8.133 8.133A8.122 8.122 0 0 1 3.867 12C3.867 7.5 7.5 3.867 12 3.867Zm-.02 2.153c-.746 0-1.37.624-1.37 1.37 0 .747.624 1.37 1.37 1.37.747 0 1.37-.623 1.37-1.37 0-.746-.623-1.37-1.37-1.37zm.02 4.36a1.068.953 0 0 0-1.068.954v6.287a1.068.953 0 0 0 1.068.953 1.068.953 0 0 0 1.068-.953v-6.287A1.068.953 0 0 0 12 10.381Z"
    />
);
