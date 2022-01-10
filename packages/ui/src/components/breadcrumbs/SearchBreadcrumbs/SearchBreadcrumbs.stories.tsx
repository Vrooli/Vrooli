import { Meta, Story } from "@storybook/react";
import { SearchBreadcrumbs as Component } from '..';
import { SearchBreadcrumbsProps as Props } from '../types';

// Define story metadata
export default {
    title: 'breadcrumbs/SearchBreadcrumbs',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});