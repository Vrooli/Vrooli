import { Meta, Story } from "@storybook/react";
import { Navbar as Component } from '../';
import { NavbarProps as Props } from '../types';

// Define story metadata
export default {
    title: 'navigation/Navbar',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});