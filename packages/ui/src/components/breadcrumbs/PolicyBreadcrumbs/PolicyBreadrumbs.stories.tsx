import { Meta, Story } from "@storybook/react";
import { PolicyBreadcrumbs as Component } from '../';
import { PolicyBreadcrumbsProps as Props } from '../types';

// Define story metadata
export default {
    title: 'breadcrumbs/PolicyBreadcrumbs',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});