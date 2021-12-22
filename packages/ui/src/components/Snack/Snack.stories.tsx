import { Meta, Story } from "@storybook/react";
import { Snack as Component } from './Snack';

// Define story metadata
export default {
    title: 'Snack',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story = () => <Component />;

// Export story
export const Default = Template.bind({});