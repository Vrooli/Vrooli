import { Meta, Story } from "@storybook/react";
import { BreadcrumbsBase as Component } from '../';
import { BreadcrumbsBaseProps as Props } from '../types';

// Define story metadata
export default {
    title: 'breadcrumbs/BreadcrumbsBase',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});