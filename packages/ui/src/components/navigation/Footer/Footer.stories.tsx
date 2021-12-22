import { Meta, Story } from "@storybook/react";
import { Footer as Component } from '../';

// Define story metadata
export default {
    title: 'navigation/Footer',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story = () => <Component />;

// Export story
export const Default = Template.bind({});