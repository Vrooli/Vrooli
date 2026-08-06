import { SearchInput } from './SearchInput';

export function SearchInputStory({ args }: StoryHarnessProps<{ placeholder: string }>) {
  return <SearchInput aria-label="Search" placeholder={args.placeholder} />;
}
