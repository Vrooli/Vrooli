/**
 * @libraryId react-component-library:SearchInput
 * @displayName SearchInput
 * @description
 * @version 1.0.6
 * @tags ["form","control","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:SearchInput */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { forwardRef, type InputHTMLAttributes } from "react";
import "./SearchInput.css";
export const SearchInput = forwardRef<
  HTMLInputElement,
  InputHTMLAttributes<HTMLInputElement>
>(function SearchInput({ placeholder, style, ...props }, ref) {
  const libraryStrings = useStrings();
  placeholder =
    placeholder ?? libraryStrings("forms.search-input.search", "Search");
  const strings = useStrings();
  return (
    <label data-rcl-search-input>
      <span data-rcl-search-input-label>
        {strings("forms.search-input.search", "Search")}
      </span>
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
});
