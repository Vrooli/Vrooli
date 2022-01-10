import { APP_LINKS } from '@local/shared';
import { BreadcrumbsBase } from '../BreadcrumbsBase/BreadcrumbsBase';
import { SearchBreadcrumbsProps } from '../types';

const paths = [
    ['Organizations', APP_LINKS.SearchOrganizations],
    ['Projects', APP_LINKS.SearchProjects],
    ['Routines', APP_LINKS.SearchRoutines],
    ['Standards', APP_LINKS.SearchStandards],
    ['Users', APP_LINKS.SearchUsers],
].map(row => ({ text: row[0], link: row[1] }))

export const SearchBreadcrumbs = ({...props}: SearchBreadcrumbsProps) => BreadcrumbsBase({
    paths: paths,
    ariaLabel: 'Search pages breadcrumb',
    ...props
})