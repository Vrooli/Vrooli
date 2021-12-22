import { Meta, Story } from "@storybook/react";
import { StatsList as Component } from '../';
import { StatsListProps as Props } from '../types';

// Define story metadata
export default {
    title: 'lists/StatsList',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});