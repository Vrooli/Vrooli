/**
 * @libraryId react-component-library:SearchInput
 * @displayName SearchInput
 * @description A token-bound search field with a stable touch target and accessible label for progressive filtering.
 * @version 1.0.4
 * @tags ["form","control","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:SearchInput */
import { resolveStrings, useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { forwardRef, type InputHTMLAttributes } from "react";
import "./SearchInput.css";
export const SearchInput = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement>>(
  function SearchInput(
    { placeholder = resolveStrings("forms.search-input.search", "Search"), style, ...props },
    ref,
  ) {
    const strings = useStrings();
    return (
      <label data-rcl-search-input>
        <span data-rcl-search-input-label>{strings("forms.search-input.search", "Search")}</span>
        <input
          data-testid="forms.search-input"
          ref={ref}
          type="search"
          placeholder={placeholder}
          data-rcl-search-input-field
          style={style}
          {...props}
        />
      </label>
    );
  },
);
