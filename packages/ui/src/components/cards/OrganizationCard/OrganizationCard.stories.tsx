import { Meta, Story } from "@storybook/react";
import { OrganizationCard as Component } from '../';
import { OrganizationCardProps as Props } from '../types';

// Define story metadata
export default {
    title: 'cards/OrganizationCard',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});